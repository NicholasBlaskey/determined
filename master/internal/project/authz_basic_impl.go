package project

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

type ProjectAuthZBasic struct{}

// CanGetProject always return true and a nil error for basic auth.
func (a *ProjectAuthZBasic) CanGetProject(
	curUser model.User, project *projectv1.Project,
) (canGetProject bool, serverError error) {
	fmt.Println("REMOVE ME")
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

func shouldBeOwnerOfProjectOrWorkspace(curUser model.User, project *projectv1.Project) error {
	// if curUser
	// var w Workspace
	// db.Bun().NewSelect().Model
	// BIG TODO in model

	return fmt.Errorf("user needs to own the project or workspace")
	return nil
}

// CanSetProjectName returns an error if the user isn't the owner of the project or workspace.
func (a *ProjectAuthZBasic) CanSetProjectName(curUser model.User, project *projectv1.Project) error {
	return fmt.Errorf("can't set project name: %w",
		shouldBeOwnerOfProjectOrWorkspace(curUser, project))
}

// CanSetProjectDescription returns an error if the user isn't the owner of the project or workspace.
func (a *ProjectAuthZBasic) CanSetProjectDescription(
	curUser model.User, project *projectv1.Project,
) error {
	return fmt.Errorf("can't set project name: %w",
		shouldBeOwnerOfProjectOrWorkspace(curUser, project))
}

// CanDeleteProject returns an error if the user isn't the owner of the project or workspace.
func (a *ProjectAuthZBasic) CanDeleteProject(
	curUser model.User, targetProject *projectv1.Project,
) error {
	return nil
}

func (a *ProjectAuthZBasic) CanMoveProject(
	curUser model.User, project *projectv1.Project, from, to *workspacev1.Workspace,
) error {
	return nil
}

func (a *ProjectAuthZBasic) CanArchiveProject(curUser model.User, project *projectv1.Project) error {
	return fmt.Errorf("can't archive project: %w",
		shouldBeOwnerOfProjectOrWorkspace(curUser, project))
}

func (a *ProjectAuthZBasic) CanUnarchiveProject(
	curUser model.User, project *projectv1.Project,
) error {
	return fmt.Errorf("can't unarchive project: %w",
		shouldBeOwnerOfProjectOrWorkspace(curUser, project))
}

func init() {
	AuthZProvider.Register("basic", &ProjectAuthZBasic{})
}
