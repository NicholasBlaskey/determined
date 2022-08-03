package project

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

type ProjectAuthZBasic struct{}

// CanGetProject always return true and a nil error for basic auth.
func (a *ProjectAuthZBasic) CanGetProject(
	curUser model.User, project *projectv1.Project,
) (canGetProject bool, serverError error) {
	return true, nil
}

// CanCreateProject always returns true and a nil error for basic auth.
func (a *ProjectAuthZBasic) CanCreateProject(
	curUser model.User, willBeInWorkspace *workspacev1.Workspace,
) error {
	return nil
}

// CanSetProjectNotes always returns nil for basic auth.
func (a *ProjectAuthZBasic) CanSetProjectNotes(curUser model.User, project *projectv1.Project) error {
	return nil
}

func shouldBeAdminOrOwnWorkspaceOrProject(curUser model.User, project *projectv1.Project) error {
	fmt.Println("REMOVE ME glue")

	// Is admin or owner of the project?
	if curUser.Admin || curUser.ID == model.UserID(project.UserId) {
		return nil
	}
	// Is owner of the workspace?
	type workspace struct {
		bun.BaseModel `bun:"table:workspaces"`
	}
	exists, err := db.Bun().NewSelect().Model((*workspace)(nil)).
		Where("id = ?", project.WorkspaceId).
		Where("user_id = ?", curUser.ID).Exists(context.TODO())
	fmt.Println("EXISTS!!!", exists, err, project.WorkspaceId, curUser.ID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("non admin users need to own the project or workspace")
	}
	return nil
}

// CanSetProjectName returns an error if the user isn't the owner of the project or workspace.
func (a *ProjectAuthZBasic) CanSetProjectName(curUser model.User, project *projectv1.Project) error {
	return fmt.Errorf("can't set project name: %w",
		shouldBeAdminOrOwnWorkspaceOrProject(curUser, project))
}

// CanSetProjectDescription returns an error if the user isn't the owner of the project or workspace.
func (a *ProjectAuthZBasic) CanSetProjectDescription(
	curUser model.User, project *projectv1.Project,
) error {
	return fmt.Errorf("can't set project name: %w",
		shouldBeAdminOrOwnWorkspaceOrProject(curUser, project))
}

// CanDeleteProject returns an error if the user isn't the owner of the project or workspace.
func (a *ProjectAuthZBasic) CanDeleteProject(curUser model.User, project *projectv1.Project) error {
	if err := shouldBeAdminOrOwnWorkspaceOrProject(curUser, project); err != nil {
		return fmt.Errorf("can't delete project: %w", err)
	}
	return nil
}

func (a *ProjectAuthZBasic) CanMoveProject(
	curUser model.User, project *projectv1.Project, from, to *workspacev1.Workspace,
) error {
	return nil
}

func (a *ProjectAuthZBasic) CanArchiveProject(curUser model.User, project *projectv1.Project) error {
	return fmt.Errorf("can't archive project: %w",
		shouldBeAdminOrOwnWorkspaceOrProject(curUser, project))
}

func (a *ProjectAuthZBasic) CanUnarchiveProject(
	curUser model.User, project *projectv1.Project,
) error {
	return fmt.Errorf("can't unarchive project: %w",
		shouldBeAdminOrOwnWorkspaceOrProject(curUser, project))
}

func init() {
	AuthZProvider.Register("basic", &ProjectAuthZBasic{})
}
