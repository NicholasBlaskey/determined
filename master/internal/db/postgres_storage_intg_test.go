package db

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	//	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

/*
func TestCheckpointStorageHasType() {
}

test every dang thing

test every field of every dang thing
*/

func fillUUIDs(s string) string {
	for strings.Contains(s, "%s") {
		s = strings.Replace(s, "%s", uuid.New().String(), 1)
	}

	fmt.Println(s)
	return s
}

func TestStorageBackend(t *testing.T) {
	cases := []struct {
		name string
		json string
	}{
		{"fs minimal", fillUUIDs(`{"type": "shared_fs", "host_path": "%s", "propagation": "rshared"}`)},
		{"fs maximal", fillUUIDs(`{"type": "shared_fs", "host_path": "%s", "container_path": "%s",
	"checkpoint_path": "%s", "tensorboard_path": "%s", "storage_path": "%s", "propagation": "rshared"}`)},
	}

	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cs := &expconf.CheckpointStorageConfig{}
			require.NoError(t, cs.UnmarshalJSON([]byte(c.json)))

			storageID, err := AddStorageBackend(ctx, Bun(), cs)
			require.NoError(t, err)

			// Test that we dedupe storage IDs.
			secondID, err := AddStorageBackend(ctx, Bun(), cs)
			require.NoError(t, err)
			require.Equal(t, storageID, secondID)

			actual, err := StorageBackend(ctx, Bun(), storageID)
			require.NoError(t, err)
			require.Equal(t, cs, actual)
		})
	}
}

func TestStorageBackendChecks(t *testing.T) {
	cases := []struct {
		name     string
		toInsert storageBackend
	}{
		{"fs missing host path", storageBackend{
			Type:                model.SharedFSStorageType,
			SharedFSPropagation: ptrs.Ptr(uuid.New().String()),
		}},
		{"fs missing propagation", storageBackend{
			Type:             model.SharedFSStorageType,
			SharedFSHostPath: ptrs.Ptr(uuid.New().String()),
		}},
	}

	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Bun().NewInsert().Model(&c.toInsert).Exec(ctx)
			require.ErrorContains(t, err, "constraint")
		})
	}
}
