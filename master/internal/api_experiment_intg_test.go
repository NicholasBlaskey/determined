//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/wrapperspb"
	//"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	//"google.golang.org/protobuf/types/known/wrapperspb"
	"google.golang.org/protobuf/types/known/structpb"

	//"github.com/determined-ai/determined/master/internal/config"
	//"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/mocks"
	//"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

type mockStream[T any] struct {
	ctx context.Context
}

func (m mockStream[T]) Send(resp T) error             { return nil }
func (m mockStream[T]) SetHeader(metadata.MD) error   { return nil }
func (m mockStream[T]) SendHeader(metadata.MD) error  { return nil }
func (m mockStream[T]) SetTrailer(metadata.MD)        {}
func (m mockStream[T]) Context() context.Context      { return m.ctx }
func (m mockStream[T]) SendMsg(mes interface{}) error { return nil }
func (m mockStream[T]) RecvMsg(mes interface{}) error { return nil }

func expNotFoundError(expID int) error {
	return status.Errorf(codes.NotFound, "experiment not found: %d", expID)
}

var authZExp *mocks.ExperimentAuthZ

func SetupExpAuthTest(t *testing.T) (
	*apiServer, *mocks.ExperimentAuthZ, model.User, context.Context,
) {
	api, _, user, ctx := SetupUserAuthzTest(t)
	if authZExp == nil {
		authZExp = &mocks.ExperimentAuthZ{}
		expauth.AuthZProvider.Register("mock", authZExp)
	}
	return api, authZExp, user, ctx
}

func createTestExp(
	t *testing.T, api *apiServer, curUser model.User, labels ...string,
) *model.Experiment {
	labelMap := make(map[string]bool)
	for _, l := range labels {
		labelMap[l] = true
	}

	exp := &model.Experiment{
		JobID:                model.JobID(uuid.New().String()),
		State:                model.PausedState,
		OwnerID:              &curUser.ID,
		ProjectID:            1,
		StartTime:            time.Now(),
		ModelDefinitionBytes: []byte{10, 11, 12},
		Config: expconf.ExperimentConfig{
			RawLabels:     labelMap,
			RawEntrypoint: &expconf.EntrypointV0{"test"},
			RawCheckpointStorage: &expconf.CheckpointStorageConfig{
				RawSharedFSConfig: &expconf.SharedFSConfig{
					RawHostPath: ptrs.Ptr("/"),
				},
			},
			RawHyperparameters: expconf.Hyperparameters{},
			RawName:            expconf.Name{ptrs.Ptr("name")},
			RawReproducibility: &expconf.ReproducibilityConfig{ptrs.Ptr(uint32(42))},
			RawSearcher: &expconf.SearcherConfig{
				RawMetric: ptrs.Ptr("loss"),
				RawSingleConfig: &expconf.SingleConfig{
					&expconf.Length{Units: 10, Unit: "batches"},
				},
			},
		},
	}
	require.NoError(t, api.m.db.AddExperiment(exp))

	// Get experiment as our API mostly will to make it easier to mock.
	exp, err := api.m.db.ExperimentWithoutConfigByID(exp.ID)
	require.NoError(t, err)
	return exp
}

func TestAuthZGetExperiment(t *testing.T) {
	api, authZExp, curUser, ctx := SetupExpAuthTest(t)
	exp := createTestExp(t, api, curUser)

	// Not found returns same as permission denied.
	_, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: -999})
	require.Equal(t, expNotFoundError(-999).Error(), err.Error())

	authZExp.On("CanGetExperiment", curUser, mock.Anything).Return(false, nil).Once()
	_, err = api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: int32(exp.ID)})
	require.Equal(t, expNotFoundError(exp.ID).Error(), err.Error())

	// Error returns error unmodified.
	expectedErr := fmt.Errorf("canGetExperimentError")
	authZExp.On("CanGetExperiment", curUser, mock.Anything).Return(false, expectedErr).Once()
	_, err = api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: int32(exp.ID)})
	require.Equal(t, expectedErr, err)

	authZExp.On("CanGetExperiment", curUser, mock.Anything).Return(true, nil).Once()
	res, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: int32(exp.ID)})
	require.NoError(t, err)
	require.Equal(t, int32(exp.ID), res.Experiment.Id)
}

func TestAuthZGetExperiments(t *testing.T) {
	api, authZExp, curUser, ctx := SetupExpAuthTest(t)

	// Returns only what we filter.
	expected := []*model.Experiment{createTestExp(t, api, curUser), createTestExp(t, api, curUser)}
	authZExp.On("FilterExperiments", curUser, mock.Anything).Return(expected, nil).Once()
	resp, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{Limit: -1})
	require.NoError(t, err)
	require.Equal(t, 2, len(resp.Experiments))
	for i := range expected {
		require.Equal(t, int32(expected[i].ID), resp.Experiments[i].Id)
	}

	// Error passes right through.
	expectedErr := fmt.Errorf("filterExperimentsError")
	authZExp.On("FilterExperiments", curUser, mock.Anything).Return(nil, expectedErr).Once()
	_, err = api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{})
	require.Equal(t, expectedErr, err)
}

func TestAuthZPreviewHPSearch(t *testing.T) {
	api, authZExp, curUser, ctx := SetupExpAuthTest(t)

	// Can't preview hp search returns error with PermissionDenied
	expectedErr := status.Errorf(codes.PermissionDenied, "canPreviewHPSearchError")
	authZExp.On("CanPreviewHPSearch", curUser).Return(fmt.Errorf("canPreviewHPSearchError")).Once()
	_, err := api.PreviewHPSearch(ctx, &apiv1.PreviewHPSearchRequest{})
	require.Equal(t, expectedErr.Error(), err.Error())
}

/*
func TestAuthZGetExperimentLabels(t *testing.T) {
	api, authZExp, curUser, ctx := SetupExpAuthTest(t)

	// Error in FilterExperiments passes through.
	expectedErr := fmt.Errorf("filterExperimentsError")
	authZExp.On("FilterExperiments", curUser, mock.Anything).Return(nil, expectedErr).Once()
	_, err := api.GetExperimentLabels(ctx, &apiv1.GetExperimentLabelsRequest{})
	require.Equal(t, expectedErr, err)
}
*/

func TestAuthZExpCompareTrialsSample(t *testing.T) {
	api, authZExp, curUser, ctx := SetupExpAuthTest(t)

	exp0 := createTestExp(t, api, curUser)
	exp1 := createTestExp(t, api, curUser)
	req := &apiv1.ExpCompareTrialsSampleRequest{
		ExperimentIds: []int32{int32(exp0.ID), int32(exp1.ID)},
	}

	// Can't view first experiment gets error.
	expectedErr := status.Errorf(codes.PermissionDenied, "firstError")
	authZExp.On("CanGetExperiment", curUser, exp0).Return(true, nil).Once()
	authZExp.On("CanGetTrialsSample", curUser, exp0).Return(fmt.Errorf("firstError")).Once()
	err := api.ExpCompareTrialsSample(req, mockStream[*apiv1.ExpCompareTrialsSampleResponse]{ctx})
	require.Equal(t, expectedErr.Error(), err.Error())

	// Can't view second experiment gets error.
	expectedErr = status.Errorf(codes.PermissionDenied, "secondError")
	authZExp.On("CanGetExperiment", curUser, exp0).Return(true, nil).Once()
	authZExp.On("CanGetTrialsSample", curUser, exp0).Return(nil).Once()
	authZExp.On("CanGetExperiment", curUser, exp1).Return(true, nil).Once()
	authZExp.On("CanGetTrialsSample", curUser, exp1).Return(fmt.Errorf("secondError")).Once()
	err = api.ExpCompareTrialsSample(req, mockStream[*apiv1.ExpCompareTrialsSampleResponse]{ctx})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthZGetExperimentAndCanDoActions(t *testing.T) {
	api, authZExp, curUser, ctx := SetupExpAuthTest(t)
	exp := createTestExp(t, api, curUser)

	cases := []struct {
		DenyFuncName string
		IDToReqCall  func(id int) error
	}{
		{"CanDeleteExperiment", func(id int) error {
			_, err := api.DeleteExperiment(ctx, &apiv1.DeleteExperimentRequest{
				ExperimentId: int32(id),
			})
			return err
		}},
		{"CanGetExperimentValidationHistory", func(id int) error {
			_, err := api.GetExperimentValidationHistory(ctx,
				&apiv1.GetExperimentValidationHistoryRequest{ExperimentId: int32(id)})
			return err
		}},
		{"CanActivateExperiment", func(id int) error {
			_, err := api.ActivateExperiment(ctx, &apiv1.ActivateExperimentRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanPauseExperiment", func(id int) error {
			_, err := api.PauseExperiment(ctx, &apiv1.PauseExperimentRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanCancelExperiment", func(id int) error {
			_, err := api.CancelExperiment(ctx, &apiv1.CancelExperimentRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanKillExperiment", func(id int) error {
			_, err := api.KillExperiment(ctx, &apiv1.KillExperimentRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanArchiveExperiment", func(id int) error {
			_, err := api.ArchiveExperiment(ctx, &apiv1.ArchiveExperimentRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanUnarchiveExperiment", func(id int) error {
			_, err := api.UnarchiveExperiment(ctx, &apiv1.UnarchiveExperimentRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanSetExperimentsName", func(id int) error {
			_, err := api.PatchExperiment(ctx, &apiv1.PatchExperimentRequest{
				Experiment: &experimentv1.PatchExperiment{
					Id:   int32(id),
					Name: wrapperspb.String("toname"),
				},
			})
			return err
		}},
		{"CanSetExperimentsNotes", func(id int) error {
			_, err := api.PatchExperiment(ctx, &apiv1.PatchExperimentRequest{
				Experiment: &experimentv1.PatchExperiment{
					Id:    int32(id),
					Notes: wrapperspb.String("tonotes"),
				},
			})
			return err
		}},
		{"CanSetExperimentsDescription", func(id int) error {
			_, err := api.PatchExperiment(ctx, &apiv1.PatchExperimentRequest{
				Experiment: &experimentv1.PatchExperiment{
					Id:          int32(id),
					Description: wrapperspb.String("todesc"),
				},
			})
			return err
		}},
		{"CanSetExperimentsLabels", func(id int) error {
			l, err := structpb.NewList([]any{"l1", "l2"})
			require.NoError(t, err)
			_, err = api.PatchExperiment(ctx, &apiv1.PatchExperimentRequest{
				Experiment: &experimentv1.PatchExperiment{
					Id:     int32(id),
					Labels: l,
				},
			})
			return err
		}},
		{"CanGetMetricNames", func(id int) error {
			return api.MetricNames(&apiv1.MetricNamesRequest{
				ExperimentId: int32(id),
			}, mockStream[*apiv1.MetricNamesResponse]{ctx})
		}},
		{"CanGetMetricBatches", func(id int) error {
			return api.MetricBatches(&apiv1.MetricBatchesRequest{
				ExperimentId: int32(id),
			}, mockStream[*apiv1.MetricBatchesResponse]{ctx})
		}},
		{"CanGetTrialsSnapshot", func(id int) error {
			return api.TrialsSnapshot(&apiv1.TrialsSnapshotRequest{
				ExperimentId: int32(id),
			}, mockStream[*apiv1.TrialsSnapshotResponse]{ctx})
		}},
		{"CanGetTrialsSample", func(id int) error {
			return api.TrialsSample(&apiv1.TrialsSampleRequest{
				ExperimentId: int32(id),
			}, mockStream[*apiv1.TrialsSampleResponse]{ctx})
		}},
		{"CanComputeHPImportance", func(id int) error {
			_, err := api.ComputeHPImportance(ctx, &apiv1.ComputeHPImportanceRequest{
				ExperimentId: int32(id),
			})
			return err
		}},
		{"CanGetHPImportance", func(id int) error {
			return api.GetHPImportance(&apiv1.GetHPImportanceRequest{
				ExperimentId: int32(id),
			}, mockStream[*apiv1.GetHPImportanceResponse]{ctx})
		}},
		{"CanGetBestSearcherValidationMetric", func(id int) error {
			_, err := api.GetBestSearcherValidationMetric(ctx,
				&apiv1.GetBestSearcherValidationMetricRequest{ExperimentId: int32(id)})
			return err
		}},
		{"CanGetModelDef", func(id int) error {
			_, err := api.GetModelDef(ctx, &apiv1.GetModelDefRequest{
				ExperimentId: int32(id),
			})
			return err
		}},
		{"CanGetModelDefTree", func(id int) error {
			_, err := api.GetModelDefTree(ctx, &apiv1.GetModelDefTreeRequest{
				ExperimentId: int32(id),
			})
			return err
		}},
		{"CanGetModelDefFile", func(id int) error {
			_, err := api.GetModelDefFile(ctx, &apiv1.GetModelDefFileRequest{
				ExperimentId: int32(id),
			})
			return err
		}},
	}

	for _, curCase := range cases {
		// Not found returns same as permission denied.
		require.Equal(t, expNotFoundError(-999).Error(), curCase.IDToReqCall(-999).Error())

		authZExp.On("CanGetExperiment", curUser, mock.Anything).Return(false, nil).Once()
		require.Equal(t, expNotFoundError(exp.ID).Error(), curCase.IDToReqCall(exp.ID).Error())

		// CanGetExperiment error returns unmodified.
		expectedErr := fmt.Errorf("canGetExperimentError")
		authZExp.On("CanGetExperiment", curUser, mock.Anything).
			Return(false, expectedErr).Once()
		require.Equal(t, expectedErr, curCase.IDToReqCall(exp.ID))

		// Deny returns error with PermissionDenied.
		expectedErr = status.Errorf(codes.PermissionDenied, curCase.DenyFuncName+"Error")
		authZExp.On("CanGetExperiment", curUser, mock.Anything).Return(true, nil).Once()
		authZExp.On(curCase.DenyFuncName, curUser, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.Equal(t, expectedErr.Error(), curCase.IDToReqCall(exp.ID).Error())
	}
}
