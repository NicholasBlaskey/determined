package user

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// ExperimentAuthZ describes authz methods for experiments.
type ExperimentAuthZ interface {
	// CanGetExperiment(model.User) error
	// TODO canGetExperiment is tough

	CanDeleteExperiment(curUser model.User, e *model.Experiment) error
}

// AuthZProvider is the authz registry for experiments.
var AuthZProvider authz.AuthZProviderType[ExperimentAuthZ]
