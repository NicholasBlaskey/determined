package user

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// ExperimentAuthZ describes authz methods for experiments.
type ExperimentAuthZ interface {
	// GET /api/v1/experiments/:exp_id
	CanGetExperiment(curUser model.User, e *model.Experiment) (canGetExp bool, serverError error)

	// DELETE /api/v1/experiments/:exp_id
	CanDeleteExperiment(curUser model.User, e *model.Experiment) error

	// GET /api/v1/experiments
	FilterExperiments(
		curUser model.User, experiments []*model.Experiment,
	) ([]*model.Experiment, error)

	// TODO (encoding business logic here?!)
	// GET /api/v1/experiments/labels

	// GET /api/v1/experiments/:exp_id/validation_history
	CanGetExperimentValidationHistory(curUser model.User, e *model.Experiment) error

	// POST /api/v1/preview-hp-search
	CanPreviewHPSearch(curUser model.User) error
}

// AuthZProvider is the authz registry for experiments.
var AuthZProvider authz.AuthZProviderType[ExperimentAuthZ]
