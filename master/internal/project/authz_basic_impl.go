package project

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

type ProjectAuthZBasic struct{}

func (a *ProjectAuthZBasic) CanGetProject(
	curUser model.User, project *projectv1.Project,
) (canGetProject bool, serverError error) {
	fmt.Println("REMOVE ME")
	return true, nil
}

func (a *ProjectAuthZBasic) CanCreateProject(
	curUser model.User, willBeInWorkspace *workspacev1.Workspace,
) error {
	return nil
}

func (a *ProjectAuthZBasic) CanSetProjectNotes(curUser model.User, project *projectv1.Project) error {
	return nil
}

func (a *ProjectAuthZBasic) CanSetProjectName(curUser model.User, project *projectv1.Project) error {
	return nil
}

func (a *ProjectAuthZBasic) CanSetProjectDescription(
	curUser model.User, project *projectv1.Project,
) error {
	return nil
}

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
	return nil
}

func (a *ProjectAuthZBasic) CanUnarchiveProject(
	curUser model.User, project *projectv1.Project,
) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &ProjectAuthZBasic{})
}
