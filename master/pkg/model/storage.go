package model

// StorageBackendID is the ID for the storage backend.
type StorageBackendID int

// TODO exhaust enums
// StorageType is the different checkpoint storage backends.
type StorageType string

const (
	// SharedFSStorageType is the type for shared_fs.
	SharedFSStorageType StorageType = "shared_fs"
)
