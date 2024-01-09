//nolint:exhaustruct
package db

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type storageBackendExhaustiveTestCases struct {
	name              string
	checkpointStorage *expconf.CheckpointStorageConfig
}

func generateStorageBackendExhaustiveTestCases(t *testing.T) []storageBackendExhaustiveTestCases {
	var cases []storageBackendExhaustiveTestCases

	cs := expconf.CheckpointStorageConfig{}
	s := reflect.ValueOf(&cs).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)

		unionTag := typeOfT.Field(i).Tag.Get("union")
		if f.Kind() != reflect.Ptr || unionTag == "" {
			continue
		}

		unionKey, unionVal, found := strings.Cut(unionTag, ",")
		require.True(t, found, "union tag not in expected format of unionKey,unionVal "+unionTag)
		testCase := map[string]any{unionKey: unionVal}

		subStruct := reflect.New(f.Type().Elem())
		subS := subStruct.Elem()
		subTypeOfS := subS.Type()
		for i := 0; i < subS.NumField(); i++ {
			if subTypeOfS.Field(i).Type.Kind() != reflect.Ptr &&
				subTypeOfS.Field(i).Type.Elem().Kind() != reflect.String {
				require.Fail(t, "this test only handles *string, you can add logic "+
					"to skip the non *string field if you add a test case in TestStorageBackend")
			}

			jsonTag, _, _ := strings.Cut(subTypeOfS.Field(i).Tag.Get("json"), ",")
			if jsonTag != "-" {
				testCase[jsonTag] = uuid.New().String()
			}
		}

		bytes, err := json.Marshal(testCase)
		require.NoError(t, err)
		cs := &expconf.CheckpointStorageConfig{}
		require.NoError(t, cs.UnmarshalJSON(bytes))
		cases = append(cases, storageBackendExhaustiveTestCases{
			name:              unionTag,
			checkpointStorage: cs,
		})
	}

	return cases
}

func TestStorageBackendExhaustive(t *testing.T) {
	// This test is designed to make sure any added checkpoint fields are persisted correctly.
	// This test should fail if you add a new checkpoint storage type or add a new field
	// and do not update the database.
	cases := generateStorageBackendExhaustiveTestCases(t)

	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			storageID, err := AddStorageBackend(ctx, Bun(), c.checkpointStorage)
			require.NoError(t, err)

			// Test that we dedupe storage IDs.
			secondID, err := AddStorageBackend(ctx, Bun(), c.checkpointStorage)
			require.NoError(t, err)
			require.Equal(t, storageID, secondID)

			actual, err := StorageBackend(ctx, Bun(), storageID)
			require.NoError(t, err)
			require.Equal(t, c.checkpointStorage, actual)
		})
	}
}

func fillUUIDs(s string) string {
	for strings.Contains(s, "%s") {
		s = strings.Replace(s, "%s", uuid.New().String(), 1)
	}

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
