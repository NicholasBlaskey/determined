package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

// TODO: rename raw_steps.

// AddTrial adds the trial to the database and sets its ID.
func (db *PgDB) AddTrial(trial *model.Trial) error {
	if trial.ID != 0 {
		return errors.Errorf("error adding a trial with non-zero id %v", trial.ID)
	}

	if err := db.namedGet(&trial.ID, `
INSERT INTO trials
(task_id, request_id, experiment_id, state, start_time, end_time,
hparams, warm_start_checkpoint_id, seed)
VALUES (:task_id, :request_id, :experiment_id, :state, :start_time,
	:end_time, :hparams, :warm_start_checkpoint_id, :seed)
RETURNING id`, trial); err != nil {
		// Assume the foreign key constraint is handled by the database.
		return errors.Wrapf(err, "error inserting trial %v", *trial)
	}

	return nil
}

// TrialByID looks up a trial by ID, returning an error if none exists.
func (db *PgDB) TrialByID(id int) (*model.Trial, error) {
	var trial model.Trial
	err := db.query(`
SELECT id, COALESCE(task_id, '') AS task_id, request_id, experiment_id, state, start_time,
	end_time, hparams, warm_start_checkpoint_id, seed, total_batches
FROM trials
WHERE id = $1`, &trial, id)
	return &trial, errors.Wrapf(err, "error querying for trial %v", id)
}

// TrialByExperimentAndRequestID looks up a trial, returning an error if none exists.
func (db *PgDB) TrialByExperimentAndRequestID(
	experimentID int, requestID model.RequestID,
) (*model.Trial, error) {
	var trial model.Trial
	err := db.query(`
SELECT id, task_id, request_id, experiment_id, state, start_time,
  end_time, hparams, warm_start_checkpoint_id, seed, total_batches
FROM trials
WHERE experiment_id = $1 AND request_id = $2`, &trial, experimentID, requestID)
	return &trial, errors.Wrapf(err, "error querying for trial %v", requestID)
}

// UpdateTrial updates an existing trial. Fields that are nil or zero are not
// updated.  end_time is set if the trial moves to a terminal state.
func (db *PgDB) UpdateTrial(id int, newState model.State) error {
	trial, err := db.TrialByID(id)
	if err != nil {
		return errors.Wrapf(err, "error finding trial %v to update", id)
	}

	if trial.State == newState {
		return nil
	}

	if !model.TrialTransitions[trial.State][newState] {
		return errors.Errorf("illegal transition %v -> %v for trial %v",
			trial.State, newState, trial.ID)
	}
	toUpdate := []string{"state"}
	trial.State = newState
	if model.TerminalStates[newState] {
		now := time.Now().UTC()
		trial.EndTime = &now
		toUpdate = append(toUpdate, "end_time")
	}

	return db.withTransaction("update_trial", func(tx *sqlx.Tx) error {
		// Only the trial actor updates this row, and it does so in a serialized
		// fashion already, so this transaction is more a matter of atomicity.
		if err := namedExecOne(tx, fmt.Sprintf(`
UPDATE trials
%v
WHERE id = :id`, setClause(toUpdate)), trial); err != nil {
			return errors.Wrapf(err, "error updating (%v) in trial %v",
				strings.Join(toUpdate, ", "), id)
		}

		if model.TerminalStates[newState] && trial.EndTime != nil {
			return completeTask(tx, trial.TaskID, *trial.EndTime)
		}

		return nil
	})
}

// UpdateTrialRunnerState updates a trial runner's state.
func (db *PgDB) UpdateTrialRunnerState(id int, state string) error {
	return db.UpdateTrialRunnerMetadata(id, &trialv1.TrialRunnerMetadata{State: state})
}

// UpdateTrialRunnerMetadata updates a trial's metadata about its runner.
func (db *PgDB) UpdateTrialRunnerMetadata(id int, md *trialv1.TrialRunnerMetadata) error {
	if _, err := db.sql.Exec(`
UPDATE trials
SET runner_state = $2
WHERE id = $1`, id, md.State); err != nil {
		return errors.Wrap(err, "saving trial runner state")
	}
	return nil
}

// TrialRunIDAndRestarts returns the run id and restart count for a trial.
func (db *PgDB) TrialRunIDAndRestarts(trialID int) (int, int, error) {
	var runID, restart int
	if err := db.sql.QueryRowx(`
SELECT run_id, restarts
FROM trials
WHERE id = $1`, trialID).Scan(&runID, &restart); err != nil {
		return 0, 0, errors.Wrap(err, "failed to scan trial restart count")
	}
	return runID, restart, nil
}

// UpdateTrialRunID sets the trial's run ID.
func (db *PgDB) UpdateTrialRunID(id, runID int) error {
	if _, err := db.sql.Exec(`
UPDATE trials
SET run_id = $2
WHERE id = $1`, id, runID); err != nil {
		return errors.Wrap(err, "updating trial run id")
	}
	return nil
}

// UpdateTrialRestarts sets the trial's restart count.
func (db *PgDB) UpdateTrialRestarts(id, restartCount int) error {
	if _, err := db.sql.Exec(`
UPDATE trials
SET restarts = $2
WHERE id = $1`, id, restartCount); err != nil {
		return errors.Wrap(err, "updating trial restarts")
	}
	return nil
}

// Now it might be simplest to say hey recompute all metrics.
// In very large trials this could be slow.
// How can we improve this?
//
// fullTrialSummaryMetricsRecompute recomputes all summary metrics for a given trial.
func (db *PgDB) fullTrialSummaryMetricsRecompute(
	ctx context.Context, tx *sqlx.Tx, trialID int,
) error {
	// cool so one question is how are we sure it is only touching that one trial?
	// we can run it on a large db...
	// sounds like a plan.

	// We could also to bun it...
	// Get this ingestion correct then get it smaller?
	// TODO this sql query is excessively long?
	// Kinda a lot to just have a 200 line sql query
	_, err := tx.ExecContext(ctx, `
-- Returns pairs of metric names and trial_ids and if they are numeric or not.
WITH training_trial_metrics as (
SELECT
    name,
    trial_id,
    sum(entries) FILTER (WHERE metric_type != 'number') as nonumbers
FROM (
    SELECT
    name,
    CASE
        WHEN (metrics->'avg_metrics'->name)::text = '"Infinity"'::text THEN 'number'
        WHEN (metrics->'avg_metrics'->name)::text = '"-Infinity"'::text THEN 'number'
        WHEN (metrics->'avg_metrics'->name)::text = '"NaN"'::text THEN 'number'
        ELSE jsonb_typeof(metrics->'avg_metrics'->name)
    END as metric_type,
    trial_id,
    count(1) as entries
    FROM (
        SELECT DISTINCT
        jsonb_object_keys(s.metrics->'avg_metrics') as name
        FROM steps s
        WHERE s.trial_id = $1
    ) names, steps
    JOIN trials ON trial_id = trials.id
    WHERE trials.id = $1
    GROUP BY name, metric_type, trial_id
) typed
where metric_type IS NOT NULL
GROUP BY name, trial_id
ORDER BY trial_id, name
),
-- Filters to only numeric metrics.
training_numeric_trial_metrics as (
SELECT name, trial_id
FROM training_trial_metrics
WHERE nonumbers IS NULL
),
-- Calculates count, sum, min, max on each numeric metric name and trial ID pair.
-- Also adds just the name for non numeric metrics to ensure we record every metric.
training_trial_metric_aggs as (
SELECT
    name,
    ntm.trial_id,
    count(1) as count_agg,
    sum((steps.metrics->'avg_metrics'->>name)::double precision) as sum_agg,
    min((steps.metrics->'avg_metrics'->>name)::double precision) as min_agg,
    max((steps.metrics->'avg_metrics'->>name)::double precision) as max_agg
FROM training_numeric_trial_metrics ntm INNER JOIN steps
ON steps.trial_id=ntm.trial_id
WHERE steps.metrics->'avg_metrics'->name IS NOT NULL
GROUP BY 1, 2
UNION
SELECT
    name,
    trial_id,
    NULL as count_agg,
    NULL as sum,
    NULL as min,
    NULL as max
FROM training_trial_metrics
WHERE nonumbers IS NOT NULL
),
-- Gets the last reported metric for each trial. Note if we report
-- {"a": 1} and {"b": 1} we consider {"b": 1} to be the last reported
-- metric and "a"'s last will be NULL.
latest_training as (
  SELECT s.trial_id,
    unpacked.key as name,
    unpacked.value as latest_value
  FROM (
      SELECT s.*,
        ROW_NUMBER() OVER(
          PARTITION BY s.trial_id
          ORDER BY s.end_time DESC
        ) as rank
      FROM steps s
      JOIN trials ON s.trial_id = trials.id
      WHERE s.trial_id = $1
    ) s, jsonb_each(s.metrics->'avg_metrics') unpacked
  WHERE s.rank = 1
),
-- Adds the last reported metric to training the aggregation.
training_combined_latest_agg as (SELECT
    coalesce(lt.trial_id, tma.trial_id) as trial_id,
    coalesce(lt.name, tma.name) as name,
    tma.count_agg,
    tma.sum_agg,
    tma.min_agg,
    tma.max_agg,
    lt.latest_value
FROM latest_training lt FULL OUTER JOIN training_trial_metric_aggs tma ON
    lt.trial_id = tma.trial_id AND lt.name = tma.name
),
-- Turns each rows into a JSONB object.
training_trial_metrics_final as (
    SELECT
        trial_id, jsonb_collect(jsonb_build_object(
            name, jsonb_build_object(
                'count', count_agg,
                'sum', sum_agg,
                'min', min_agg,
                'max', max_agg,
                'last', latest_value
            )
        )) as training_metrics
    FROM training_combined_latest_agg
    GROUP BY trial_id
),
-- We repeat the same process as above to validation metrics.
validation_trial_metrics as (
SELECT
    name,
    trial_id,
    sum(entries) FILTER (WHERE metric_type != 'number') as nonumbers
FROM (
    SELECT
    name,
    CASE
        WHEN (metrics->'validation_metrics'->name)::text = '"Infinity"'::text THEN 'number'
        WHEN (metrics->'validation_metrics'->name)::text = '"-Infinity"'::text THEN 'number'
        WHEN (metrics->'validation_metrics'->name)::text = '"NaN"'::text THEN 'number'
        ELSE jsonb_typeof(metrics->'validation_metrics'->name)
    END as metric_type,
    trial_id,
    count(1) as entries
    FROM (
        SELECT DISTINCT
        jsonb_object_keys(s.metrics->'validation_metrics') as name
        FROM validations s
        JOIN trials ON s.trial_id = trials.id
        WHERE s.trial_id = $1
    ) names, validations
    JOIN trials ON trial_id = trials.id
    WHERE trials.id = $1
    GROUP BY name, metric_type, trial_id
) typed
where metric_type is not NULL
GROUP BY name, trial_id
ORDER BY trial_id, name
),
validation_numeric_trial_metrics as (
SELECT name, trial_id
FROM validation_trial_metrics
WHERE nonumbers IS NULL
),
validation_trial_metric_aggs as (
SELECT
    name,
    ntm.trial_id,
    count(1) as count_agg,
    sum((validations.metrics->'validation_metrics'->>name)::double precision) as sum_agg,
    min((validations.metrics->'validation_metrics'->>name)::double precision) as min_agg,
    max((validations.metrics->'validation_metrics'->>name)::double precision) as max_agg
FROM validation_numeric_trial_metrics ntm INNER JOIN validations
ON validations.trial_id=ntm.trial_id
WHERE validations.metrics->'validation_metrics'->name IS NOT NULL
GROUP BY 1, 2
UNION
SELECT
    name,
    trial_id,
    NULL as count_agg,
    NULL as sum,
    NULL as min,
    NULL as max
FROM validation_trial_metrics
WHERE nonumbers IS NOT NULL
),
latest_validation as (
    SELECT s.trial_id,
        unpacked.key as name,
        unpacked.value as latest_value
    FROM (
        SELECT s.*,
            ROW_NUMBER() OVER(
                PARTITION BY s.trial_id
                ORDER BY s.end_time DESC
            ) as rank
        FROM validations s
        JOIN trials ON s.trial_id = trials.id
        WHERE s.trial_id = $1
    ) s, jsonb_each(s.metrics->'validation_metrics') unpacked
    WHERE s.rank = 1
),
validation_combined_latest_agg as (SELECT
    coalesce(lt.trial_id, tma.trial_id) as trial_id,
    coalesce(lt.name, tma.name) as name,
    tma.count_agg,
    tma.sum_agg,
    tma.min_agg,
    tma.max_agg,
    lt.latest_value
FROM latest_validation lt FULL OUTER JOIN validation_trial_metric_aggs tma ON
    lt.trial_id = tma.trial_id AND lt.name = tma.name
),
validation_trial_metrics_final as (
    SELECT
        trial_id, jsonb_collect(jsonb_build_object(
            name, jsonb_build_object(
                'count', count_agg,
                'sum', sum_agg,
                'min', min_agg,
                'max', max_agg,
                'last', latest_value
            )
        )) as validation_metrics
    FROM validation_combined_latest_agg
    GROUP BY trial_id
),
-- Combine both training and validation metrics into a single JSON object.
validation_training_combined_json as (
    SELECT
    coalesce(ttm.trial_id, vtm.trial_id) as trial_id,
    (CASE
        WHEN ttm.training_metrics IS NOT NULL AND vtm.validation_metrics IS NOT NULL THEN
            jsonb_build_object(
                'avg_metrics', ttm.training_metrics,
                'validation_metrics', vtm.validation_metrics
            )
        WHEN ttm.training_metrics IS NOT NULL THEN
            jsonb_build_object(
                'avg_metrics', ttm.training_metrics
            )
        WHEN vtm.validation_metrics IS NOT NULL THEN jsonb_build_object(
                'validation_metrics', vtm.validation_metrics
           )
        ELSE '{}'::jsonb END) as summary_metrics
    FROM training_trial_metrics_final ttm FULL OUTER JOIN validation_trial_metrics_final vtm
    ON ttm.trial_id = vtm.trial_id
)
-- Updates trials with this training and validation object.
UPDATE trials SET
    summary_metrics = vtcj.summary_metrics
FROM validation_training_combined_json vtcj WHERE vtcj.trial_id = trials.id;
`, trialID)
	if err != nil {
		return errors.Wrapf(err, "updating trial %d summary metrics", trialID)
	}

	return nil
}

// updateTotalBatches update precomputed total_batches based on existing steps and validations.
func (db *PgDB) updateTotalBatches(ctx context.Context, tx *sqlx.Tx, trialID int) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE trials SET total_batches = sub.new_max_total_batches_processed
		FROM (
			SELECT max(q.total_batches) AS new_max_total_batches_processed
			FROM (
			SELECT coalesce(max(s.total_batches), 0) AS total_batches
			FROM steps s
			WHERE s.trial_id = $1
			UNION ALL
			SELECT coalesce(max(v.total_batches), 0) AS total_batches
			FROM validations v
			WHERE v.trial_id = $1
		) q
		) AS sub;
		`, trialID); err != nil {
		return errors.Wrap(err, "error computing total_batches")
	}
	return nil
}

// AddTrainingMetrics adds a completed step to the database with the given training metrics.
// If these training metrics occur before any others, a rollback is assumed and later
// training and validation metrics are cleaned up.
func (db *PgDB) AddTrainingMetrics(ctx context.Context, m *trialv1.TrialMetrics) error {
	return db.withTransaction("add training metrics", func(tx *sqlx.Tx) error {
		if err := checkTrialRunID(ctx, tx, m.TrialId, m.TrialRunId); err != nil {
			return err
		}

		// Archive all trial_run_ids with batches.
		resT, err := tx.ExecContext(ctx, `
UPDATE raw_steps SET archived = true
WHERE trial_id = $1
  AND trial_run_id < $2
  AND total_batches >= $3;
`, m.TrialId, m.TrialRunId, m.StepsCompleted)
		if err != nil {
			return errors.Wrap(err, "archiving training metrics")
		}

		// Archive all validations with batches.
		resV, err := tx.ExecContext(ctx, `
UPDATE raw_validations SET archived = true
WHERE trial_id = $1
  AND trial_run_id < $2
  AND total_batches > $3;
`, m.TrialId, m.TrialRunId, m.StepsCompleted)
		if err != nil {
			return errors.Wrap(err, "archiving validations")
		}

		if _, err := tx.NamedExecContext(ctx, `
INSERT INTO raw_steps
	(trial_id, trial_run_id, end_time, metrics, total_batches)
VALUES
	(:trial_id, :trial_run_id, now(), :metrics, :total_batches)
`, model.TrialMetrics{
			TrialID:    int(m.TrialId),
			TrialRunID: int(m.TrialRunId),
			Metrics: map[string]interface{}{
				"avg_metrics":   m.Metrics.AvgMetrics,
				"batch_metrics": m.Metrics.BatchMetrics,
			},
			TotalBatches: int(m.StepsCompleted),
		}); err != nil {
			return errors.Wrap(err, "inserting training metrics")
		}

		// Rollback recompute.
		resTRows, err := resT.RowsAffected()
		if err != nil {
			return errors.Wrap(err, "checking for rollback in training metrics")
		}
		resVRows, err := resV.RowsAffected()
		if err != nil {
			return errors.Wrap(err, "checking for rollback in validation metrics")
		}
		if resTRows > 0 || resVRows > 0 {
			if _, err = tx.ExecContext(ctx, `
			SELECT * FROM trials WHERE id = $1 FOR UPDATE; 
		`, m.TrialId); err != nil {
				return errors.Wrap(err, "locking the row with this trial id using select for update")
			}

			if err := db.updateTotalBatches(ctx, tx, int(m.TrialId)); err != nil {
				return errors.Wrap(err, "rollback")
			}

			if err := db.fullTrialSummaryMetricsRecompute(ctx, tx, int(m.TrialId)); err != nil {
				return errors.Wrap(err, "error on rollback compute of summary metrics")
			}

		} else {
			if _, err = tx.ExecContext(ctx, `
		SELECT * FROM trials WHERE id = $1 FOR UPDATE; 
	`, m.TrialId); err != nil {
				return errors.Wrap(err, "locking the row with this trial id using select for update")
			}
			if _, err := tx.ExecContext(ctx, `
UPDATE trials SET total_batches = GREATEST(total_batches, $2)
WHERE id = $1;
`, m.TrialId, m.StepsCompleted); err != nil {
				return errors.Wrap(err, "updating trial total batches")
			}

			// TODO summary metrics incremental.
			if err := db.fullTrialSummaryMetricsRecompute(ctx, tx, int(m.TrialId)); err != nil {
				return errors.Wrap(err, "error on rollback compute of summary metrics")
			}

		}
		return nil
	})
}

// AddValidationMetrics adds a completed validation to the database with the given
// validation metrics. If these validation metrics occur before any others, a rollback
// is assumed and later metrics are cleaned up from the database.
func (db *PgDB) AddValidationMetrics(
	ctx context.Context, m *trialv1.TrialMetrics,
) error {
	return db.withTransaction("add validation metrics", func(tx *sqlx.Tx) error {
		if err := checkTrialRunID(ctx, tx, m.TrialId, m.TrialRunId); err != nil {
			return err
		}

		// QUESTION: why don't we check this with raw_steps as well same as AddTrainingMetrics?
		resV, err := tx.ExecContext(ctx, `
UPDATE raw_validations SET archived = true
WHERE trial_id = $1
  AND trial_run_id < $2
  AND total_batches >= $2;
`, m.TrialId, m.StepsCompleted)
		if err != nil {
			return errors.Wrap(err, "archiving validations")
		}
		if _, err := tx.NamedExecContext(ctx, `
INSERT INTO raw_validations
	(trial_id, trial_run_id, end_time, metrics, total_batches)
VALUES
	(:trial_id, :trial_run_id, now(), :metrics, :total_batches)
`, model.TrialMetrics{
			TrialID:    int(m.TrialId),
			TrialRunID: int(m.TrialRunId),
			Metrics: map[string]interface{}{
				"validation_metrics": m.Metrics.AvgMetrics,
			},
			TotalBatches: int(m.StepsCompleted),
		}); err != nil {
			return errors.Wrap(err, "inserting validation metrics")
		}

		resVRows, err := resV.RowsAffected()
		if err != nil {
			return errors.Wrap(err, "checking for rollback in validation metrics")
		}

		if resVRows > 0 {
			if _, err = tx.ExecContext(ctx, `
			SELECT * FROM trials WHERE id = $1 FOR UPDATE; 
		`, m.TrialId); err != nil {
				return errors.Wrap(err, "locking the row with this trial id using select for update")
			}

			if err := db.updateTotalBatches(ctx, tx, int(m.TrialId)); err != nil {
				return errors.Wrap(err, "rollback")
			}
		} else {
			if _, err = tx.ExecContext(ctx, `
			SELECT * FROM trials WHERE id = $1 FOR UPDATE; 
		`, m.TrialId); err != nil {
				return errors.Wrap(err, "locking the row with this trial id using select for update")
			}
			if _, err := tx.ExecContext(ctx, `
UPDATE trials SET total_batches = GREATEST(total_batches, $2)
WHERE id = $1;
`, m.TrialId, m.StepsCompleted); err != nil {
				return errors.Wrap(err, "updating trial total batches")
			}
		}

		if err := setTrialBestValidation(
			tx, int(m.TrialId),
			int(m.TrialRunId),
			int(m.StepsCompleted)); err != nil {
			return errors.Wrap(err, "updating trial best validation")
		}

		return nil
	})
}

// AddCheckpointMetadata persists metadata for a completed checkpoint to the database.
func AddCheckpointMetadata(ctx context.Context, m *model.CheckpointV2) error {
	var size int64
	for _, v := range m.Resources {
		size += v
	}
	m.Size = size

	err := Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(m).Exec(context.TODO()); err != nil {
			return errors.Wrap(err, "inserting checkpoint")
		}

		if err := UpdateCheckpointSizeTx(ctx, tx, []uuid.UUID{m.UUID}); err != nil {
			return errors.Wrap(err, "updating checkpoint size")
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error adding checkpoint metadata: %w", err)
	}

	return nil
}

func checkTrialRunID(ctx context.Context, tx *sqlx.Tx, trialID, runID int32) error {
	var cRunID int
	switch err := tx.QueryRowxContext(ctx, `
SELECT run_id
FROM trials
WHERE id = $1
`, trialID).Scan(&cRunID); {
	case err != nil:
		return errors.Wrap(err, "querying current run")
	case int(runID) != cRunID:
		return api.AsValidationError("invalid run id, %d (reported) != %d (expected)", runID, cRunID)
	default:
		return nil
	}
}

// ValidationByTotalBatches looks up a validation by trial and total batches,
// returning nil if none exists.
func (db *PgDB) ValidationByTotalBatches(trialID, totalBatches int) (*model.TrialMetrics, error) {
	var validation model.TrialMetrics
	if err := db.query(`
SELECT id, trial_id, total_batches, end_time, metrics
FROM validations
WHERE trial_id = $1
AND total_batches = $2`, &validation, trialID, totalBatches); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for validation (%v, %v)",
			trialID, totalBatches)
	}
	return &validation, nil
}

// CheckpointByTotalBatches looks up a checkpoint by trial and total batch,
// returning nil if none exists.
func (db *PgDB) CheckpointByTotalBatches(trialID, totalBatches int) (*model.Checkpoint, error) {
	var checkpoint model.Checkpoint
	if err := db.query(`
SELECT *
FROM checkpoints_view c
WHERE c.trial_id = $1 AND c.steps_completed = $2`, &checkpoint, trialID, totalBatches,
	); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for checkpoint (%v, %v)",
			trialID, totalBatches)
	}
	return &checkpoint, nil
}

// LatestCheckpointForTrial finds the latest completed checkpoint for a trial, returning nil if
// none exists.
func (db *PgDB) LatestCheckpointForTrial(trialID int) (*model.Checkpoint, error) {
	var checkpoint model.Checkpoint
	if err := db.query(`
SELECT *
FROM checkpoints_view c
WHERE c.trial_id = $1 AND c.state = 'COMPLETED'
ORDER BY c.steps_completed DESC
LIMIT 1`, &checkpoint, trialID); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for latest trial checkpoint (%v)", trialID)
	}
	return &checkpoint, nil
}

// TrialState returns the current state of the given trial.
func (db *PgDB) TrialState(trialID int) (model.State, error) {
	var state model.State
	err := db.sql.QueryRow(`
SELECT state
FROM trials
WHERE id = $1
`, trialID).Scan(&state)
	return state, err
}

// TrialStatus returns the current status of the given trial, including the end time
// without returning all its hparams and other unneeded details. Called in paths hotter
// than TrialByID allows.
func (db *PgDB) TrialStatus(trialID int) (model.State, *time.Time, error) {
	status := struct {
		State   model.State `db:"state"`
		EndTime *time.Time  `db:"end_time"`
	}{}
	err := db.query(`
SELECT state, end_time
FROM trials
WHERE id = $1
`, &status, trialID)
	return status.State, status.EndTime, err
}

// setTrialBestValidation sets `public.trials.best_validation_id` to the `id` of the row in
// `public.validations` corresponding to the trial's best validation.
func setTrialBestValidation(tx *sqlx.Tx, trialID int, trialRunID int, stepsCompleted int) error {
	_, err := tx.Exec(`
WITH const AS (
    SELECT t.id as trial_id,
           config->'searcher'->>'metric' AS metric_name,
           (SELECT
               CASE WHEN coalesce((config->'searcher'->>'smaller_is_better')::boolean, true)
			   THEN 1
			   ELSE -1 END) AS sign
    FROM experiments e
    INNER JOIN trials t ON t.experiment_id = e.id
  	WHERE t.id = $1
), best_validation AS (
	SELECT
		v.id AS id,
		const.sign * (v.metrics->'validation_metrics'->>const.metric_name)::float8 AS metric,
		(v.metrics->'validation_metrics'->>const.metric_name)::float8 AS searcher_metric_value
	FROM (
		SELECT * FROM validations where id = (select best_validation_id from trials where id = $1)
		UNION ALL
		SELECT * FROM validations
			where trial_id = $1
			and trial_run_id = $2
			and total_batches = $3
	) v, const
	WHERE v.trial_id = $1
	ORDER BY metric ASC
	LIMIT 1
)
UPDATE trials t
SET best_validation_id = (SELECT bv.id FROM best_validation bv),
searcher_metric_value = (SELECT bv.searcher_metric_value FROM best_validation bv)
WHERE t.id = $1;
`, trialID, trialRunID, stepsCompleted)
	return errors.Wrapf(err, "error updating best validation for trial %d", trialID)
}
