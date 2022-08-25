//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/mocks"
	trialauth "github.com/determined-ai/determined/master/internal/trial"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

var authZTrial *mocks.TrialAuthZ

func SetupTrialAuthTest(t *testing.T) (
	*apiServer, *mocks.TrialAuthZ, model.User, context.Context,
) {
	api, _, _, user, ctx := SetupExpAuthTest(t)
	if authZTrial == nil {
		authZTrial = &mocks.TrialAuthZ{}
		trialauth.AuthZProvider.Register("mock", authZTrial)
	}
	return api, authZTrial, user, ctx
}

func createTestTrial(
	t *testing.T, api *apiServer, curUser model.User,
) *model.Trial {
	exp := createTestExpWithProjectID(t, api, curUser, 1)

	task := &model.Task{TaskType: model.TaskTypeTrial}
	require.NoError(t, api.m.db.AddTask(task))

	trial := &model.Trial{
		StartTime:    time.Now(),
		State:        model.PausedState,
		ExperimentID: exp.ID,
		TaskID:       task.TaskID,
	}
	require.NoError(t, api.m.db.AddTrial(trial))

	// Return trial exactly the way the API will generally get it.
	outTrial, err := api.m.db.TrialByID(trial.ID)
	require.NoError(t, err)
	return outTrial
}

func TestAuthZGetTrialAndCanDoActions(t *testing.T) {
	api, authZTrial, curUser, ctx := SetupTrialAuthTest(t)
	trial := createTestTrial(t, api, curUser)

	cases := []struct {
		DenyFuncName   string
		IDToReqCall    func(id int) error
		SkipActionFunc bool
	}{
		{"CanGetTrialLogs", func(id int) error {
			return api.TrialLogs(&apiv1.TrialLogsRequest{
				TrialId: int32(id),
			}, mockStream[*apiv1.TrialLogsResponse]{ctx})
		}, false},
		{"CanGetTrialLogs", func(id int) error {
			return api.TrialLogsFields(&apiv1.TrialLogsFieldsRequest{
				TrialId: int32(id),
			}, mockStream[*apiv1.TrialLogsFieldsResponse]{ctx})
		}, false},
		{"CanGetTrialsCheckpoints", func(id int) error {
			_, err := api.GetTrialCheckpoints(ctx, &apiv1.GetTrialCheckpointsRequest{
				Id: int32(id),
			})
			return err
		}, false},
		{"CanKillTrial", func(id int) error {
			_, err := api.KillTrial(ctx, &apiv1.KillTrialRequest{
				Id: int32(id),
			})
			return err
		}, false},
		{"CanGetTrial", func(id int) error {
			_, err := api.GetTrial(ctx, &apiv1.GetTrialRequest{
				TrialId: int32(id),
			})
			return err
		}, true},
		{"CanGetTrialsSummary", func(id int) error {
			_, err := api.SummarizeTrial(ctx, &apiv1.SummarizeTrialRequest{
				TrialId: int32(id),
			})
			return err
		}, false},
		{"CanGetTrialsSummary", func(id int) error {
			_, err := api.CompareTrials(ctx, &apiv1.CompareTrialsRequest{
				TrialIds: []int32{int32(id)},
			})
			return err
		}, false},
		{"CanGetTrialsWorkloads", func(id int) error {
			_, err := api.GetTrialWorkloads(ctx, &apiv1.GetTrialWorkloadsRequest{
				TrialId: int32(id),
			})
			return err
		}, false},
		{"CanGetTrialsProfilerMetrics", func(id int) error {
			return api.GetTrialProfilerMetrics(&apiv1.GetTrialProfilerMetricsRequest{
				Labels: &trialv1.TrialProfilerMetricLabels{TrialId: int32(id)},
			}, mockStream[*apiv1.GetTrialProfilerMetricsResponse]{ctx})
		}, false},
		{"CanGetTrialsProfilerAvailableSeries", func(id int) error {
			return api.GetTrialProfilerAvailableSeries(
				&apiv1.GetTrialProfilerAvailableSeriesRequest{
					TrialId: int32(id),
				}, mockStream[*apiv1.GetTrialProfilerAvailableSeriesResponse]{ctx})
		}, false},
		{"CanPostTrialsProfilerMetricsBatch", func(id int) error {
			_, err := api.PostTrialProfilerMetricsBatch(ctx,
				&apiv1.PostTrialProfilerMetricsBatchRequest{
					Batches: []*trialv1.TrialProfilerMetricsBatch{
						{
							Labels: &trialv1.TrialProfilerMetricLabels{TrialId: int32(id)},
						},
					},
				})
			return err
		}, false},
		{"CanGetTrialsSearcherOperation", func(id int) error {
			_, err := api.GetCurrentTrialSearcherOperation(ctx,
				&apiv1.GetCurrentTrialSearcherOperationRequest{
					TrialId: int32(id),
				})
			return err
		}, false},
		{"CanCompleteTrialsSearcherValidation", func(id int) error {
			_, err := api.CompleteTrialSearcherValidation(ctx,
				&apiv1.CompleteTrialSearcherValidationRequest{
					TrialId: int32(id),
				})
			return err
		}, false},
		{"CanReportTrialsSearcherEarlyExit", func(id int) error {
			_, err := api.ReportTrialSearcherEarlyExit(ctx,
				&apiv1.ReportTrialSearcherEarlyExitRequest{
					TrialId: int32(id),
				})
			return err
		}, false},
		{"CanReportTrialsProgress", func(id int) error {
			_, err := api.ReportTrialProgress(ctx,
				&apiv1.ReportTrialProgressRequest{
					TrialId: int32(id),
				})
			return err
		}, false},
		{"CanReportTrialsTrainingMetrics", func(id int) error {
			_, err := api.ReportTrialTrainingMetrics(ctx,
				&apiv1.ReportTrialTrainingMetricsRequest{
					TrainingMetrics: &trialv1.TrialMetrics{TrialId: int32(id)},
				})
			return err
		}, false},
		{"CanReportTrialsValidationMetrics", func(id int) error {
			_, err := api.ReportTrialValidationMetrics(ctx,
				&apiv1.ReportTrialValidationMetricsRequest{
					ValidationMetrics: &trialv1.TrialMetrics{TrialId: int32(id)},
				})
			return err
		}, false},
		{"CanPostTrialsRunnerMetadata", func(id int) error {
			_, err := api.PostTrialRunnerMetadata(ctx, &apiv1.PostTrialRunnerMetadataRequest{
				TrialId: int32(id),
			})
			return err
		}, false},
	}

	for _, curCase := range cases {
		// Actual trial doesn't exist gives 404.
		require.ErrorIs(t, curCase.IDToReqCall(-999), errTrialNotFound(-999))

		// Can't view trial gives same error.
		authZTrial.On("CanGetTrial", curUser, mock.Anything).Return(false, nil).Once()
		require.ErrorIs(t, curCase.IDToReqCall(trial.ID), errTrialNotFound(trial.ID))

		// Trial view error returns error unmodified.
		expectedErr := fmt.Errorf("canGetTrialError")
		authZTrial.On("CanGetTrial", curUser, mock.Anything).Return(false, expectedErr).Once()
		require.ErrorIs(t, curCase.IDToReqCall(trial.ID), expectedErr)

		if curCase.SkipActionFunc {
			continue
		}

		// Action func error returns error in forbidden.
		expectedErr = status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Error")
		authZTrial.On("CanGetTrial", curUser, mock.Anything).Return(true, nil).Once()
		authZTrial.On(curCase.DenyFuncName, curUser, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.ErrorIs(t, curCase.IDToReqCall(trial.ID), expectedErr)
	}
}
