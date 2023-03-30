//go:build integration
// +build integration

package db

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func sortUUIDSlice(uuids []uuid.UUID) {
	sort.Slice(uuids, func(i, j int) bool {
		return uuids[i].String() < uuids[j].String()
	})
}

// JSON doesn't like NaN / inf so just replace strings with them.
func correctFloats(m map[string]any) map[string]any {
	for k, v := range m {
		s, ok := v.(string)
		if !ok {
			continue
		}

		if s == "NaN" {
			m[k] = math.NaN()
		} else if s == "-inf" {
			m[k] = math.Inf(1)
		} else {
			m[k] = math.Inf(-1)
		}
	}
	return m
}

func addMetrics(t *testing.T, db *PgDB, trial *model.Trial, trainMetricsJSON, valMetricsJSON string) {
	var trainMetrics []map[string]any
	require.NoError(t, json.Unmarshal([]byte(trainMetricsJSON), &trainMetrics))
	for i, m := range trainMetrics {
		metrics, err := structpb.NewStruct(correctFloats(m))
		require.NoError(t, err)
		require.NoError(t, db.AddTrainingMetrics(context.TODO(), &trialv1.TrialMetrics{
			TrialId:        int32(trial.ID),
			TrialRunId:     0,
			StepsCompleted: int32(i),
			Metrics: &commonv1.Metrics{
				AvgMetrics: metrics,
			},
		}))
	}

	var valMetrics []map[string]any
	require.NoError(t, json.Unmarshal([]byte(valMetricsJSON), &valMetrics))
	for i, m := range trainMetrics {
		metrics, err := structpb.NewStruct(m)
		require.NoError(t, err)
		require.NoError(t, db.AddValidationMetrics(context.TODO(), &trialv1.TrialMetrics{
			TrialId:        int32(trial.ID),
			TrialRunId:     0,
			StepsCompleted: int32(i),
			Metrics: &commonv1.Metrics{
				AvgMetrics: metrics,
			},
		}))
	}

	fmt.Println("REMOVE ME")
}

func runSummaryMigration(t *testing.T) {
	const migrationPath = `../../static/up.sql`
	bytes, err := os.ReadFile(migrationPath)
	require.NoError(t, err)

	fmt.Println(string(bytes))
	_, err = Bun().Exec(string(bytes))
	require.NoError(t, err)
}

func validateSummaryMetrics(t *testing.T, trialID int, expected map[string]summaryMetrics) {
	var actualRaw map[string]any
	err := Bun().NewSelect().Table("trials").
		Column("summary_metrics").
		Where("id = ?", trialID).
		Scan(context.TODO(), &actualRaw)
	require.NoError(t, err)

	// TODO simply this conversion.
	actual := make(map[string]summaryMetrics)
	for m, v := range actualRaw {
		fmt.Println("V", v)
		a := v.(map[string]any)
		actual[m] = summaryMetrics{
			min:   a["min"].(float64),
			max:   a["max"].(float64),
			sum:   a["sum"].(float64),
			count: a["count"].(int),
			last:  a["last"],
		}
	}

	require.Equal(t, actual, expected)
}

type summaryMetrics struct {
	min   float64
	max   float64
	sum   float64
	count int
	last  any
}

func TestSummaryMetricsMigration(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)

	exp := RequireMockExperiment(t, db, user)

	/*
		noMetrics := RequireMockTrial(t, db, exp)
		addMetrics(t, db, noMetrics, `[{"loss":1.0}, {""}, {""}]`)
	*/

	numericMetrics := RequireMockTrial(t, db, exp)
	addMetrics(t, db, numericMetrics,
		`[{"a":1.0, "b":-0.5}, {"a":1.5,"b":0.0}, {"a":2.0}]`,
		`[{"val_loss": 1.5}]`,
	)
	expectedNumericMetrics := map[string]summaryMetrics{
		"a": {min: 1.0, max: 2.0, sum: 1.0 + 1.5 + 2.0, count: 3, last: 2.0},
		"b": {min: -0.5, max: 0.0, sum: -0.5 + 0.0 + -0.5, count: 2, last: 0.0}, // HMM last?
	}

	runSummaryMigration(t)

	validateSummaryMetrics(t, numericMetrics.ID, expectedNumericMetrics)
	/*
				nonNumericMetrics := RequireMockTrial(t, db, exp)
				mixedMetrics := RequireMockTrial(t, db, exp)
				nanMetrics := RequireMockTrial(t, db, exp)

		infMetrics
	*/

	// Add metrics.
}

func TestUpdateCheckpointSize(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)

	var resources []map[string]int64
	for i := 0; i < 8; i++ {
		resources = append(resources, map[string]int64{"TEST": int64(i) + 1})
	}

	// Create two experiments with two trials each with two checkpoints.
	var experimentIDs []int
	var trialIDs []int
	var checkpointIDs []uuid.UUID

	resourcesIndex := 0
	for i := 0; i < 2; i++ {
		exp := RequireMockExperiment(t, db, user)
		experimentIDs = append(experimentIDs, exp.ID)

		for j := 0; j < 2; j++ {
			tr := RequireMockTrial(t, db, exp)
			allocation := RequireMockAllocation(t, db, tr.TaskID)
			trialIDs = append(trialIDs, tr.ID)

			for k := 0; k < 2; k++ {
				ckpt := uuid.New()
				checkpointIDs = append(checkpointIDs, ckpt)
				// Ensure it works with both checkpoint versions.
				if i == 0 && j == 0 && k == 0 {
					checkpointBun := struct {
						bun.BaseModel `bun:"table:checkpoints"`
						TrialID       int
						TrialRunID    int
						TotalBatches  int
						State         model.State
						UUID          string
						EndTime       time.Time
						Resources     map[string]int64
						Size          int64
					}{
						TrialID:      tr.ID,
						TrialRunID:   1,
						TotalBatches: 1,
						State:        model.ActiveState,
						UUID:         ckpt.String(),
						EndTime:      time.Now().UTC().Truncate(time.Millisecond),
						Resources:    resources[resourcesIndex],
						Size:         resources[resourcesIndex]["TEST"],
					}

					_, err := Bun().NewInsert().Model(&checkpointBun).Exec(ctx)
					require.NoError(t, err)
				} else {
					checkpoint := MockModelCheckpoint(ckpt, tr, allocation)
					checkpoint.Resources = resources[resourcesIndex]
					err := AddCheckpointMetadata(ctx, &checkpoint)
					require.NoError(t, err)
				}

				resourcesIndex++
			}
		}
	}

	type expected struct {
		checkpointSizes []int64

		trialCounts []int
		trialSizes  []int64

		experimentCounts []int
		experimentSizes  []int64
	}

	verifySizes := func(e expected) {
		for i, checkpointID := range checkpointIDs {
			var size int64
			err := Bun().NewSelect().Table("checkpoints_view").
				Column("size").
				Where("uuid = ?", checkpointID).
				Scan(context.Background(), &size)
			require.NoError(t, err)
			require.Equal(t, e.checkpointSizes[i], size)
		}

		for i, trialID := range trialIDs {
			actual := struct {
				CheckpointSize  int64
				CheckpointCount int
			}{}
			err := Bun().NewSelect().Table("trials").
				Column("checkpoint_size").
				Column("checkpoint_count").
				Where("id = ?", trialID).
				Scan(context.Background(), &actual)
			require.NoError(t, err)

			require.Equal(t, e.trialCounts[i], actual.CheckpointCount)
			require.Equal(t, e.trialSizes[i], actual.CheckpointSize)
		}

		for i, experimentID := range experimentIDs {
			actual := struct {
				CheckpointSize  int64
				CheckpointCount int
			}{}
			err := Bun().NewSelect().Table("experiments").
				Column("checkpoint_size").
				Column("checkpoint_count").
				Where("id = ?", experimentID).
				Scan(context.Background(), &actual)
			require.NoError(t, err)

			require.Equal(t, e.experimentCounts[i], actual.CheckpointCount)
			require.Equal(t, e.experimentSizes[i], actual.CheckpointSize)
		}
	}

	e := expected{
		checkpointSizes: []int64{1, 2, 3, 4, 5, 6, 7, 8},

		trialCounts: []int{2, 2, 2, 2},
		trialSizes:  []int64{1 + 2, 3 + 4, 5 + 6, 7 + 8},

		experimentCounts: []int{4, 4},
		experimentSizes:  []int64{1 + 2 + 3 + 4, 5 + 6 + 7 + 8},
	}
	verifySizes(e)

	require.NoError(t, MarkCheckpointsDeleted(ctx, checkpointIDs[:2]))
	e.trialCounts = []int{0, 2, 2, 2}
	e.trialSizes = []int64{0, 3 + 4, 5 + 6, 7 + 8}
	e.experimentCounts = []int{2, 4}
	e.experimentSizes = []int64{3 + 4, 5 + 6 + 7 + 8}
	verifySizes(e)

	require.NoError(t, MarkCheckpointsDeleted(ctx, checkpointIDs[3:5]))
	e.trialCounts = []int{0, 1, 1, 2}
	e.trialSizes = []int64{0, 3, 6, 7 + 8}
	e.experimentCounts = []int{1, 3}
	e.experimentSizes = []int64{3, 6 + 7 + 8}
	verifySizes(e)

	require.NoError(t, MarkCheckpointsDeleted(ctx, checkpointIDs))
	e.trialCounts = []int{0, 0, 0, 0}
	e.trialSizes = []int64{0, 0, 0, 0}
	e.experimentCounts = []int{0, 0}
	e.experimentSizes = []int64{0, 0}
	verifySizes(e)
}

func TestDeleteCheckpoints(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)
	exp := RequireMockExperiment(t, db, user)
	tr := RequireMockTrial(t, db, exp)
	allocation := RequireMockAllocation(t, db, tr.TaskID)

	// Create checkpoints
	ckpt1 := uuid.New()
	checkpoint1 := MockModelCheckpoint(ckpt1, tr, allocation)
	err := AddCheckpointMetadata(ctx, &checkpoint1)
	require.NoError(t, err)
	ckpt2 := uuid.New()
	checkpoint2 := MockModelCheckpoint(ckpt2, tr, allocation)
	err = AddCheckpointMetadata(ctx, &checkpoint2)
	require.NoError(t, err)
	ckpt3 := uuid.New()
	checkpoint3 := MockModelCheckpoint(ckpt3, tr, allocation)
	err = AddCheckpointMetadata(ctx, &checkpoint3)
	require.NoError(t, err)

	// Insert a model.
	now := time.Now()
	mdl := model.Model{
		Name:            uuid.NewString(),
		Description:     "some important model",
		CreationTime:    now,
		LastUpdatedTime: now,
		Labels:          []string{"some other label"},
		Username:        user.Username,
		WorkspaceID:     1,
	}
	mdlNotes := "some notes2"
	var pmdl modelv1.Model
	err = db.QueryProto(
		"insert_model", &pmdl, mdl.Name, mdl.Description, emptyMetadata,
		strings.Join(mdl.Labels, ","), mdlNotes, user.ID, mdl.WorkspaceID,
	)

	require.NoError(t, err)

	// Register checkpoint_1 and checkpoint_2 in ModelRegistry
	var retCkpt1 checkpointv1.Checkpoint
	err = db.QueryProto("get_checkpoint", &retCkpt1, checkpoint1.UUID)
	require.NoError(t, err)
	var retCkpt2 checkpointv1.Checkpoint
	err = db.QueryProto("get_checkpoint", &retCkpt2, checkpoint2.UUID)
	require.NoError(t, err)

	addmv := modelv1.ModelVersion{
		Model:      &pmdl,
		Checkpoint: &retCkpt1,
		Name:       "checkpoint 1",
		Comment:    "empty",
	}
	var mv modelv1.ModelVersion
	err = db.QueryProto(
		"insert_model_version", &mv, pmdl.Id, retCkpt1.Uuid, addmv.Name, addmv.Comment,
		emptyMetadata, strings.Join(addmv.Labels, ","), addmv.Notes, user.ID,
	)
	require.NoError(t, err)

	addmv = modelv1.ModelVersion{
		Model:      &pmdl,
		Checkpoint: &retCkpt2,
		Name:       "checkpoint 2",
		Comment:    "empty",
	}
	err = db.QueryProto(
		"insert_model_version", &mv, pmdl.Id, retCkpt2.Uuid, addmv.Name, addmv.Comment,
		emptyMetadata, strings.Join(addmv.Labels, ","), addmv.Notes, user.ID,
	)
	require.NoError(t, err)

	// Test CheckpointsByUUIDs
	reqCheckpointUUIDs := []uuid.UUID{checkpoint1.UUID, checkpoint2.UUID, checkpoint3.UUID}
	checkpointsByUUIDs, err := db.CheckpointByUUIDs(reqCheckpointUUIDs)
	require.NoError(t, err)
	dbCheckpointsUUIDs := []uuid.UUID{
		*checkpointsByUUIDs[0].UUID, *checkpointsByUUIDs[1].UUID, *checkpointsByUUIDs[2].UUID,
	}
	sortUUIDSlice(reqCheckpointUUIDs)
	sortUUIDSlice(dbCheckpointsUUIDs)
	require.Equal(t, reqCheckpointUUIDs, dbCheckpointsUUIDs)

	// Test GetModelIDsAssociatedWithCheckpoint
	expmodelIDsCheckpoint := []int32{pmdl.Id}
	modelIDsCheckpoint, err := GetModelIDsAssociatedWithCheckpoint(context.TODO(), checkpoint1.UUID)
	require.NoError(t, err)
	require.Equal(t, expmodelIDsCheckpoint, modelIDsCheckpoint)
	// Send a list of delete checkpoints uuids the user wants to delete and
	// check if it's in model registry.
	requestedDeleteCheckpoints := []uuid.UUID{checkpoint1.UUID, checkpoint3.UUID}
	expectedDeleteInModelRegistryCheckpoints := make(map[uuid.UUID]bool)
	expectedDeleteInModelRegistryCheckpoints[checkpoint1.UUID] = true
	dCheckpointsInRegistry, err := db.GetRegisteredCheckpoints(requestedDeleteCheckpoints)
	require.NoError(t, err)
	require.Equal(t, expectedDeleteInModelRegistryCheckpoints, dCheckpointsInRegistry)

	validDeleteCheckpoint := checkpoint3.UUID
	numValidDCheckpoints := 1

	require.NoError(t, MarkCheckpointsDeleted(ctx, []uuid.UUID{validDeleteCheckpoint}))

	var numDStateCheckpoints int

	err = db.sql.QueryRowx(`SELECT count(c.uuid) AS numC from checkpoints_view AS c WHERE
	c.uuid::text = $1 AND c.state = 'DELETED';`, validDeleteCheckpoint).Scan(&numDStateCheckpoints)
	require.NoError(t, err)
	require.Equal(t, numValidDCheckpoints, numDStateCheckpoints,
		"didn't correctly delete the valid checkpoints")
}

func BenchmarkUpdateCheckpointSize(b *testing.B) {
	ctx := context.Background()
	t := (*testing.T)(unsafe.Pointer(b)) //nolint: gosec // Hack to still use methods that take t.
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)

	var checkpoints []uuid.UUID
	exp := RequireMockExperiment(t, db, user)
	for j := 0; j < 10; j++ {
		t.Logf("Adding trial #%d", j)
		tr := RequireMockTrial(t, db, exp)
		allocation := RequireMockAllocation(t, db, tr.TaskID)
		for k := 0; k < 10; k++ {
			ckpt := uuid.New()
			checkpoints = append(checkpoints, ckpt)

			resources := make(map[string]int64)
			for r := 0; r < 100000; r++ {
				resources[uuid.New().String()] = rand.Int63n(2500) //nolint: gosec
			}

			checkpoint := MockModelCheckpoint(ckpt, tr, allocation)
			checkpoint.Resources = resources

			err := AddCheckpointMetadata(ctx, &checkpoint)
			require.NoError(t, err)
		}
	}

	require.NoError(t, MarkCheckpointsDeleted(ctx, checkpoints))
}
