package trial

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// TrialAuthZBasic is basic OSS controls.
type TrialAuthZBasic struct{}

func (a *TrialAuthZBasic) CanGetTrial(
	curUser model.User, trial *model.Trial,
) (canGetTrial bool, serverError error) {
	return true, nil
}

func init() {
	AuthZProvider.Register("basic", &TrialAuthZBasic{})
}
