//nolint:exhaustruct
package storage

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type storageBackendExhaustiveTestCases struct {
	name              string
	checkpointStorage string
}

func TestMain(m *testing.M) {
	pgDB, err := db.ResolveTestPostgres()
	if err != nil {
		log.Panicln(err)
	}

	err = db.MigrateTestPostgres(pgDB, "file://../../static/migrations", "up")
	if err != nil {
		log.Panicln(err)
	}

	err = etc.SetRootPath("../../static/srv")
	if err != nil {
		log.Panicln(err)
	}

	os.Exit(m.Run())
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
				testCase["connection_string"] = nil
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
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cs := &expconf.CheckpointStorageConfig{}
			require.NoError(t, cs.UnmarshalJSON([]byte(c.checkpointStorage)))

			storageID, err := AddStorageBackend(ctx, db.Bun(), cs)
			require.NoError(t, err)

			// Test that we dedupe storage IDs.
			secondID, err := AddStorageBackend(ctx, db.Bun(), cs)
			require.NoError(t, err)
			require.Equal(t, storageID, secondID)

			actual, err := StorageBackend(ctx, db.Bun(), storageID)
			require.NoError(t, err)
			require.Equal(t, cs, actual)
		})
	}
}

func TestStorageBackendChecks(t *testing.T) {
	const reserved = "DeterminedReservedNullUniqueValue"

	cases := []struct {
		name     string
		toInsert storageBackend
	}{
		{"s3 bucket ..", &storageBackendS3{
			Bucket: uuid.New().String(),
			Prefix: ptrs.Ptr(".."),
		}},
		{"s3 bucket starts with ../", &storageBackendS3{
			Bucket: uuid.New().String(),
			Prefix: ptrs.Ptr("../test"),
		}},
		{"s3 bucket ends with /..", &storageBackendS3{
			Bucket: uuid.New().String(),
			Prefix: ptrs.Ptr("./test/.."),
		}},
		{"s3 bucket contains /../", &storageBackendS3{
			Bucket: uuid.New().String(),
			Prefix: ptrs.Ptr("./test/../git/."),
		}},
		{"gcs bucket ..", &storageBackendGCS{
			Bucket: uuid.New().String(),
			Prefix: ptrs.Ptr(".."),
		}},
		{"gcs bucket starts with ../", &storageBackendGCS{
			Bucket: uuid.New().String(),
			Prefix: ptrs.Ptr("../test"),
		}},
		{"gcs bucket ends with /..", &storageBackendGCS{
			Bucket: uuid.New().String(),
			Prefix: ptrs.Ptr("./test/.."),
		}},
		{"gcs bucket contains /../", &storageBackendGCS{
			Bucket: uuid.New().String(),
			Prefix: ptrs.Ptr("./test/../git/."),
		}},
		{"azure connect + url set", &storageBackendAzure{
			Container:        uuid.New().String(),
			ConnectionString: ptrs.Ptr(uuid.New().String()),
			AccountURL:       ptrs.Ptr(uuid.New().String()),
		}},
		{"azure connect + credential set", &storageBackendAzure{
			Container:        uuid.New().String(),
			ConnectionString: ptrs.Ptr(uuid.New().String()),
			AccountURL:       ptrs.Ptr(uuid.New().String()),
		}},
		{"fs reserved", &storageBackendSharedFS{
			HostPath:      uuid.New().String(),
			ContainerPath: ptrs.Ptr(reserved),
		}},
		{"s3 reserved", &storageBackendS3{
			Bucket: uuid.New().String(),
			Prefix: ptrs.Ptr(reserved),
		}},
		{"gcs reserved", &storageBackendGCS{
			Bucket: uuid.New().String(),
			Prefix: ptrs.Ptr(reserved),
		}},
		{"azure reserved", &storageBackendAzure{
			Container:  uuid.New().String(),
			Credential: ptrs.Ptr(reserved),
		}},
	}

	ctx := context.Background()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := db.Bun().NewInsert().Model(c.toInsert).Exec(ctx)
			require.ErrorContains(t, err, "constraint")
		})
	}
}

func TestStorageBackendValidate(t *testing.T) {
	ctx := context.Background()
	t.Run("s3 fails validate prefix errors", func(t *testing.T) {
		cs := &expconf.CheckpointStorageConfig{
			RawS3Config: &expconf.S3Config{
				RawBucket: ptrs.Ptr(uuid.New().String()),
				RawPrefix: ptrs.Ptr("../invalid"),
			},
		}
		_, err := AddStorageBackend(ctx, db.Bun(), cs)
		require.ErrorContains(t, err, "config is invalid")
	})

	t.Run("shared_fs defaults propgation", func(t *testing.T) {
		cs := &expconf.CheckpointStorageConfig{
			RawSharedFSConfig: &expconf.SharedFSConfigV0{
				RawHostPath: ptrs.Ptr(uuid.New().String()),
			},
		}

		_, err := AddStorageBackend(ctx, db.Bun(), cs)
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

			type clause struct {
				Where string
				Args  []any
			}

			backend, _ := expconfToStorage(cs)
			var expected []clause
			for k, v := range csMap {
				if k == "type" {
					continue
				}

				if v == nil {
					expected = append(expected, clause{
						Where: "? IS NULL",
						Args:  []any{bun.Safe(k)},
					})
				} else {
					expected = append(expected, clause{
						Where: "? = ?",
						Args:  []any{bun.Safe(k), v},
					})
				}
			}

			wheres, args := getChildBackendRowWheres(backend)
			var actual []clause
			for i := 0; i < len(wheres); i++ {
				actual = append(actual, clause{
					Where: wheres[i],
					Args:  args[i],
				})
			}
			require.ElementsMatch(t, expected, actual)
		})
	}
}
