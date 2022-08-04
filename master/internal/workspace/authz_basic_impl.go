package workspace

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

type WorkspaceAuthZBasic struct{}

func (a *WorkspaceAuthZBasic) CanGetWorkspace(
	curUser model.User, workspace *workspacev1.Workspace,
) (canGetWorkspace bool, serverError error) {
	return true, nil
}

func (a *WorkspaceAuthZBasic) FilterWorkspaceProjects(
	curUser model.User, projects []*projectv1.Project,
) ([]*projectv1.Project, error) {
	return projects, nil
}

func (a *WorkspaceAuthZBasic) FilterWorkspaces(
	curUser model.User, workspaces []*workspacev1.Workspace,
) ([]*workspacev1.Workspace, error) {
	return workspaces, nil
}

func (a *WorkspaceAuthZBasic) CanCreateWorkspace(curUser model.User) error {
	return nil
}

func (a *WorkspaceAuthZBasic) CanSetWorkspacesName(
	curUser model.User, workspace *workspacev1.Workspace,
) error {
	if !curUser.Admin && curUser.ID != model.UserID(workspace.UserId) {
		return fmt.Errorf("only admins may set other user's workspaces names")
	}
	return nil
}

func (a *WorkspaceAuthZBasic) CanDeleteWorkspace(
	curUser model.User, workspace *workspacev1.Workspace,
) error {
	if !curUser.Admin && curUser.ID != model.UserID(workspace.UserId) {
		return fmt.Errorf("only admins may delete other user's workspaces")
	}
	return nil
}

func (a *WorkspaceAuthZBasic) CanArchiveWorkspace(
	curUser model.User, workspace *workspacev1.Workspace,
) error {
	if !curUser.Admin && curUser.ID != model.UserID(workspace.UserId) {
		return fmt.Errorf("only admins may archive other user's workspaces")
	}
	return nil
}

func (a *WorkspaceAuthZBasic) CanUnarchiveWorkspace(
	curUser model.User, workspace *workspacev1.Workspace,
) error {
	if !curUser.Admin && curUser.ID != model.UserID(workspace.UserId) {
		return fmt.Errorf("only admins may unarchive other user's workspaces")
	}
	return nil
}

func (a *WorkspaceAuthZBasic) CanPinWorkspace(
	curUser model.User, workspace *workspacev1.Workspace,
) error {
	return nil
}

func (a *WorkspaceAuthZBasic) CanUnpinWorkspace(
	curUser model.User, workspace *workspacev1.Workspace,
) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &WorkspaceAuthZBasic{})
}
