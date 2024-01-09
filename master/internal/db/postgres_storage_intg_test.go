package db

import (
	"context"
	"encoding/json"
	"fmt"
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

func generateTestCases(t *testing.T) []storageBackendExhaustiveTestCases {
	var cases []storageBackendExhaustiveTestCases

	cs := expconf.CheckpointStorageConfig{}
	s := reflect.ValueOf(&cs).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)

		fmt.Printf("%d: %s %s = %v\n", i,
			typeOfT.Field(i).Name, f.Type(), f.Interface())

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

	cases := generateTestCases(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fmt.Println(c.checkpointStorage)
		})
	}

	/*
		value := reflect.ValueOf(cs)
		typeOf := value.Type()
		for i := 0; i < value.NumField(); i++ {
			fieldValue := value.Field(i)
			fieldType := typeOf.Field(i)

			// Check if the field has a "union" tag
			unionTag := fieldType.Tag.Get("union")
			if unionTag == "" {
				continue
			}

			t.Run(fmt.Sprintf("storage %s", unionTag), func(t *testing.T) {
				m := make(map[string]any)
				unionKey, unionVal, found := strings.Cut(unionTag, ",")
				require.True(t, found, "union tag not in expected format of unionKey,unionVal "+unionTag)
				m[unionKey] = unionVal

				subStruct := reflect.ValueOf(fieldValue.Interface())
				subValue := reflect.ValueOf(subStruct)
				subTypeOf := subValue.Type()
				fmt.Printf("TYPES %T %T %T %T\n", subStruct, subValue, subTypeOf)
				for i := 0; i < subValue.NumField(); i++ {
					// fieldValue := subValue.Field(i)
					fieldType := subTypeOf.Field(i)
					fmt.Printf("fieldType %T %v\n", fieldType, fieldType.Name)

					// Check if the field has a "union" tag
					jsonTag := fieldType.Tag.Get("json")
					if jsonTag == "" {
						continue
					}
					fmt.Println(jsonTag)
				}
			})
		}
	*/
}

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
