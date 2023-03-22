//go:build integration
// +build integration

package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/labstack/echo/v4"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func trialNotFoundErrEcho(id int) error {
	return echo.NewHTTPError(http.StatusNotFound, "trial not found: %d", id)
}

func createTestTrialWithMetrics(
	ctx context.Context, t *testing.T, api *apiServer, curUser model.User, includeBatchMetrics bool,
) (*model.Trial, []*commonv1.Metrics, []*commonv1.Metrics) {
	var trainingMetrics, validationMetrics []*commonv1.Metrics
	trial := createTestTrial(t, api, curUser)

	for i := 0; i < 10; i++ {
		trainMetrics := &commonv1.Metrics{
			AvgMetrics: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"loss": {
						Kind: &structpb.Value_NumberValue{
							NumberValue: float64(i),
						},
					},
				},
			},
		}
		if includeBatchMetrics {
			trainMetrics.BatchMetrics = []*structpb.Struct{
				{
					Fields: map[string]*structpb.Value{
						"batch_loss": {
							Kind: &structpb.Value_NumberValue{
								NumberValue: float64(i),
							},
						},
					},
				},
			}
		}

		_, err := api.ReportTrialTrainingMetrics(ctx,
			&apiv1.ReportTrialTrainingMetricsRequest{
				TrainingMetrics: &trialv1.TrialMetrics{
					TrialId:        int32(trial.ID),
					TrialRunId:     0,
					StepsCompleted: int32(i),
					Metrics:        trainMetrics,
				},
			})
		require.NoError(t, err)
		trainingMetrics = append(trainingMetrics, trainMetrics)

		valMetrics := &commonv1.Metrics{
			AvgMetrics: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"val_loss": {
						Kind: &structpb.Value_NumberValue{
							NumberValue: float64(i),
						},
					},
				},
			},
		}
		_, err = api.ReportTrialValidationMetrics(ctx,
			&apiv1.ReportTrialValidationMetricsRequest{
				ValidationMetrics: &trialv1.TrialMetrics{
					TrialId:        int32(trial.ID),
					TrialRunId:     0,
					StepsCompleted: int32(i),
					Metrics:        valMetrics,
				},
			})
		require.NoError(t, err)
		validationMetrics = append(validationMetrics, valMetrics)
	}

	return trial, trainingMetrics, validationMetrics
}

func compareMetrics(
	t *testing.T, trialID int, resp *httptest.ResponseRecorder, expected []*commonv1.Metrics,
) {
	defer resp.Result().Body.Close()
	b, err := io.ReadAll(resp.Result().Body)
	require.NoError(t, err)

	split := strings.Split(string(b), "\n")
	require.Equal(t, len(split)-1, len(expected)) // Extra newline in split.
	require.Equal(t, split[len(split)-1], "")

	for i, jsonString := range split[:len(split)-1] {
		var actual map[string]any
		require.NoError(t, json.Unmarshal([]byte(jsonString), &actual))

		metrics := map[string]any{
			"avg_metrics":   expected[i].AvgMetrics.AsMap(),
			"batch_metrics": nil, // map[string]any{},
		}
		if expected[i].BatchMetrics != nil {
			var batchMetrics []any
			for _, b := range expected[i].BatchMetrics {
				batchMetrics = append(batchMetrics, b.AsMap())
			}
			metrics["batch_metrics"] = batchMetrics
		}

		require.Equal(t, map[string]any{
			"trialId":      float64(trialID),
			"endTime":      actual["endTime"],
			"metrics":      metrics,
			"totalBatches": float64(i),
			"trialRunId":   float64(0),
			"id":           actual["id"],
		}, actual)
	}
}

func TestStreamTrainingMetrics(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	var trials []*model.Trial
	var trainingMetrics, validationMetrics [][]*commonv1.Metrics
	for _, haveBatchMetrics := range []bool{false, true} {
		trial, trainMetrics, valMetrics := createTestTrialWithMetrics(
			ctx, t, api, curUser, haveBatchMetrics)
		trials = append(trials, trial)
		trainingMetrics = append(trainingMetrics, trainMetrics)
		validationMetrics = append(validationMetrics, valMetrics)
	}

	endpoint := "/trials/metrics/training"
	requestFunc := api.m.streamTrainingMetrics

	// No trial IDs.
	c := newTestEchoContext(curUser)
	c.SetRequest(httptest.NewRequest(http.MethodGet, endpoint, nil))
	err := requestFunc(c)
	require.Error(t, err)
	require.Equal(t, err.(*echo.HTTPError).Code, http.StatusBadRequest)

	// Not parsable trial IDs.
	c = newTestEchoContext(curUser)
	c.SetRequest(httptest.NewRequest(http.MethodGet, endpoint+"?trialIds=x", nil))
	err = requestFunc(c)
	require.Error(t, err)
	require.Equal(t, err.(*echo.HTTPError).Code, http.StatusBadRequest)

	// Trial IDs not found.
	c = newTestEchoContext(curUser)
	c.SetRequest(httptest.NewRequest(http.MethodGet, endpoint+"?trialIds=-1", nil))
	err = requestFunc(c)
	require.Error(t, err)
	require.Equal(t, err.(*echo.HTTPError).Code, http.StatusNotFound)

	// One trial.
	c = newTestEchoContext(curUser)
	c.SetRequest(httptest.NewRequest(http.MethodGet,
		endpoint+fmt.Sprintf("?trialIds=%d", trials[0].ID), nil))
	resp := httptest.NewRecorder()
	c.SetResponse(echo.NewResponse(resp, nil))
	require.NoError(t, requestFunc(c))
	compareMetrics(t, trials[0].ID, resp, trainingMetrics[0])

	// Other trial.

	// Both trials.
}

func TestTrialAuthZEcho(t *testing.T) {
	api, authZExp, _, curUser, _ := setupExpAuthTest(t, nil)
	trial := createTestTrial(t, api, curUser)

	funcCalls := []func(id int) error{
		func(id int) error {
			ctx := newTestEchoContext(curUser)
			ctx.SetParamNames("trial_id")
			ctx.SetParamValues(fmt.Sprintf("%d", id))
			ctx.SetRequest(httptest.NewRequest(http.MethodGet, "/", nil))
			_, err := api.m.getTrial(ctx)
			return err
		},
		func(id int) error {
			ctx := newTestEchoContext(curUser)
			ctx.SetParamNames("trial_id")
			ctx.SetParamValues(fmt.Sprintf("%d", id))
			ctx.SetRequest(httptest.NewRequest(http.MethodGet, "/", nil))
			_, err := api.m.getTrialMetrics(ctx)
			return err
		},
	}

	for i, funcCall := range funcCalls {
		require.Equal(t, trialNotFoundErrEcho(-999), funcCall(-999))

		// Can't view trials experiment gives same error.
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(false, nil).Once()
		require.Equal(t, trialNotFoundErrEcho(trial.ID), funcCall(trial.ID))

		// Experiment view error returns error unmodified.
		expectedErr := fmt.Errorf("canGetTrialError")
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(false, expectedErr).Once()
		require.Equal(t, expectedErr, funcCall(trial.ID))

		// Action func error returns error in forbidden.
		expectedErr = echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("%dError", i))
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(true, nil).Once()
		authZExp.On("CanGetExperimentArtifacts", mock.Anything, curUser, mock.Anything).
			Return(fmt.Errorf("%dError", i)).Once()
		require.Equal(t, expectedErr, funcCall(trial.ID))
	}
}
