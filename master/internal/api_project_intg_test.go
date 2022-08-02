//go:build integration
// +build integration

package internal

import (
	"context"
	//"fmt"
	"testing"

	// "google.golang.org/grpc/codes"
	// "google.golang.org/grpc/status"

	// "github.com/google/uuid"
	// "github.com/pkg/errors"
	// "github.com/stretchr/testify/mock"
	// "github.com/stretchr/testify/require"
	// "google.golang.org/protobuf/types/known/wrapperspb"

	// "github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/project"
	// "github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	// "github.com/determined-ai/determined/proto/pkg/apiv1"
	// "github.com/determined-ai/determined/proto/pkg/projectv1"
	// "github.com/determined-ai/determined/proto/pkg/workspacev1"
	//"github.com/determined-ai/determined/proto/pkg/userv1"
)

var projectAuthZ *mocks.ProjectAuthZ

/*
func workspaceNotFoundErr(id int) error {
	return status.Errorf(codes.NotFound, fmt.Sprintf("workspace (%d) not found", id))
}
*/

func SetupProjectAuthZTest(
	t *testing.T,
) (*apiServer, *mocks.ProjectAuthZ, *mocks.WorkspaceAuthZ, model.User, context.Context) {
	api, workspaceAuthZ, curUser, ctx := SetupWorkspaceAuthZTest(t)

	if projectAuthZ == nil {
		projectAuthZ = &mocks.ProjectAuthZ{}
		project.AuthZProvider.Register("mock", projectAuthZ)
	}
	return api, projectAuthZ, workspaceAuthZ, curUser, ctx
}

func TestAuthzGetProject(t *testing.T) {
	api, projectAuthZ, workspaceAuthZ, _, ctx := SetupProjectAuthZTest(t)

	// Deny returns same as 494,
	//_, err := api.GetProject

	/*
		// Deny returns same as 404.
		_, err := api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: -9999})
		require.Equal(t, workspaceNotFoundErr(-9999).Error(), err.Error())

		workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(false, nil).Once()
		_, err = api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: 1})
		require.Equal(t, workspaceNotFoundErr(1).Error(), err.Error())

		// A error returned by CanGetWorkspace is returned unmodified.
		expectedErr := fmt.Errorf("canGetWorkspaceError")
		workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).
			Return(false, expectedErr).Once()
		_, err = api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: 1})
		require.Equal(t, expectedErr, err)
	*/
}
