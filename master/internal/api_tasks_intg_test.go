//go:build integration
// +build integration

package internal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
)

func TestTaskAuthZ(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t)

	trial := createTestTrial(t, api, curUser)
	taskID := string(trial.TaskID)

	cases := []struct {
		DenyFuncName string
		IDToReqCall  func(id string) error
	}{
		{"CanGetExperimentArtifacts", func(id string) error {
			_, err := api.GetTask(ctx, &apiv1.GetTaskRequest{
				TaskId: id,
			})
			return err
		}},
		{"CanEditExperiment", func(id string) error {
			_, err := api.ReportCheckpoint(ctx, &apiv1.ReportCheckpointRequest{
				Checkpoint: &checkpointv1.Checkpoint{TaskId: id},
			})
			return err
		}},
		{"CanGetExperimentArtifacts", func(id string) error {
			return api.TaskLogs(&apiv1.TaskLogsRequest{
				TaskId: id,
			}, mockStream[*apiv1.TaskLogsResponse]{ctx})
		}},
		{"CanGetExperimentArtifacts", func(id string) error {
			return api.TaskLogsFields(&apiv1.TaskLogsFieldsRequest{
				TaskId: id,
			}, mockStream[*apiv1.TaskLogsFieldsResponse]{ctx})
		}},
	}

	for _, curCase := range cases {
		require.ErrorIs(t, curCase.IDToReqCall("-999"), errTaskNotFound("-999"))

		// Can't view allocation's experiment gives same error.
		authZExp.On("CanGetExperiment", curUser, mock.Anything).Return(false, nil).Once()
		require.ErrorIs(t, curCase.IDToReqCall(taskID), errTaskNotFound(model.TaskID(taskID)))

		// Experiment view error is returned unmodified.
		expectedErr := fmt.Errorf("canGetExperimentError")
		authZExp.On("CanGetExperiment", curUser, mock.Anything).Return(false, expectedErr).Once()
		require.ErrorIs(t, curCase.IDToReqCall(taskID), expectedErr)

		// Action func error returns err in forbidden.
		expectedErr = status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Error")
		authZExp.On("CanGetExperiment", curUser, mock.Anything).Return(true, nil).Once()
		authZExp.On(curCase.DenyFuncName, curUser, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.ErrorIs(t, curCase.IDToReqCall(taskID), expectedErr)
	}
}
