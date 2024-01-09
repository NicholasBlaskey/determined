CREATE TYPE checkpoint_storage_type AS ENUM (
  'shared_fs',
  's3'
);

-- Default fields work the same as required right?
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
