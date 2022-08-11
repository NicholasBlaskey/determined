package experiment

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
)

// ExperimentAuthZBasic is basic OSS controls.
type ExperimentAuthZBasic struct{}

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
	curUser model.User, project *projectv1.Project, experiments []*model.Experiment,
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

func (a *ExperimentAuthZBasic) CanCreateExperiment(
	curUser model.User, proj *projectv1.Project, e *model.Experiment,
) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanForkFromExperiment(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanGetMetricNames(curUser model.User, e *model.Experiment) error {
	return nil
}

// ?

func (a *ExperimentAuthZBasic) CanGetMetricBatches(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanGetTrialsSnapshot(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanGetTrialsSample(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanComputeHPImportance(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanGetHPImportance(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanGetBestSearcherValidationMetric(
	curUser model.User, e *model.Experiment,
) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanGetModelDef(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanMoveExperiment(
	curUser model.User, from, to *projectv1.Project, e *model.Experiment,
) error {
	return nil
}

// Do we need to filter the tree?!
func (a *ExperimentAuthZBasic) CanGetModelDefTree(curUser model.User, e *model.Experiment) error {
	return nil
}

func (a *ExperimentAuthZBasic) CanGetModelDefFile(curUser model.User, e *model.Experiment) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &ExperimentAuthZBasic{})
}
