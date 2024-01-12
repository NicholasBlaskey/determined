//nolint:exhaustruct
package db

import (
	"context"
	"encoding/json"
	//"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/etc"
	//"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type storageBackendExhaustiveTestCases struct {
	name              string
	checkpointStorage string
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
			if jsonTag == "-" {
				continue
			}
			if unionVal == "azure" && jsonTag == "connection_string" {
				continue // Azure only can set one of these. So skip account_url.
			}

			testCase[jsonTag] = uuid.New().String()
		}

		bytes, err := json.Marshal(testCase)
		require.NoError(t, err)

		cases = append(cases, storageBackendExhaustiveTestCases{
			name:              unionTag,
			checkpointStorage: string(bytes),
		})
	}

	return cases
}

func fillUUIDs(s string) string {
	for strings.Contains(s, "%s") {
		s = strings.Replace(s, "%s", uuid.New().String(), 1)
	}

	return s
}

func TestStorageBackend(t *testing.T) {
	cases := []storageBackendExhaustiveTestCases{
		{"fs minimal", fillUUIDs(`{"type": "shared_fs", "host_path": "%s", "propagation": "rshared"}`)},
		{"s3 minimal", fillUUIDs(`{"type": "s3", "bucket": "%s"}`)},
		{"gcs minimal", fillUUIDs(`{"type": "gcs", "bucket": "%s"}`)},
		{"azure connection_string", fillUUIDs(`{"type": "azure", "container": "%s", "connection_string": "%s"}`)},
		{"azure url", fillUUIDs(`{"type": "azure", "container": "%s", "account_url": "%s", "credential": "%s"}`)},
		{"container minimal", fillUUIDs(`{"type": "directory", "container_path": "%s"}`)},
	}
	cases = append(cases, generateStorageBackendExhaustiveTestCases(t)...)

	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cs := &expconf.CheckpointStorageConfig{}
			require.NoError(t, cs.UnmarshalJSON([]byte(c.checkpointStorage)))

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

/*
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
		{"s3 missing bucket", storageBackend{
			Type: model.S3StorageType,
		}},
		{"gcs missing bucket", storageBackend{
			Type: model.GCSStorageType,
		}},
		{"azure missing container", storageBackend{
			Type:                  model.AzureStorageType,
			AzureConnectionString: ptrs.Ptr(uuid.New().String()),
		}},
		{"azure connect + url set", storageBackend{
			Type:                  model.AzureStorageType,
			AzureContainer:        ptrs.Ptr(uuid.New().String()),
			AzureConnectionString: ptrs.Ptr(uuid.New().String()),
			AzureAccountURL:       ptrs.Ptr(uuid.New().String()),
		}},
		{"azure connect + credential set", storageBackend{
			Type:                  model.AzureStorageType,
			AzureContainer:        ptrs.Ptr(uuid.New().String()),
			AzureConnectionString: ptrs.Ptr(uuid.New().String()),
			AzureAccountURL:       ptrs.Ptr(uuid.New().String()),
		}},
		{"directory missing container_path", storageBackend{
			Type: model.DirectoryStorageType,
		}},
		{"s3 wrong type", storageBackend{
			Type:             model.S3StorageType,
			SharedFSHostPath: ptrs.Ptr(uuid.New().String()),
			S3Bucket:         ptrs.Ptr(uuid.New().String()),
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
*/

func TestStorageBackendValidate(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	t.Run("s3 fails validate prefix errors", func(t *testing.T) {
		cs := &expconf.CheckpointStorageConfig{
			RawS3Config: &expconf.S3Config{
				RawBucket: ptrs.Ptr(uuid.New().String()),
				RawPrefix: ptrs.Ptr("../invalid"),
			},
		}
		_, err := AddStorageBackend(ctx, Bun(), cs)
		require.ErrorContains(t, err, "config is invalid")
	})

	t.Run("shared_fs defaults propgation", func(t *testing.T) {
		cs := &expconf.CheckpointStorageConfig{
			RawSharedFSConfig: &expconf.SharedFSConfigV0{
				RawHostPath: ptrs.Ptr(uuid.New().String()),
			},
		}

		_, err := AddStorageBackend(ctx, Bun(), cs)
		require.NoError(t, err)
	})
}

func TestStorageBackendQuery(t *testing.T) {
	cases := generateStorageBackendExhaustiveTestCases(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cs := &expconf.CheckpointStorageConfig{}
			require.NoError(t, cs.UnmarshalJSON([]byte(c.checkpointStorage)))

			csMap := make(map[string]any)
			require.NoError(t, json.Unmarshal([]byte(c.checkpointStorage), &csMap))

			backend, _ := expconfToStorage(cs)
			expected := Bun().NewSelect().Model(&backend).Column("id")
			for k, v := range csMap {
				if k == "type" {
					continue
				}

				if v == nil {
					expected = expected.Where("? IS NULL", bun.Safe(k))
				} else {
					expected = expected.Where("? = ?", bun.Safe(k), v)
				}
			}

			actual := getChildBackendRowsQuery(Bun(), backend)

			require.Equal(t, expected.String(), actual.String())
		})
	}

	/*
		backend := storageBackend{
			Type:             model.SharedFSStorageType,
			SharedFSHostPath: ptrs.Ptr(uuid.New().String()),
		}

		q := Bun().NewSelect().Model(&backend).Column("id")

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

		expected := storageIDByConfigQuery(Bun(), &backend)
		require.Equal(t, expected.String(), q.String())
	*/
}
