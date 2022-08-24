package trial

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TrialAuthZ describes authz methods for trials.
type TrialAuthZ interface {
	// TODO
	CanGetTrial(curUser model.User, trial *model.Trial) (canGetTrial bool, serverError error)
}

// AuthZProvider is the authz registry for trials.
var AuthZProvider authz.AuthZProviderType[TrialAuthZ]
