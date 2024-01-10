package model

// StorageBackendID is the ID for the storage backend.
type StorageBackendID int

// StorageType is the different checkpoint storage backends.
type StorageType string

const (
	// SharedFSStorageType is the type for shared_fs.
	SharedFSStorageType StorageType = "shared_fs"
	// S3StorageType is the type for s3.
	S3StorageType StorageType = "s3"
	// GCSStorageType is the type for s3.
	GCSStorageType StorageType = "gcs"
	// AzureStorageType is the type for azure.
	AzureStorageType StorageType = "azure"
	// DirectoryStorageType is the type for directory.
	DirectoryStorageType StorageType = "directory"
)
