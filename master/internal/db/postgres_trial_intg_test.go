//go:build integration
// +build integration

package db

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3" // Can't use ghodss/yaml since NaNs error.

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/model"
)

func addMetrics(ctx context.Context,
	t *testing.T, db *PgDB, trialID int, trainMetricsJSON, valMetricsJSON string, archive bool,
) {
	var trainMetrics []map[string]any
	require.NoError(t, json.Unmarshal([]byte(trainMetricsJSON), &trainMetrics))

	trialRunID := 0
	for i, m := range trainMetrics {
		if archive && i == len(trainMetrics)-1 {
			// Add step that will be archived.
			metrics, err := structpb.NewStruct(map[string]any{"archive_metric_dont_appear": "3.14"})
			require.NoError(t, err)
			require.NoError(t, db.AddTrainingMetrics(ctx, &trialv1.TrialMetrics{
				TrialId:        int32(trialID),
				TrialRunId:     int32(trialRunID),
				StepsCompleted: int32(i) + 1,
				Metrics: &commonv1.Metrics{
					AvgMetrics: metrics,
				},
			}))
			trialRunID++
			require.NoError(t, db.UpdateTrialRunID(trialID, trialRunID))
		}

		metrics, err := structpb.NewStruct(m)
		require.NoError(t, err)
		require.NoError(t, db.AddTrainingMetrics(ctx, &trialv1.TrialMetrics{
			TrialId:        int32(trialID),
			TrialRunId:     int32(trialRunID),
			StepsCompleted: int32(i) + 1,
			Metrics: &commonv1.Metrics{
				AvgMetrics: metrics,
			},
		}))
	}

	var valMetrics []map[string]any
	require.NoError(t, json.Unmarshal([]byte(valMetricsJSON), &valMetrics))
	for i, m := range valMetrics {
		if archive && i == len(valMetrics)-1 {
			// Add step that will be archived.
			metrics, err := structpb.NewStruct(map[string]any{"archive_metric_dont_appear": "3.14"})
			require.NoError(t, err)
			require.NoError(t, db.AddValidationMetrics(ctx, &trialv1.TrialMetrics{
				TrialId:        int32(trialID),
				TrialRunId:     int32(trialRunID),
				StepsCompleted: int32(i + len(trainMetrics)),
				Metrics: &commonv1.Metrics{
					AvgMetrics: metrics,
				},
			}))
			trialRunID++
			require.NoError(t, db.UpdateTrialRunID(trialID, trialRunID))
		}

		metrics, err := structpb.NewStruct(m)
		require.NoError(t, err)
		require.NoError(t, db.AddValidationMetrics(ctx, &trialv1.TrialMetrics{
			TrialId:        int32(trialID),
			TrialRunId:     int32(trialRunID),
			StepsCompleted: int32(i + len(trainMetrics)),
			Metrics: &commonv1.Metrics{
				AvgMetrics: metrics,
			},
		}))
	}
}

func addMetricCustomTime(ctx context.Context, t *testing.T, trialID int, endTime time.Time) {
	metric := struct {
		bun.BaseModel `bun:"table:steps"`
		TrialID       int
		TrialRunID    int
		Metrics       map[string]any
		TotalBatches  int
		EndTime       time.Time
	}{
		TrialID:    trialID,
		TrialRunID: 1,
		Metrics: map[string]any{
			"avg_metrics": map[string]any{
				"b": -1.0,
			},
		},
		TotalBatches: 999999,
		EndTime:      endTime,
	}
	_, err := Bun().NewInsert().Model(&metric).Exec(ctx)
	require.NoError(t, err)

	valMetric := struct {
		bun.BaseModel `bun:"table:validations"`
		TrialID       int
		TrialRunID    int
		Metrics       map[string]any
		TotalBatches  int
		EndTime       time.Time
	}{
		TrialID:    trialID,
		TrialRunID: 1,
		Metrics: map[string]any{
			"validation_metrics": map[string]any{
				"val_loss": 3.0,
			},
		},
		TotalBatches: 999999,
		EndTime:      endTime,
	}
	_, err = Bun().NewInsert().Model(&valMetric).Exec(ctx)
	require.NoError(t, err)
}

func runSummaryMigration(t *testing.T) {
	bytes, err := os.ReadFile("../../static/migrations/20230405164440_add-summary-metrics.tx.up.sql")
	require.NoError(t, err)

	_, err = Bun().Exec(string(bytes))
	require.NoError(t, err)
}

func nanEqual(t *testing.T, expected, actual map[string]summaryMetrics) {
	e, err := yaml.Marshal(&expected)
	require.NoError(t, err)

	a, err := yaml.Marshal(&actual)
	require.NoError(t, err)

	require.Equal(t, string(e), string(a))
}

func validateSummaryMetrics(ctx context.Context, t *testing.T, trialID int,
	expectedTrain map[string]summaryMetrics,
	expectedVal map[string]summaryMetrics,
) {
	query := `SELECT name,
summary_metrics->'avg_metrics'->name->>'max' AS max,
summary_metrics->'avg_metrics'->name->>'min' AS min,
summary_metrics->'avg_metrics'->name->>'sum' AS sum,
summary_metrics->'avg_metrics'->name->>'last' AS last,
summary_metrics->'avg_metrics'->name->>'count' AS count
FROM trials
CROSS JOIN jsonb_object_keys(summary_metrics->'avg_metrics') AS name
WHERE id = ?;`

	trainRows := []*summaryMetrics{}
	err := Bun().NewRaw(query, trialID).Scan(ctx, &trainRows)
	require.NoError(t, err)

	actualTrain := make(map[string]summaryMetrics)
	for _, v := range trainRows {
		name := v.Name
		v.Name = ""
		actualTrain[name] = *v
	}
	nanEqual(t, expectedTrain, actualTrain)

	valRows := []*summaryMetrics{}
	err = Bun().NewRaw(strings.ReplaceAll(query, "avg_metrics", "validation_metrics"), trialID).
		Scan(ctx, &valRows)
	require.NoError(t, err)

	actualVal := make(map[string]summaryMetrics)
	for _, v := range valRows {
		name := v.Name
		v.Name = ""
		actualVal[name] = *v
	}
	nanEqual(t, expectedVal, actualVal)
}

func generateSummaryMetricsTestCases(
	ctx context.Context, t *testing.T, db *PgDB, archive bool,
) ([]int, []map[string]summaryMetrics, []map[string]summaryMetrics) {
	user := RequireMockUser(t, db)
	exp := RequireMockExperiment(t, db, user)

	noMetrics := RequireMockTrial(t, db, exp).ID
	addMetrics(ctx, t, db, noMetrics, `[]`, `[]`, archive)
	expectedNoMetrics := make(map[string]summaryMetrics)
	expectedNoValMetrics := make(map[string]summaryMetrics)

	numericMetrics := RequireMockTrial(t, db, exp).ID
	addMetrics(ctx, t, db, numericMetrics,
		`[{"a":1.0, "b":-0.5}, {"a":1.5,"b":0.0}, {"a":2.0}]`,
		`[{"val_loss": 1.5}]`, archive,
	)
	expectedNumericMetrics := map[string]summaryMetrics{
		"a": {Min: 1.0, Max: 2.0, Sum: 1.0 + 1.5 + 2.0, Count: 3, Last: "2"},
		"b": {Min: -0.5, Max: 0.0, Sum: -0.5 + 0.0, Count: 2}, // empty last.
	}
	expectedNumericValMetrics := map[string]summaryMetrics{
		"val_loss": {Min: 1.5, Max: 1.5, Sum: 1.5, Count: 1, Last: "1.5"},
	}

	nonNumericMetrics := RequireMockTrial(t, db, exp).ID
	addMetrics(ctx, t, db, nonNumericMetrics,
		`[{"a":"a", "b":-0.5}, {"a":1.67, "b":0.3, "c":"test"}, {"a":"c", "b":[{"loss":5.0}]}]`,
		`[{"val_loss": "c"}, {"val_gain": "d"}]`, archive,
	)
	expectedNonNumericMetrics := map[string]summaryMetrics{
		"a": {Last: "c"},
		"b": {Last: `[{"loss": 5}]`},
		"c": {},
	}
	expectedNonNumericValMetrics := map[string]summaryMetrics{
		"val_loss": {},
		"val_gain": {Last: "d"},
	}

	infNaNMetrics := RequireMockTrial(t, db, exp).ID
	addMetrics(ctx, t, db, infNaNMetrics,
		`[{"a":"NaN", "b":"-Infinity"}, {"a":1.0, "b":"Infinity"}]`,
		`[{"a":1.0, "b":"Infinity"}, {"a":"NaN", "b":"-Infinity"}]`, archive,
	)
	expectedInfNaNMetrics := map[string]summaryMetrics{
		"a": {Min: math.NaN(), Max: math.NaN(), Sum: math.NaN(), Count: 2, Last: "1"},
		"b": {Min: math.Inf(-1), Max: math.Inf(+1), Sum: math.NaN(), Count: 2, Last: "Infinity"},
	}
	expectedInfNaNValMetrics := map[string]summaryMetrics{
		"a": {Min: math.NaN(), Max: math.NaN(), Sum: math.NaN(), Count: 2, Last: "NaN"},
		"b": {Min: math.Inf(-1), Max: math.Inf(+1), Sum: math.NaN(), Count: 2, Last: "-Infinity"},
	}

	trialIDs := []int{noMetrics, numericMetrics, nonNumericMetrics, infNaNMetrics}
	expectedTrain := []map[string]summaryMetrics{
		expectedNoMetrics,
		expectedNumericMetrics,
		expectedNonNumericMetrics,
		expectedInfNaNMetrics,
	}
	expectedVal := []map[string]summaryMetrics{
		expectedNoValMetrics,
		expectedNumericValMetrics,
		expectedNonNumericValMetrics,
		expectedInfNaNValMetrics,
	}

	return trialIDs, expectedTrain, expectedVal
}

type summaryMetrics struct {
	Name  string
	Min   float64
	Max   float64
	Sum   float64
	Count int
	Last  any
}

func TestSummaryMetricsInsert(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	trialIDs, expectedTrain, expectedVal := generateSummaryMetricsTestCases(ctx, t, db, false)

	for i := 0; i < len(trialIDs); i++ {
		validateSummaryMetrics(ctx, t, trialIDs[i], expectedTrain[i], expectedVal[i])
	}
}

func TestSummaryMetricsInsertRollback(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	trialIDs, expectedTrain, expectedVal := generateSummaryMetricsTestCases(ctx, t, db, true)

	for i := 0; i < len(trialIDs); i++ {
		validateSummaryMetrics(ctx, t, trialIDs[i], expectedTrain[i], expectedVal[i])
	}
}

func TestSummaryMetricsMigration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	trialIDs, expectedTrain, expectedVal := generateSummaryMetricsTestCases(ctx, t, db, false)

	_, err := Bun().NewUpdate().Table("trials").
		Set("summary_metrics = '{}'").
		Set("summary_metrics_timestamp = NULL").
		Where("id IN (?)", bun.In(trialIDs)).
		Exec(ctx)
	require.NoError(t, err)

	runSummaryMigration(t)

	for i := 0; i < len(trialIDs); i++ {
		validateSummaryMetrics(ctx, t, trialIDs[i], expectedTrain[i], expectedVal[i])
	}

	// Add a metric with an older endtime to ensure metric isn't computed.
	addMetricCustomTime(ctx, t, trialIDs[0], time.Now().AddDate(0, 0, -1))

	// Verify metric is recomputed with new metrics added.
	addMetricCustomTime(ctx, t, trialIDs[1], time.Now())
	expectedTrain[1] = map[string]summaryMetrics{
		"a": {Min: 1.0, Max: 2.0, Sum: 1.0 + 1.5 + 2.0, Count: 3},
		"b": {Min: -1.0, Max: 0.0, Sum: -1.0 + -0.5 + 0.0, Count: 3, Last: "-1"},
	}
	expectedVal[1] = map[string]summaryMetrics{
		"val_loss": {Min: 1.5, Max: 3.0, Sum: 1.5 + 3.0, Count: 2, Last: "3"},
	}

	runSummaryMigration(t)

	for i := 0; i < len(trialIDs); i++ {
		validateSummaryMetrics(ctx, t, trialIDs[i], expectedTrain[i], expectedVal[i])
	}
}

func TestProtoGetTrial(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	exp, activeConfig := model.ExperimentModel()
	err := db.AddExperiment(exp, activeConfig)
	require.NoError(t, err, "failed to add experiment")

	task := RequireMockTask(t, db, exp.OwnerID)
	tr := model.Trial{
		TaskID:       task.TaskID,
		JobID:        exp.JobID,
		ExperimentID: exp.ID,
		State:        model.ActiveState,
		StartTime:    time.Now(),
	}
	err = db.AddTrial(&tr)
	require.NoError(t, err, "failed to add trial")

	startTime := time.Now().UTC()
	for i := 0; i < 3; i++ {
		a := &model.Allocation{
			AllocationID: model.AllocationID(fmt.Sprintf("%s-%d", tr.TaskID, i)),
			TaskID:       tr.TaskID,
			StartTime:    ptrs.Ptr(startTime.Add(time.Duration(i) * time.Second)),
			EndTime:      ptrs.Ptr(startTime.Add(time.Duration(i+1) * time.Second)),
		}
		err = db.AddAllocation(a)
		require.NoError(t, err, "failed to add allocation")
		err = db.CompleteAllocation(a)
		require.NoError(t, err, "failed to complete allocation")
	}

	var trResp trialv1.Trial
	err = db.QueryProtof(
		"proto_get_trials_plus",
		[]any{"($1::int, $2::int)"},
		&trResp,
		tr.ID,
		1,
	)
	require.NoError(t, err, "failed to query trial")
	require.Equal(t, trResp.WallClockTime, float64(3), "wall clock time is wrong")
}

// Covers an issue where checkpoint_view returned multiple records per checkpoint
// due to the LEFT JOIN raw_steps ON total_batches AND trial_id.
// This is in this file because AddValidationMetrics broke the assumption the join uses.
// That assumption being that for each trial_id and total_batches that there
// is at most one unarchived result.
func TestAddValidationMetricsDupeCheckpoints(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	exp, activeConfig := model.ExperimentModel()
	require.NoError(t, db.AddExperiment(exp, activeConfig))
	task := RequireMockTask(t, db, exp.OwnerID)
	tr := model.Trial{
		TaskID:       task.TaskID,
		JobID:        exp.JobID,
		ExperimentID: exp.ID,
		State:        model.ActiveState,
		StartTime:    time.Now(),
	}
	require.NoError(t, db.AddTrial(&tr))

	trainMetrics, err := structpb.NewStruct(map[string]any{"loss": 10})
	require.NoError(t, err)
	valMetrics, err := structpb.NewStruct(map[string]any{"loss": 50})
	require.NoError(t, err)

	// First trial run.
	a := &model.Allocation{
		AllocationID: model.AllocationID(fmt.Sprintf("%s-%d", tr.TaskID, 0)),
		TaskID:       tr.TaskID,
		StartTime:    ptrs.Ptr(time.Now()),
	}
	require.NoError(t, db.AddAllocation(a))

	// Report training metrics.
	require.NoError(t, db.AddTrainingMetrics(ctx, &trialv1.TrialMetrics{
		TrialId:        int32(tr.ID),
		TrialRunId:     0,
		StepsCompleted: 50,
		Metrics:        &commonv1.Metrics{AvgMetrics: trainMetrics},
	}))

	require.NoError(t, AddCheckpointMetadata(ctx, &model.CheckpointV2{
		UUID:         uuid.New(),
		TaskID:       task.TaskID,
		AllocationID: a.AllocationID,
		ReportTime:   time.Now(),
		State:        model.ActiveState,
		Metadata:     map[string]any{"steps_completed": 50},
	}))

	// Trial gets interrupted and starts in the future with a new trial run ID.
	a = &model.Allocation{
		AllocationID: model.AllocationID(fmt.Sprintf("%s-%d", tr.TaskID, 1)),
		TaskID:       tr.TaskID,
		StartTime:    ptrs.Ptr(time.Now()),
	}
	require.NoError(t, db.AddAllocation(a))
	require.NoError(t, db.UpdateTrialRunID(tr.ID, 1))

	// Now trial runs validation.
	require.NoError(t, db.AddValidationMetrics(ctx, &trialv1.TrialMetrics{
		TrialId:        int32(tr.ID),
		TrialRunId:     1,
		StepsCompleted: 50,
		Metrics:        &commonv1.Metrics{AvgMetrics: valMetrics},
	}))

	checkpoints := []*checkpointv1.Checkpoint{}
	require.NoError(t, db.QueryProto("get_checkpoints_for_experiment", &checkpoints, exp.ID))
	require.Len(t, checkpoints, 1)
	require.Equal(t, 10.0, checkpoints[0].Training.TrainingMetrics.AvgMetrics.AsMap()["loss"])
	require.Equal(t, 50.0, checkpoints[0].Training.ValidationMetrics.AvgMetrics.AsMap()["loss"])

	// Dummy metrics still happen if no other results at given total_batches.
	checkpoint2UUID := uuid.New()
	valMetrics2, err := structpb.NewStruct(map[string]any{"loss": 1.5})
	require.NoError(t, err)
	require.NoError(t, db.AddValidationMetrics(ctx, &trialv1.TrialMetrics{
		TrialId:        int32(tr.ID),
		TrialRunId:     1,
		StepsCompleted: 400,
		Metrics:        &commonv1.Metrics{AvgMetrics: valMetrics2},
	}))
	require.NoError(t, AddCheckpointMetadata(ctx, &model.CheckpointV2{
		UUID:         checkpoint2UUID,
		TaskID:       task.TaskID,
		AllocationID: a.AllocationID,
		ReportTime:   time.Now(),
		State:        model.ActiveState,
		Metadata:     map[string]any{"steps_completed": 400},
	}))
	checkpoints = []*checkpointv1.Checkpoint{}
	require.NoError(t, db.QueryProto("get_checkpoints_for_experiment", &checkpoints, exp.ID))
	require.Len(t, checkpoints, 2)
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Uuid != checkpoint2UUID.String() // Have second checkpoint later.
	})

	require.Equal(t, 10.0, checkpoints[0].Training.TrainingMetrics.AvgMetrics.AsMap()["loss"])
	require.Equal(t, 50.0, checkpoints[0].Training.ValidationMetrics.AvgMetrics.AsMap()["loss"])

	require.Equal(t, nil, checkpoints[1].Training.TrainingMetrics.AvgMetrics.AsMap()["loss"])
	require.Equal(t, 1.5, checkpoints[1].Training.ValidationMetrics.AvgMetrics.AsMap()["loss"])
}

func TestBatchesProcessed(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	exp, activeConfig := model.ExperimentModel()
	require.NoError(t, db.AddExperiment(exp, activeConfig))
	task := RequireMockTask(t, db, exp.OwnerID)
	tr := model.Trial{
		TaskID:       task.TaskID,
		JobID:        exp.JobID,
		ExperimentID: exp.ID,
		State:        model.ActiveState,
		StartTime:    time.Now(),
	}
	require.NoError(t, db.AddTrial(&tr))

	dbTr, err := db.TrialByID(tr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, dbTr.TotalBatches)

	a := &model.Allocation{
		AllocationID: model.AllocationID(fmt.Sprintf("%s-%d", tr.TaskID, 0)),
		TaskID:       tr.TaskID,
		StartTime:    ptrs.Ptr(time.Now()),
	}
	err = db.AddAllocation(a)
	require.NoError(t, err, "failed to add allocation")

	metrics, err := structpb.NewStruct(map[string]any{"loss": 10})
	require.NoError(t, err)

	testMetricReporting := func(typ string, trialRunId, batches, expectedTotalBatches int) error {
		require.NoError(t, db.UpdateTrialRunID(tr.ID, trialRunId))
		trialMetrics := &trialv1.TrialMetrics{
			TrialId:        int32(tr.ID),
			TrialRunId:     int32(trialRunId),
			StepsCompleted: int32(batches),
			Metrics:        &commonv1.Metrics{AvgMetrics: metrics},
		}
		switch typ {
		case "training":
			require.NoError(t, db.AddTrainingMetrics(ctx, trialMetrics))
		case "validation":
			require.NoError(t, db.AddValidationMetrics(ctx, trialMetrics))
		case "checkpoint":
			require.NoError(t, AddCheckpointMetadata(ctx, &model.CheckpointV2{
				UUID:         uuid.New(),
				TaskID:       task.TaskID,
				AllocationID: a.AllocationID,
				ReportTime:   time.Now(),
				State:        model.CompletedState,
				Metadata:     map[string]any{"steps_completed": batches},
			}))

		default:
			return errors.Errorf("unknown type %s", typ)
		}

		dbTr, err = db.TrialByID(tr.ID)
		require.NoError(t, err)
		require.Equal(t, expectedTotalBatches, dbTr.TotalBatches)
		return nil
	}

	cases := []struct {
		typ             string
		trialRunID      int
		batches         int
		expectedBatches int
	}{ // order matters.
		{"training", 0, 10, 10},
		{"validation", 0, 10, 10},
		{"training", 0, 20, 20},
		{"validation", 0, 20, 20},
		{"validation", 0, 30, 30}, // will be rolled back.
		{"training", 0, 25, 30},
		{"validation", 1, 25, 25}, // rollback via validations.
		{"validation", 1, 30, 30}, // will be rolled back.
		{"training", 1, 30, 30},   // will be rolled back.
		{"training", 2, 27, 27},   // rollback via training.
		{"checkpoint", 2, 30, 27}, // CHECK: do NOT account for steps_completed here.
		{"checkpoint", 3, 25, 27}, // CHECK: do NOT account for steps_completed here.
	}
	for _, c := range cases {
		require.NoError(t, testMetricReporting(c.typ, c.trialRunID, c.batches, c.expectedBatches))
	}

	// check rollbacks happened as expected.
	archivedSteps, err := Bun().NewSelect().Table("raw_steps").
		Where("trial_id = ?", tr.ID).Where("archived = true").Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, archivedSteps, "trial id %d", tr.ID)

	archivedValidations, err := Bun().NewSelect().Table("raw_validations").
		Where("trial_id = ?", tr.ID).Where("archived = true").Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, archivedValidations, "trial id %d", tr.ID)
}
