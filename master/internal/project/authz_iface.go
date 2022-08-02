package project

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

type ProjectAuthZ interface {
	// GET /api/v1/projects/:project_id
	CanGetProject(
		curUser model.User, project *projectv1.Project,
	) (canGetProject bool, serverError error)

	// POST /api/v1/workspaces/:workspace_id/projects
	CanCreateProject(curUser model.User, targetWorkspace *workspacev1.Workspace) error

	// HMMM do we need put????
	// BOTH
	CanSetProjectNotes(curUser model.User, project *projectv1.Project) error

	//
	CanSetProjectName(curUser model.User, project *projectv1.Project) error
	CanSetProjectDescription(curUser model.User, project *projectv1.Project) error

	//
	CanDeleteProject(curUser model.User, targetProject *projectv1.Project) error

	//
	CanMoveProject(curUser model.User, fromWorkspace, toWorkspace *workspacev1.Workspace) error

	//
	CanArchiveProject(curUSer model.User, project *projectv1.Project) error
	//
	CanUnarchiveProject(curUSer model.User, project *projectv1.Project) error
}

var AuthZProvider authz.AuthZProviderType[ProjectAuthZ]
