package trial

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TrialAuthZ describes authz methods for trials.
type TrialAuthZ interface {
	// TODO above

	// /api/v1/trials/:trial_id/checkpoints
	CanGetTrialsCheckpoints(curUser model.User, trial *model.Trial) error

	// /api/v1/trials/:trial_id/kill
	CanKillTrial(curUser model.User, trial *model.Trial) error

	// /api/v1/trials/:trial_id
	CanGetTrial(curUser model.User, trial *model.Trial) (canGetTrial bool, serverError error)

	// /api/v1/trials/:trial_id/summarize
	CanGetTrialSummary(curUser model.User, trial *model.Trial) error
}

// AuthZProvider is the authz registry for trials.
var AuthZProvider authz.AuthZProviderType[TrialAuthZ]
