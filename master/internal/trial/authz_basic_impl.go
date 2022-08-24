package trial

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// TrialAuthZBasic is basic OSS controls.
type TrialAuthZBasic struct{}

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

func (a *TrialAuthZBasic) CanGetTrialSummary(curUser model.User, trial *model.Trial) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &TrialAuthZBasic{})
}
