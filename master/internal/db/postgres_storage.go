//nolint:exhaustruct
package db

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
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

func (p *storageBackendRow) toChildRowOnlyIDPopulated() storageBackend {
	switch {
	case p.SharedFSID != nil:
		return &storageBackendSharedFS{ID: *p.SharedFSID}
	case p.S3ID != nil:
		return &storageBackendS3{ID: *p.S3ID}
	case p.GCSID != nil:
		return &storageBackendGCS{ID: *p.GCSID}
	case p.AzureID != nil:
		return &storageBackendAzure{ID: *p.AzureID}
	case p.DirectoryID != nil:
		return &storageBackendDirectory{ID: *p.DirectoryID}
	default:
		panic(fmt.Sprintf("expected one of p to be nil %+v", p))
	}
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
			RawHostPath:        &s.HostPath,
			RawContainerPath:   s.ContainerPath,
			RawCheckpointPath:  s.CheckpointPath,
			RawTensorboardPath: s.TensorboardPath,
			RawStoragePath:     s.StoragePath,
			RawPropagation:     &s.Propagation,
		},
	}
}

type storageBackendS3 struct {
	bun.BaseModel `bun:"table:storage_backend_s3"`
	ID            int `bun:",pk,autoincrement"`

	Bucket      string  `bun:"bucket"`
	AccessKey   *string `bun:"access_key"`
	SecretKey   *string `bun:"secret_key"`
	EndpointURL *string `bun:"endpoint_url"`
	Prefix      *string `bun:"prefix"`
}

func (s *storageBackendS3) getID() int {
	return s.ID
}

func (s *storageBackendS3) toExpconf() *expconf.CheckpointStorageConfig {
	return &expconf.CheckpointStorageConfig{
		RawS3Config: &expconf.S3Config{
			RawBucket:      &s.Bucket,
			RawAccessKey:   s.AccessKey,
			RawSecretKey:   s.SecretKey,
			RawEndpointURL: s.EndpointURL,
			RawPrefix:      s.Prefix,
		},
	}
}

type storageBackendGCS struct {
	bun.BaseModel `bun:"table:storage_backend_gcs"`
	ID            int `bun:",pk,autoincrement"`

	Bucket string  `bun:"bucket"`
	Prefix *string `bun:"prefix"`
}

func (s *storageBackendGCS) getID() int {
	return s.ID
}

func (s *storageBackendGCS) toExpconf() *expconf.CheckpointStorageConfig {
	return &expconf.CheckpointStorageConfig{
		RawGCSConfig: &expconf.GCSConfig{
			RawBucket: &s.Bucket,
			RawPrefix: s.Prefix,
		},
	}
}

type storageBackendAzure struct {
	bun.BaseModel `bun:"table:storage_backend_azure"`
	ID            int `bun:",pk,autoincrement"`

	Container        string  `bun:"container"`
	ConnectionString *string `bun:"connection_string"`
	AccountURL       *string `bun:"account_url"`
	Credential       *string `bun:"credential"`
}

func (s *storageBackendAzure) getID() int {
	return s.ID
}

func (s *storageBackendAzure) toExpconf() *expconf.CheckpointStorageConfig {
	return &expconf.CheckpointStorageConfig{
		RawAzureConfig: &expconf.AzureConfig{
			RawContainer:        &s.Container,
			RawConnectionString: s.ConnectionString,
			RawAccountURL:       s.AccountURL,
			RawCredential:       s.Credential,
		},
	}
}

type storageBackendDirectory struct {
	bun.BaseModel `bun:"table:storage_backend_directory"`
	ID            int `bun:",pk,autoincrement"`

	ContainerPath string `bun:"container_path"`
}

func (s *storageBackendDirectory) getID() int {
	return s.ID
}

func (s *storageBackendDirectory) toExpconf() *expconf.CheckpointStorageConfig {
	return &expconf.CheckpointStorageConfig{
		RawDirectoryConfig: &expconf.DirectoryConfig{
			RawContainerPath: &s.ContainerPath,
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
	case expconf.S3Config:
		return &storageBackendS3{
			Bucket:      storage.Bucket(),
			AccessKey:   storage.AccessKey(),
			SecretKey:   storage.SecretKey(),
			EndpointURL: storage.EndpointURL(),
			Prefix:      storage.Prefix(),
		}, "s3_id"
	case expconf.GCSConfig:
		return &storageBackendGCS{
			Bucket: storage.Bucket(),
			Prefix: storage.Prefix(),
		}, "gcs_id"
	case expconf.AzureConfig:
		return &storageBackendAzure{
			Container:        storage.Container(),
			ConnectionString: storage.ConnectionString(),
			AccountURL:       storage.AccountURL(),
			Credential:       storage.Credential(),
		}, "azure_id"
	case expconf.DirectoryConfig:
		return &storageBackendDirectory{
			ContainerPath: storage.ContainerPath(),
		}, "directory_id"
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
	q := idb.NewSelect().Model(backend).Column("id")
	wheres, args := getChildBackendRowWheres(backend)
	for i := 0; i < len(wheres); i++ {
		q.Where(wheres[i], args[i]...)
	}

	if err := q.Scan(ctx, backend); err != nil {
		return 0, fmt.Errorf("running storage child lookup query: %w", err)
	}
	return backend.getID(), nil
}

// This is written like this so we can easily test this. Without testing the query it
// is really hard to generate test cases that will error if someone forgets to add a column here.
// Returning a *bun.SelectQuery is a good idea but harder to test, since the Where order can
// make the query generate differently.
func getChildBackendRowWheres(backend storageBackend) ([]string, [][]any) {
	var wheres []string
	var args [][]any

	addStringWhere := func(colName, v string) {
		wheres = append(wheres, "? = ?")
		args = append(args, []any{bun.Safe(colName), v})
	}
	addStringPtrWhere := func(colName string, v *string) {
		if v != nil {
			addStringWhere(colName, *v)
		} else {
			wheres = append(wheres, "? IS NULL")
			args = append(args, []any{bun.Safe(colName)})
		}
	}

	switch b := backend.(type) {
	case *storageBackendSharedFS:
		addStringWhere("host_path", b.HostPath)
		addStringPtrWhere("container_path", b.ContainerPath)
		addStringPtrWhere("checkpoint_path", b.CheckpointPath)
		addStringPtrWhere("tensorboard_path", b.TensorboardPath)
		addStringPtrWhere("storage_path", b.StoragePath)
		addStringWhere("propagation", b.Propagation)
	case *storageBackendS3:
		addStringWhere("bucket", b.Bucket)
		addStringPtrWhere("access_key", b.AccessKey)
		addStringPtrWhere("secret_key", b.SecretKey)
		addStringPtrWhere("endpoint_url", b.EndpointURL)
		addStringPtrWhere("prefix", b.Prefix)
	case *storageBackendGCS:
		addStringWhere("bucket", b.Bucket)
		addStringPtrWhere("prefix", b.Prefix)
	case *storageBackendAzure:
		addStringWhere("container", b.Container)
		addStringPtrWhere("connection_string", b.ConnectionString)
		addStringPtrWhere("account_url", b.AccountURL)
		addStringPtrWhere("credential", b.Credential)
	case *storageBackendDirectory:
		addStringWhere("container_path", b.ContainerPath)
	}

	return wheres, args
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

	childRow := parentRow.toChildRowOnlyIDPopulated()
	if err := idb.NewSelect().Model(childRow).
		Where("id = ?", childRow.getID()).
		Scan(ctx, childRow); err != nil {
		return nil, fmt.Errorf("getting child of storage backend ID %d: %w", id, err)
	}

	return childRow.toExpconf(), nil
}
