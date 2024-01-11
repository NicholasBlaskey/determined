package db

import (
	"context"
	"fmt"
	"reflect"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type storageBackend struct {
	bun.BaseModel `bun:"table:storage_backend"`
	ID            model.StorageBackendID `bun:",pk,autoincrement"`
	Type          model.StorageType      `bun:"type"`

	SharedFSHostPath        *string `bun:"shared_fs_host_path"`
	SharedFSContainerPath   *string `bun:"shared_fs_container_path"`
	SharedFSCheckpointPath  *string `bun:"shared_fs_checkpoint_path"`
	SharedFSTensorboardPath *string `bun:"shared_fs_tensorboard_path"`
	SharedFSStoragePath     *string `bun:"shared_fs_storage_path"`
	SharedFSPropagation     *string `bun:"shared_fs_propagation"`

	S3Bucket      *string `bun:"s3_bucket"`
	S3AccessKey   *string `bun:"s3_access_key"`
	S3SecretKey   *string `bun:"s3_secret_key"`
	S3EndpointURL *string `bun:"s3_endpoint_url"`
	S3Prefix      *string `bun:"s3_prefix"`

	GCSBucket *string `bun:"gcs_bucket"`
	GCSPrefix *string `bun:"gcs_prefix"`

	AzureContainer        *string `bun:"azure_container"`
	AzureConnectionString *string `bun:"azure_connection_string"`
	AzureAccountURL       *string `bun:"azure_account_url"`
	AzureCredential       *string `bun:"azure_credential"`

	DirectoryContainerPath *string `bun:"directory_container_path"`
}

//nolint:exhaustruct
func expconfToStorageBackend(cs *expconf.CheckpointStorageConfig) storageBackend {
	switch storage := cs.GetUnionMember().(type) {
	case expconf.SharedFSConfig:
		return storageBackend{
			Type:                    model.SharedFSStorageType,
			SharedFSHostPath:        ptrs.Ptr(storage.HostPath()),
			SharedFSContainerPath:   storage.ContainerPath(),
			SharedFSCheckpointPath:  storage.CheckpointPath(),
			SharedFSTensorboardPath: storage.TensorboardPath(),
			SharedFSStoragePath:     storage.StoragePath(),
			SharedFSPropagation:     ptrs.Ptr(storage.Propagation()),
		}
	case expconf.S3Config:
		return storageBackend{
			Type:          model.S3StorageType,
			S3Bucket:      ptrs.Ptr(storage.Bucket()),
			S3AccessKey:   storage.AccessKey(),
			S3SecretKey:   storage.SecretKey(),
			S3EndpointURL: storage.EndpointURL(),
			S3Prefix:      storage.Prefix(),
		}
	case expconf.GCSConfig:
		return storageBackend{
			Type:      model.GCSStorageType,
			GCSBucket: ptrs.Ptr(storage.Bucket()),
			GCSPrefix: storage.Prefix(),
		}
	case expconf.AzureConfig:
		return storageBackend{
			Type:                  model.AzureStorageType,
			AzureContainer:        ptrs.Ptr(storage.Container()),
			AzureConnectionString: storage.ConnectionString(),
			AzureAccountURL:       storage.AccountURL(),
			AzureCredential:       storage.Credential(),
		}
	case expconf.DirectoryConfig:
		return storageBackend{
			Type:                   model.DirectoryStorageType,
			DirectoryContainerPath: ptrs.Ptr(storage.ContainerPath()),
		}
	default:
		panic(fmt.Sprintf("unknown type converting expconf to storage backend %T", storage))
	}
}

// AddStorageBackend adds storage backend information.
// We won't persist the "save_*_best" fields on the expconf struct.
// If the same storage backend has been persisted before then we will return the ID and
// not insert another row.
func AddStorageBackend(
	ctx context.Context, idb bun.IDB, cs *expconf.CheckpointStorageConfig,
) (model.StorageBackendID, error) {
	cs = schemas.WithDefaults(cs)
	if err := schemas.IsComplete(cs); err != nil {
		return 0, fmt.Errorf("schema is not complete: %w", err)
	}

	backend := expconfToStorageBackend(cs)
	if _, err := idb.NewInsert().Model(&backend).
		Returning("id").
		On("CONFLICT DO NOTHING").
		Exec(ctx); err != nil {
		json, jsonErr := cs.MarshalJSON()
		if jsonErr != nil {
			return 0, fmt.Errorf("adding storage backend: %w: %w", jsonErr, err)
		}
		return 0, fmt.Errorf("adding storage backend %s: %w", string(json), err)
	}

	// "ON CONFLICT DO NOTHING" won't return ID so just look it up.
	// We could hack in "ON CONFLICT DO UPDATE" but that would require listing
	// all unique possibilities.
	// Other solutions look worse than what we have.
	// https://stackoverflow.com/questions/34708509/how-to-use-returning-with-on-conflict-in-postgresql/37543015#37543015
	if backend.ID == 0 {
		id, err := storageIDByConfig(ctx, idb, cs)
		if err != nil {
			return 0, fmt.Errorf("looking up storage ID since it already exists: %w", err)
		}

		return id, nil
	}

	return backend.ID, nil
}

func storageIDByConfig(
	ctx context.Context, idb bun.IDB, cs *expconf.CheckpointStorageConfig,
) (model.StorageBackendID, error) {
	backend := expconfToStorageBackend(cs)
	q := idb.NewSelect().Model(&backend).Column("id")

	// This is really gross but the alternative of listing every column seems worse from a
	// bug potential standpoint.
	valueOfStruct := reflect.ValueOf(backend)
	typeOfStruct := reflect.TypeOf(backend)
	for i := 0; i < valueOfStruct.NumField(); i++ {
		fieldValue := valueOfStruct.Field(i)
		fieldType := typeOfStruct.Field(i)

		tag := fieldType.Tag.Get("bun")
		if fieldValue.IsValid() {
			switch v := fieldValue.Interface().(type) {
			case *string:
				if v == nil {
					q = q.Where("? IS NULL", bun.Safe(tag))
				} else {
					q = q.Where("? = ?", bun.Safe(tag), *v)
				}
			case model.StorageType:
				q = q.Where("? = ?", bun.Safe(tag), v)
			case bun.BaseModel, model.StorageBackendID:
			default:
				panic(fmt.Sprintf("unknown field type %T", fieldValue.Interface()))
			}
		}
	}

	if err := q.Scan(ctx, &backend); err != nil {
		return 0, fmt.Errorf("looking up storage backend by config: %w", err)
	}

	return backend.ID, nil
}

// StorageBackend returns the checkpoint storage backend information.
// We won't return the "save_*_best" fields on the expconf struct.
//
//nolint:exhaustruct
func StorageBackend(
	ctx context.Context, idb bun.IDB, id model.StorageBackendID,
) (*expconf.CheckpointStorageConfig, error) {
	var backend storageBackend
	if err := idb.NewSelect().Model(&backend).
		Where("id = ?", id).
		Scan(ctx, &backend); err != nil {
		return nil, fmt.Errorf("getting storage backend ID %d: %w", id, err)
	}

	switch backend.Type {
	case model.SharedFSStorageType:
		return &expconf.CheckpointStorageConfig{
			RawSharedFSConfig: &expconf.SharedFSConfig{
				RawHostPath:        backend.SharedFSHostPath,
				RawContainerPath:   backend.SharedFSContainerPath,
				RawCheckpointPath:  backend.SharedFSCheckpointPath,
				RawTensorboardPath: backend.SharedFSTensorboardPath,
				RawStoragePath:     backend.SharedFSStoragePath,
				RawPropagation:     backend.SharedFSPropagation,
			},
		}, nil
	case model.S3StorageType:
		return &expconf.CheckpointStorageConfig{
			RawS3Config: &expconf.S3Config{
				RawBucket:      backend.S3Bucket,
				RawAccessKey:   backend.S3AccessKey,
				RawSecretKey:   backend.S3SecretKey,
				RawEndpointURL: backend.S3EndpointURL,
				RawPrefix:      backend.S3Prefix,
			},
		}, nil
	case model.GCSStorageType:
		return &expconf.CheckpointStorageConfig{
			RawGCSConfig: &expconf.GCSConfig{
				RawBucket: backend.GCSBucket,
				RawPrefix: backend.GCSPrefix,
			},
		}, nil
	case model.AzureStorageType:
		return &expconf.CheckpointStorageConfig{
			RawAzureConfig: &expconf.AzureConfig{
				RawContainer:        backend.AzureContainer,
				RawConnectionString: backend.AzureConnectionString,
				RawAccountURL:       backend.AzureAccountURL,
				RawCredential:       backend.AzureCredential,
			},
		}, nil
	case model.DirectoryStorageType:
		return &expconf.CheckpointStorageConfig{
			RawDirectoryConfig: &expconf.DirectoryConfig{
				RawContainerPath: backend.DirectoryContainerPath,
			},
		}, nil
	default:
		return nil, fmt.Errorf("got unexpected backendType %s for storageID %d",
			backend.Type, id)
	}

	// TODO should we validate? Kinda think we shouldn't???
	// I guess depends on backwards validation checks.
}
