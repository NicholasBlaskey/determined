package user

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
)

// ExperimentAuthZBasic is basic OSS controls.
type ExperimentAuthZBasic struct{}

/*
func CanGetExperiment(curUser model.User) error {
}
*/

func (a *ExperimentAuthZBasic) CanGetExperiment(
	curUser model.User, e *model.Experiment,
) (canGetExp bool, serverError error) {
	return true, nil
}

func (a *ExperimentAuthZBasic) CanDeleteExperiment(curUser model.User, e *model.Experiment) error {
	curUserIsOwner := e.OwnerID == nil || *e.OwnerID == curUser.ID
	if !curUser.Admin && !curUserIsOwner {
		return fmt.Errorf("non admin users may not delete other user's experiments")
	}
	return nil
}

func (a *ExperimentAuthZBasic) FilterExperiments(
	curUser model.User, experiments []*model.Experiment,
) ([]*model.Experiment, error) {
	return experiments, nil
}

func (a *ExperimentAuthZBasic) CanGetExperimentValidationHistory(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanPreviewHPSearch(curUser model.User) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanActivateExperiment(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanPauseExperiment(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanCancelExperiment(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanKillExperiment(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanArchiveExperiment(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanUnarchiveExperiment(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanSetExperimentsName(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanSetExperimentsNotes(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanSetExperimentsDescription(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanSetExperimentsLabels(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

func (a *ExperimentAuthZBasic) FilterCheckpoints(
	curUser model.User, e *model.Experiment, checkpoints []*checkpointv1.Checkpoint,
) ([]*checkpointv1.Checkpoint, error) {
	return checkpoints, nil
}

func init() {
	AuthZProvider.Register("basic", &ExperimentAuthZBasic{})
}
