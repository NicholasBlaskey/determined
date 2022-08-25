package trial

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// TrialAuthZBasic is basic OSS controls.
type TrialAuthZBasic struct{}

func (a *TrialAuthZBasic) CanGetTrialLogs(curUser model.User, trial *model.Trial) error {
	return nil
}

func (a *TrialAuthZBasic) CanGetTrialsCheckpoints(curUser model.User, trial *model.Trial) error {
	return nil
}

func (a *TrialAuthZBasic) CanKillTrial(curUser model.User, trial *model.Trial) error {
	return nil
}

func (a *TrialAuthZBasic) CanGetTrial(
	curUser model.User, trial *model.Trial,
) (canGetTrial bool, serverError error) {
	return true, nil
}

func (a *TrialAuthZBasic) CanGetTrialsSummary(curUser model.User, trial *model.Trial) error {
	return nil
}

func (a *TrialAuthZBasic) CanGetTrialsWorkloads(curUser model.User, trial *model.Trial) error {
	return nil
}

func (a *TrialAuthZBasic) CanGetTrialsProfilerMetrics(
	curUser model.User, trial *model.Trial,
) error {
	return nil
}

func (a *TrialAuthZBasic) CanGetTrialsProfilerAvailableSeries(
	curUser model.User, trial *model.Trial,
) error {
	return nil
}

func (a *TrialAuthZBasic) CanPostTrialsProfilerMetricsBatch(
	curUser model.User, trial *model.Trial,
) error {
	return nil
}

func (a *TrialAuthZBasic) CanGetTrialsSearcherOperation(
	curUser model.User, trial *model.Trial,
) error {
	return nil
}

func (a *TrialAuthZBasic) CanCompleteTrialsSearcherValidation(
	curUser model.User, trial *model.Trial,
) error {
	return nil
}

func (a *TrialAuthZBasic) CanReportTrialsSearcherEarlyExit(
	curUser model.User, trial *model.Trial,
) error {
	return nil
}

func (a *TrialAuthZBasic) CanReportTrialsProgress(curUser model.User, trial *model.Trial) error {
	return nil
}

func (a *TrialAuthZBasic) CanReportTrialsTrainingMetrics(
	curUser model.User, trial *model.Trial,
) error {
	return nil
}

func (a *TrialAuthZBasic) CanReportTrialsValidationMetrics(
	curUser model.User, trial *model.Trial,
) error {
	return nil
}

func (a *TrialAuthZBasic) CanPostTrialsRunnerMetadata(curUser model.User, trial *model.Trial) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &TrialAuthZBasic{})
}
