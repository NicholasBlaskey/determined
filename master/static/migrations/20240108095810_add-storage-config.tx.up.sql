CREATE TYPE checkpoint_storage_type AS ENUM (
  'shared_fs',
  's3'
);

CREATE TABLE storage_backend (
  id integer PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
  type checkpoint_storage_type NOT NULL,

  shared_fs_host_path        TEXT,
  shared_fs_container_path   TEXT,
  shared_fs_checkpoint_path  TEXT,
  shared_fs_tensorboard_path TEXT,
  shared_fs_storage_path     TEXT,
  shared_fs_propagation      TEXT,
  CONSTRAINT shared_fs_required_fields CHECK (type = 'shared_fs' AND (
      shared_fs_host_path IS NOT NULL AND
      shared_fs_propagation IS NOT NULL
  ))
);

-- Hack for lack of pre-postgres 15 NULLS NOT DISTINCT.
-- https://stackoverflow.com/questions/8289100/create-unique-constraint-with-null-columns
CREATE UNIQUE INDEX ix_storage_backend_shared_fs_unique ON storage_backend (
  COALESCE(shared_fs_host_path, 'DeterminedReservedNullUniqueValue'),
  COALESCE(shared_fs_container_path, 'DeterminedReservedNullUniqueValue'),
  COALESCE(shared_fs_checkpoint_path, 'DeterminedReservedNullUniqueValue'),
  COALESCE(shared_fs_tensorboard_path, 'DeterminedReservedNullUniqueValue'),
  COALESCE(shared_fs_storage_path, 'DeterminedReservedNullUniqueValue'),
  COALESCE(shared_fs_propagation, 'DeterminedReservedNullUniqueValue')
);

ALTER TABLE checkpoints_v2
  ADD COLUMN storage_id int REFERENCES storage_backend(id);

DROP VIEW proto_checkpoints_view;
DROP VIEW checkpoints_view;

CREATE OR REPLACE VIEW checkpoints_view AS
    SELECT
        c.id AS id,
        c.uuid AS uuid,
        c.task_id,
        c.allocation_id,
        c.report_time,
        c.state,
        c.resources,
        c.metadata,
        t.id AS trial_id,
        e.id AS experiment_id,
        e.config AS experiment_config,
        t.hparams AS hparams,
        s.metrics AS training_metrics,
        v.metrics->'validation_metrics' AS validation_metrics,
        (v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8 AS searcher_metric,
        CAST(c.metadata->>'steps_completed' AS int) as steps_completed,
        c.size,
        c.storage_id
    FROM checkpoints_v2 AS c
    LEFT JOIN trial_id_task_id AS task ON c.task_id = task.task_id
    LEFT JOIN trials AS t on t.id = task.trial_id
    LEFT JOIN experiments AS e on t.experiment_id = e.id
    LEFT JOIN raw_validations AS v on CAST(c.metadata->>'steps_completed' AS int) = v.total_batches and t.id = v.trial_id AND NOT v.archived
    LEFT JOIN raw_steps AS s on CAST(c.metadata->>'steps_completed' AS int) = s.total_batches and t.id = s.trial_id AND NOT s.archived;

CREATE OR REPLACE VIEW proto_checkpoints_view AS
    SELECT
        c.uuid::text AS uuid,
        c.task_id,
        c.allocation_id,
        c.report_time as report_time,
        'STATE_' || c.state AS state,
        c.resources,
        c.metadata,
        c.storage_id,
        -- Build a training substruct for protobuf.
        jsonb_build_object(
            'trial_id', c.trial_id,
            'experiment_id', c.experiment_id,
            'experiment_config', c.experiment_config,
            'hparams', c.hparams,
            -- construct training metrics from the untyped jsonb deterministically, since older
            -- versions may have old keys (e.g., num_inputs) and our unmarshaling is strict.
            'training_metrics', jsonb_build_object(
                'avg_metrics', c.training_metrics->'avg_metrics',
                'batch_metrics', c.training_metrics->'batch_metrics'
            ),
            'validation_metrics', json_build_object('avg_metrics', c.validation_metrics),
            'searcher_metric', c.searcher_metric
        ) AS training
    FROM checkpoints_view AS c;;
