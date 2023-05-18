package internal

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	modelauth "github.com/determined-ai/determined/master/internal/model"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

func errCheckpointNotFound(id string) error {
	return status.Errorf(codes.NotFound, "checkpoint not found: %s", id)
}

func errCheckpointsNotFound(ids []string) error {
	tmp := make([]string, len(ids))
	for i, id := range ids {
		tmp[i] = id
	}
	sort.Strings(tmp)
	return status.Errorf(codes.NotFound, "checkpoints not found: %s", strings.Join(tmp, ", "))
}

func (m *Master) canDoActionOnCheckpoint(
	ctx context.Context,
	curUser model.User,
	id string,
	action func(context.Context, model.User, *model.Experiment) error,
) error {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	checkpoint, err := m.db.CheckpointByUUID(uuid)
	if err != nil {
		return err
	} else if checkpoint == nil {
		return errCheckpointNotFound(id)
	}
	if checkpoint.CheckpointTrainingMetadata.ExperimentID == 0 {
		return nil // TODO(nick) add authz for other task types.
	}
	exp, err := db.ExperimentByID(ctx, checkpoint.CheckpointTrainingMetadata.ExperimentID)
	if err != nil {
		return err
	}

	if err := expauth.AuthZProvider.Get().CanGetExperiment(ctx, curUser, exp); err != nil {
		return authz.SubIfUnauthorized(err, errCheckpointNotFound(id))
	}
	if err := action(ctx, curUser, exp); err != nil {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	return nil
}

func (m *Master) canDoActionOnCheckpointThroughModel(
	ctx context.Context, curUser model.User, id uuid.UUID,
) error {
	modelIDs, err := db.GetModelIDsAssociatedWithCheckpoint(ctx, id)
	if err != nil {
		return err
	}
	for _, id := range modelIDs {
		model := &modelv1.Model{}
		err = m.db.QueryProto("get_model_by_id", model, id)
		if !errors.Is(err, db.ErrNotFound) {
			return err
		}
		if err := modelauth.AuthZProvider.Get().CanGetModel(
			ctx, curUser, model, model.WorkspaceId); err != nil {
			return authz.SubIfUnauthorized(err, nil)
		}
	}
	return status.Error(codes.PermissionDenied,
		fmt.Sprintf("cannot access checkpoint: %s", id.String()))
}

func (a *apiServer) GetCheckpoint(
	ctx context.Context, req *apiv1.GetCheckpointRequest,
) (*apiv1.GetCheckpointResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	errE := a.m.canDoActionOnCheckpoint(ctx, *curUser, req.CheckpointUuid,
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts)

	if errE != nil {
		ckptUUID, err := uuid.Parse(req.CheckpointUuid)
		if err != nil {
			return nil, err
		}
		errM := a.m.canDoActionOnCheckpointThroughModel(ctx, *curUser, ckptUUID)
		if errM != nil {
			return nil, errE
		}
	}

	resp := &apiv1.GetCheckpointResponse{}
	resp.Checkpoint = &checkpointv1.Checkpoint{}

	if err = a.m.db.QueryProto(
		"get_checkpoint", resp.Checkpoint, req.CheckpointUuid); err != nil {
		return resp,
			errors.Wrapf(err, "error fetching checkpoint %s from database", req.CheckpointUuid)
	}

	return resp, nil
}

func (a *apiServer) checkpointsRBACEditCheck(
	ctx context.Context, uuids []uuid.UUID,
) ([]*model.Experiment, []*db.ExperimentCheckpointGrouping, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, nil, err
	}

	groupCUUIDsByEIDs, err := a.m.db.GroupCheckpointUUIDsByExperimentID(uuids)
	if err != nil {
		return nil, nil, err
	}

	// Get checkpoints IDs not associated to any experiments.
	checkpointsRequested := make(map[string]bool)
	for _, c := range uuids {
		checkpointsRequested[c.String()] = false
	}
	for _, expIDcUUIDs := range groupCUUIDsByEIDs {
		for _, c := range strings.Split(expIDcUUIDs.CheckpointUUIDSStr, ",") {
			checkpointsRequested[c] = true
		}
	}
	var notFoundCheckpoints []string
	for c, found := range checkpointsRequested {
		if !found {
			notFoundCheckpoints = append(notFoundCheckpoints, c)
		}
	}

	// Get experiments for all checkpoints and validate
	// that the user has permission to view and edit.
	exps := make([]*model.Experiment, len(groupCUUIDsByEIDs))
	for i, expIDcUUIDs := range groupCUUIDsByEIDs {
		exp, err := db.ExperimentByID(ctx, expIDcUUIDs.ExperimentID)
		if err != nil {
			return nil, nil, err
		}
		err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, *curUser, exp)
		if authz.IsPermissionDenied(err) {
			notFoundCheckpoints = append(notFoundCheckpoints,
				strings.Split(expIDcUUIDs.CheckpointUUIDSStr, ",")...)
			continue
		} else if err != nil {
			return nil, nil, err
		}
		if err = expauth.AuthZProvider.Get().CanEditExperiment(ctx, *curUser, exp); err != nil {
			return nil, nil, status.Error(codes.PermissionDenied, err.Error())
		}

		exps[i] = exp
	}

	if len(notFoundCheckpoints) > 0 {
		return nil, nil, errCheckpointsNotFound(notFoundCheckpoints)
	}

	return exps, groupCUUIDsByEIDs, nil
}

func (a *apiServer) PatchCheckpoints(
	ctx context.Context,
	req *apiv1.PatchCheckpointsRequest,
) (*apiv1.PatchCheckpointsResponse, error) {
	fmt.Println("PATCH checkpoints", req.Checkpoints)
	var uuidStrings []string
	for _, c := range req.Checkpoints {
		uuidStrings = append(uuidStrings, c.Uuid)
	}

	conv := &protoconverter.ProtoConverter{}
	uuids := conv.ToUUIDList(uuidStrings)
	if cErr := conv.Error(); cErr != nil {
		return nil, status.Errorf(codes.InvalidArgument, "converting checkpoint: %s", cErr)
	}

	if _, _, err := a.checkpointsRBACEditCheck(ctx, uuids); err != nil {
		return nil, err
	}

	registeredCheckpointUUIDs, err := a.m.db.GetRegisteredCheckpoints(uuids)
	if err != nil {
		return nil, err
	}
	if len(registeredCheckpointUUIDs) > 0 {
		return nil, status.Errorf(codes.InvalidArgument,
			"this subset of checkpoints provided are in the model registry and cannot be deleted: %v.",
			registeredCheckpointUUIDs)
	}

	err = db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var updatedCheckpointSizes []uuid.UUID
		for i, c := range req.Checkpoints {
			if c.Resources != nil {
				fmt.Println("uuid / resources", c.Uuid, c.Resources.Resources)
				size := int64(0)
				for _, v := range c.Resources.Resources {
					size += v
				}

				v1Update := tx.NewUpdate().Model(&model.CheckpointV1{}).
					Where("uuid = ?", c.Uuid)
				v2Update := tx.NewUpdate().Model(&model.CheckpointV2{}).
					Where("uuid = ?", c.Uuid)

				if len(c.Resources.Resources) == 0 { // Full delete case.
					v1Update = v1Update.Set("state = ?", model.DeletedState)
					v2Update = v2Update.Set("state = ?", model.DeletedState)
				} else { // Partial delete case.
					v1Update = v1Update.
						Set(`state =
							(CASE WHEN ? @> resources AND resources @> ? THEN state ELSE ? END)`,
							c.Resources.Resources, c.Resources.Resources, model.PartiallyDeletedState).
						Set("resources = ?", c.Resources.Resources).
						Set("size = ?", size)
					v2Update = v2Update.
						Set(`state =
							(CASE WHEN ? @> resources AND resources @> ? THEN state ELSE ? END)`,
							c.Resources.Resources, c.Resources.Resources, model.PartiallyDeletedState).
						Set("resources = ?", c.Resources.Resources).
						Set("size = ?", size)
				}

				if _, err := v1Update.Exec(ctx); err != nil {
					return fmt.Errorf("deleting checkpoints from raw_checkpoints: %w", err)
				}

				if _, err := v2Update.Exec(ctx); err != nil {
					return fmt.Errorf("deleting checkpoints from checkpoints_v2: %w", err)
				}

				updatedCheckpointSizes = append(updatedCheckpointSizes, uuids[i])
			}
		}

		if len(updatedCheckpointSizes) > 0 {
			if err := db.UpdateCheckpointSizeTx(ctx, tx, updatedCheckpointSizes); err != nil {
				return fmt.Errorf("updating checkpoint size: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error patching checkpoints: %w", err)
	}

	return &apiv1.PatchCheckpointsResponse{}, nil
}

func (a *apiServer) CheckpointsRemoveFiles(
	ctx context.Context,
	req *apiv1.CheckpointsRemoveFilesRequest,
) (*apiv1.CheckpointsRemoveFilesResponse, error) {
	fmt.Printf("REQ %+v\n", req)
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	conv := &protoconverter.ProtoConverter{}
	checkpointsToDelete := conv.ToUUIDList(req.CheckpointUuids)
	if cErr := conv.Error(); cErr != nil {
		return nil, status.Errorf(codes.InvalidArgument, "converting checkpoint: %s", cErr)
	}

	// TODO don't allow GLOB!!! .. !! .. !!
	// Like actually "../**" -- maybe same as s3 validation but test this
	/*
		// TODO validate globs here!
		uuidToGlob := make(map[uuid.UUID]string)
		for i := 0; i < len(checkpointsToDelete); i++ {
			uuidToGlob[checkpointsToDelete[i]] = req.CheckpointGlobs[i]
		}
	*/

	exps, groupCUUIDsByEIDs, err := a.checkpointsRBACEditCheck(ctx, checkpointsToDelete)
	if err != nil {
		return nil, err
	}

	// TODO is this okay to just block model registry from partial deletes?
	registeredCheckpointUUIDs, err := a.m.db.GetRegisteredCheckpoints(checkpointsToDelete)
	if err != nil {
		return nil, err
	}
	if len(registeredCheckpointUUIDs) > 0 {
		return nil, status.Errorf(codes.InvalidArgument,
			"this subset of checkpoints provided are in the model registry and cannot be deleted: %v.",
			registeredCheckpointUUIDs)
	}

	addr := actor.Addr(fmt.Sprintf("checkpoints-gc-%s", uuid.New().String()))

	taskSpec := *a.m.taskSpec

	jobID := model.NewJobID()
	if err = a.m.db.AddJob(&model.Job{
		JobID:   jobID,
		JobType: model.JobTypeCheckpointGC,
		OwnerID: &curUser.ID,
	}); err != nil {
		return nil, fmt.Errorf("persisting new job: %w", err)
	}

	// Submit checkpoint GC tasks for all checkpoints.
	for i, expIDcUUIDs := range groupCUUIDsByEIDs {
		agentUserGroup, err := user.GetAgentUserGroup(curUser.ID, exps[i])
		if err != nil {
			return nil, err
		}

		jobSubmissionTime := time.Now().UTC().Truncate(time.Millisecond)
		taskID := model.NewTaskID()
		conv := &protoconverter.ProtoConverter{}
		checkpointUUIDs := conv.ToUUIDList(strings.Split(expIDcUUIDs.CheckpointUUIDSStr, ","))

		ckptGCTask := newCheckpointGCTask(
			a.m.rm, a.m.db, a.m.taskLogger, taskID, jobID, jobSubmissionTime, taskSpec, exps[i].ID,
			exps[i].Config, checkpointUUIDs, req.CheckpointGlobs, false, agentUserGroup, curUser, nil,
		)
		a.m.system.MustActorOf(addr, ckptGCTask)
	}

	return &apiv1.CheckpointsRemoveFilesResponse{}, nil
}

func (a *apiServer) DeleteCheckpoints(
	ctx context.Context,
	req *apiv1.DeleteCheckpointsRequest,
) (*apiv1.DeleteCheckpointsResponse, error) {
	if _, err := a.CheckpointsRemoveFiles(ctx, &apiv1.CheckpointsRemoveFilesRequest{
		CheckpointUuids: req.CheckpointUuids,
		CheckpointGlobs: []string{"**/*"},
	}); err != nil {
		return nil, err
	}

	return &apiv1.DeleteCheckpointsResponse{}, nil
}

func (a *apiServer) PostCheckpointMetadata(
	ctx context.Context, req *apiv1.PostCheckpointMetadataRequest,
) (*apiv1.PostCheckpointMetadataResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err = a.m.canDoActionOnCheckpoint(ctx, *curUser, req.Checkpoint.Uuid,
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	currCheckpoint := &checkpointv1.Checkpoint{}
	if err = a.m.db.QueryProto("get_checkpoint", currCheckpoint, req.Checkpoint.Uuid); err != nil {
		return nil,
			errors.Wrapf(err, "error fetching checkpoint %s from database", req.Checkpoint.Uuid)
	}

	currMeta, err := protojson.Marshal(currCheckpoint.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling database checkpoint metadata")
	}

	newMeta, err := protojson.Marshal(req.Checkpoint.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling request checkpoint metadata")
	}

	currCheckpoint.Metadata = req.Checkpoint.Metadata
	log.Infof("checkpoint (%s) metadata changing from %s to %s",
		req.Checkpoint.Uuid, currMeta, newMeta)
	err = a.m.db.QueryProto("update_checkpoint_metadata",
		&checkpointv1.Checkpoint{}, req.Checkpoint.Uuid, newMeta)

	return &apiv1.PostCheckpointMetadataResponse{Checkpoint: currCheckpoint},
		errors.Wrapf(err, "error updating checkpoint %s in database", req.Checkpoint.Uuid)
}
