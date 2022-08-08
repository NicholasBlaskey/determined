package user

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// ExperimentAuthZ describes authz methods for experiments.
type ExperimentAuthZ interface {
	// /api/v1/experiments/:exp_id
	CanGetExperiment(curUser model.User, e *model.Experiment) (canGetExp bool, serverError error)

	// /api/v1/experiments/:exp_id
	CanDeleteExperiment(curUser model.User, e *model.Experiment) error

	// /api/v1/experiments
	FilterExperiments(
		curUser model.User, experiments []*model.Experiment,
	) ([]*model.Experiment, error)
}

// AuthZProvider is the authz registry for experiments.
var AuthZProvider authz.AuthZProviderType[ExperimentAuthZ]
