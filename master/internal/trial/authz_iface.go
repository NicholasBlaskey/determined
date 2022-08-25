package trial

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TrialAuthZ describes authz methods for trials.
type TrialAuthZ interface {
	// /api/v1/trials/:trial_id/logs
	CanGetTrialLogs(curUser model.User, trial *model.Trial) error

	// GET /api/v1/trials/:trial_id/checkpoints
	CanGetTrialsCheckpoints(curUser model.User, trial *model.Trial) error

	// POST /api/v1/trials/:trial_id/kill
	CanKillTrial(curUser model.User, trial *model.Trial) error

	// GET /api/v1/trials/:trial_id
	CanGetTrial(curUser model.User, trial *model.Trial) (canGetTrial bool, serverError error)

	// GET /api/v1/trials/:trial_id/summarize
	CanGetTrialSummary(curUser model.User, trial *model.Trial) error

	// GET /api/v1/trials/:trial_id/workloads
	CanGetTrialsWorkloads(curUser model.User, trial *model.Trial) error

	// GET /api/v1/trials/:trial_id/profiler/metrics
	CanGetTrialsProfilerMetrics(curUser model.User, trial *model.Trial) error

	// GET /api/v1/trials/:trial_id/profiler/available_series
	CanGetTrialsProfilerAvailableSeries(curUser model.User, trial *model.Trial) error

	// POST /api/v1/trials/profiler/metrics
	CanPostTrialsProfilerMetricsBatch(curUser model.User, trial *model.Trial) error

	// GET /api/v1/trials/:trial_id/searcher/operation
	CanGetTrialsSearcherOperation(curUser model.User, trial *model.Trial) error

	// POST /api/v1/trials/:trial_id/searcher/completed_operation
	CanCompleteTrialsSearcherValidation(curUser model.User, trial *model.Trial) error

	// POST /api/v1/trials/:trial_id/early_exit
	CanReportTrialsSearcherEarlyExit(curUser model.User, trial *model.Trial) error

	// POST /api/v1/trials/:trial_id/progress
	CanReportTrialsProgress(curUser model.User, trial *model.Trial) error

	// POST /api/v1/trials/:trial_id/training_metrics
	CanReportTrialsTrainingMetrics(curUser model.User, trial *model.Trial) error

	// POST /api/v1/trials/:trial_id/validation_metrics
	CanReportTrialsValidationMetrics(curUser model.User, trial *model.Trial) error

	// POST /api/v1/trials/:trial_id/runner/metadata
	CanPostTrialsRunnerMetadata(curUser model.User, trial *model.Trial) error
}

// AuthZProvider is the authz registry for trials.
var AuthZProvider authz.AuthZProviderType[TrialAuthZ]
