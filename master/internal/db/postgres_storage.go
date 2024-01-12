//nolint:exhaustruct
package db

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type storageBackendRow struct {
	bun.BaseModel `bun:"table:storage_backend"`
	ID            model.StorageBackendID `bun:",pk,autoincrement"`
	SharedFSID    *int                   `bun:"shared_fs_id"`
	S3ID          *int                   `bun:"s3_id"`
	GCSID         *int                   `bun:"gcs_id"`
	AzureID       *int                   `bun:"azure_id"`
	DirectoryID   *int                   `bun:"directory_id"`
}

type storageBackend interface {
	toExpconf() *expconf.CheckpointStorageConfig
	getID() int
}

type storageBackendSharedFS struct {
	bun.BaseModel `bun:"table:storage_backend_shared_fs"`
	ID            int `bun:",pk,autoincrement"`

	HostPath        string  `bun:"host_path"`
	ContainerPath   *string `bun:"container_path"`
	CheckpointPath  *string `bun:"checkpoint_path"`
	TensorboardPath *string `bun:"tensorboard_path"`
	StoragePath     *string `bun:"storage_path"`
	Propagation     string  `bun:"propagation"`
}

func (s *storageBackendSharedFS) getID() int {
	return s.ID
}

func (s *storageBackendSharedFS) toExpconf() *expconf.CheckpointStorageConfig {
	return &expconf.CheckpointStorageConfig{
		RawSharedFSConfig: &expconf.SharedFSConfig{
			RawHostPath:        ptrs.Ptr(s.HostPath),
			RawContainerPath:   s.ContainerPath,
			RawCheckpointPath:  s.CheckpointPath,
			RawTensorboardPath: s.TensorboardPath,
			RawStoragePath:     s.StoragePath,
			RawPropagation:     ptrs.Ptr(s.Propagation),
		},
	}
}

func expconfToStorage(cs *expconf.CheckpointStorageConfig) (storageBackend, string) {
	switch storage := cs.GetUnionMember().(type) {
	case expconf.SharedFSConfig:
		return &storageBackendSharedFS{
			HostPath:        storage.HostPath(),
			ContainerPath:   storage.ContainerPath(),
			CheckpointPath:  storage.CheckpointPath(),
			TensorboardPath: storage.TensorboardPath(),
			StoragePath:     storage.StoragePath(),
			Propagation:     storage.Propagation(),
		}, "shared_fs_id"
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

	childTableRow, unionType := expconfToStorage(cs)
	// q := idb.NewInsert().Returning("id").On("CONFLICT DO NOTHING").Model(childTableRow).Exec(ctx)

	/*
		switch r := childTableRow.(type) {
		case *storageBackendSharedFS:
			q = q.Model(r)
		}
	*/
	if _, err := idb.NewInsert().Returning("id").
		On("CONFLICT DO NOTHING").
		Model(childTableRow).
		Exec(ctx); err != nil {
		json, jsonErr := cs.MarshalJSON()
		if jsonErr != nil {
			return 0, fmt.Errorf("adding storage backend: %w: %w", jsonErr, err)
		}
		return 0, fmt.Errorf("adding storage backend %s: %w", string(json), err)
	}

	// ON CONFLICT DO NOTHING returns a non zero ID only when we insert a new row.
	// When we insert a new row also insert a new row in th parent table.
	if childTableRow.getID() != 0 {
		unionTableRow := &storageBackendRow{}
		if _, err := idb.NewInsert().Model(unionTableRow).
			Value(unionType, "?", childTableRow.getID()).
			Returning("id").
			Exec(ctx); err != nil {
			return 0, fmt.Errorf("adding storage backend row: %w", err)
		}

		return unionTableRow.ID, nil
	}

	// This case we have already inserted a parent and a child row.
	// First do a lookup for the child row then do another lookup of the parent.
	childBackendID, err := getChildBackendRows(ctx, idb, childTableRow)
	if err != nil {
		return 0, fmt.Errorf("getting child backend row in dupe case %v: %w", childTableRow, err)
	}

	unionTableRow := &storageBackendRow{}
	if err := idb.NewSelect().Model(unionTableRow).
		Where("? = ?", bun.Safe(unionType), childBackendID).
		Scan(ctx, unionTableRow); err != nil {
		return 0, fmt.Errorf("getting parent backend row in dupe case %v: %w", childTableRow, err)
	}

	return unionTableRow.ID, nil
}

func getChildBackendRows(ctx context.Context, idb bun.IDB, backend storageBackend) (int, error) {
	fmt.Println("WORKING?")
	q := getChildBackendRowsQuery(idb, backend)
	if err := q.Scan(ctx, backend); err != nil {
		return 0, fmt.Errorf("running storage child lookup query: %w", err)
	}
	fmt.Println("WORKING ID?", backend.getID())

	return backend.getID(), nil
}

func getChildBackendRowsQuery(idb bun.IDB, backend storageBackend) *bun.SelectQuery {
	q := idb.NewSelect().Model(backend).Column("id")

	addStringPtrWhere := func(colName string, v *string) {
		if v != nil {
			q = q.Where("? = ?", bun.Safe(colName), *v)
		} else {
			q = q.Where("? IS NULL", bun.Safe(colName))
		}
	}

	switch b := backend.(type) {
	case *storageBackendSharedFS:
		q = q.Where("host_path = ?", b.HostPath)
		addStringPtrWhere("container_path", b.ContainerPath)
		addStringPtrWhere("checkpoint_path", b.CheckpointPath)
		addStringPtrWhere("tensorboard_path", b.TensorboardPath)
		addStringPtrWhere("storage_path", b.StoragePath)
		q = q.Where("propagation = ?", b.Propagation)
	}

	return q
}

// StorageBackend returns the checkpoint storage backend information.
// We won't return the "save_*_best" fields on the expconf struct.
func StorageBackend(
	ctx context.Context, idb bun.IDB, id model.StorageBackendID,
) (*expconf.CheckpointStorageConfig, error) {
	var parentRow storageBackendRow
	if err := idb.NewSelect().Model(&parentRow).
		Where("id = ?", id).
		Scan(ctx, &parentRow); err != nil {
		return nil, fmt.Errorf("getting storage backend ID %d: %w", id, err)
	}

	q := idb.NewSelect()
	var err error
	var childRow storageBackend
	switch {
	case parentRow.SharedFSID != nil:
		r := &storageBackendSharedFS{}
		childRow = r
		err = q.Model(r).Where("id = ?", *parentRow.SharedFSID).Scan(ctx, r)
	default:
		panic(fmt.Sprintf("expected one of parentRow to be nil %+v", parentRow))
	}
	if err != nil {
		return nil, fmt.Errorf("getting child of storage backend ID %d: %w", id, err)
	}

	return childRow.toExpconf(), nil
}

/*
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
*/

/*
func storageIDByConfig(
	ctx context.Context, idb bun.IDB, cs *expconf.CheckpointStorageConfig,
) (model.StorageBackendID, error) {
	backend := expconfToStorageBackend(cs)
	if err := storageIDByConfigQuery(idb, &backend).Scan(ctx, &backend); err != nil {
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
}
*/

/*
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
*/

/*
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


// This is written as a seperate function so we can test this.
func storageIDByConfigQuery(
	idb bun.IDB, backend *storageBackend,
) *bun.SelectQuery {
	q := idb.NewSelect().Model(backend).Column("id").Where("type = ?", backend.Type)
	addStringPtrWhere := func(colName string, v *string) {
		if v != nil {
			q = q.Where("? = ?", bun.Safe(colName), *v)
		} else {
			q = q.Where("? IS NULL", bun.Safe(colName))
		}
	}

	addStringPtrWhere("shared_fs_host_path", backend.SharedFSHostPath)
	addStringPtrWhere("shared_fs_container_path", backend.SharedFSContainerPath)
	addStringPtrWhere("shared_fs_checkpoint_path", backend.SharedFSCheckpointPath)
	addStringPtrWhere("shared_fs_tensorboard_path", backend.SharedFSTensorboardPath)
	addStringPtrWhere("shared_fs_storage_path", backend.SharedFSStoragePath)
	addStringPtrWhere("shared_fs_propagation", backend.SharedFSPropagation)

	addStringPtrWhere("s3_bucket", backend.S3Bucket)
	addStringPtrWhere("s3_access_key", backend.S3AccessKey)
	addStringPtrWhere("s3_secret_key", backend.S3SecretKey)
	addStringPtrWhere("s3_endpoint_url", backend.S3EndpointURL)
	addStringPtrWhere("s3_prefix", backend.S3Prefix)

	addStringPtrWhere("gcs_bucket", backend.GCSBucket)
	addStringPtrWhere("gcs_prefix", backend.GCSPrefix)

	addStringPtrWhere("azure_container", backend.AzureContainer)
	addStringPtrWhere("azure_connection_string", backend.AzureConnectionString)
	addStringPtrWhere("azure_account_url", backend.AzureAccountURL)
	addStringPtrWhere("azure_credential", backend.AzureCredential)

	addStringPtrWhere("directory_container_path", backend.DirectoryContainerPath)

	return q
}

*/
