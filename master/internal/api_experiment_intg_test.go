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
	//"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	//"google.golang.org/protobuf/types/known/wrapperspb"

	//"github.com/determined-ai/determined/master/internal/config"
	//"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/mocks"
	//"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	//"github.com/determined-ai/determined/proto/pkg/userv1"
)

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

func TestAuthzGetExperiment(t *testing.T) {
	api, authZExp, curUser, ctx := SetupExpAuthTest(t)

	// Create test experiment.
	exp := &model.Experiment{
		JobID:                model.JobID(uuid.New().String()),
		State:                model.PausedState,
		OwnerID:              &curUser.ID,
		ProjectID:            1,
		StartTime:            time.Now(),
		ModelDefinitionBytes: []byte{10, 11, 12},
		Config: expconf.ExperimentConfig{
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

	// Not found returns same as permission denied.
	_, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: -999})
	require.Equal(t, expNotFoundError(-999).Error(), err.Error())

	authZExp.On("CanGetExperiment", mock.Anything, mock.Anything).Return(false, nil).Once()
	_, err = api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: int32(exp.ID)})
	require.Equal(t, expNotFoundError(exp.ID).Error(), err.Error())

	// Error returns error unmodified.
	expectedErr := fmt.Errorf("canGetExperimentError")
	authZExp.On("CanGetExperiment", mock.Anything, mock.Anything).Return(false, expectedErr).Once()
	_, err = api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: int32(exp.ID)})
	require.Equal(t, expectedErr, err)

	authZExp.On("CanGetExperiment", mock.Anything, mock.Anything).Return(true, nil).Once()
	res, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: int32(exp.ID)})
	require.NoError(t, err)
	require.Equal(t, int32(exp.ID), res.Experiment.Id)
}
