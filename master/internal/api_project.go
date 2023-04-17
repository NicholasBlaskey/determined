package internal

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api/apiutils"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/project"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

func (a *apiServer) GetProjectByID(
	ctx context.Context, id int32, curUser model.User,
) (*projectv1.Project, error) {
	notFoundErr := status.Errorf(codes.NotFound, "project (%d) not found", id)
	p := &projectv1.Project{}
	if err := a.m.db.QueryProto("get_project", p, id); errors.Is(err, db.ErrNotFound) {
		return nil, notFoundErr
	} else if err != nil {
		return nil, fmt.Errorf("error fetching project (%d) from database: %w", id, err)
	}

	if ok, err := project.AuthZProvider.Get().CanGetProject(ctx, curUser, p); err != nil {
		return nil, err
	} else if !ok {
		return nil, notFoundErr
	}
	return p, nil
}

func (a *apiServer) getProjectColumnsByID(
	ctx context.Context, id int32, curUser model.User,
) (*apiv1.GetProjectColumnsResponse, error) {
	notFoundErr := status.Errorf(codes.NotFound, "project (%d) not found", id)
	p := &projectv1.Project{}
	if err := a.m.db.QueryProto("get_project", p, id); errors.Is(err, db.ErrNotFound) {
		return nil, notFoundErr
	} else if err != nil {
		return nil, fmt.Errorf("error fetching project (%d) from database: %w", id, err)
	}
	if ok, err := project.AuthZProvider.Get().CanGetProject(ctx, curUser, p); err != nil {
		return nil, err
	} else if !ok {
		return nil, notFoundErr
	}
	// Get general columns
	generalColumns := make([]projectv1.GeneralColumn, 0, len(projectv1.GeneralColumn_value))
	for gc := range projectv1.GeneralColumn_value {
		generalColumns = append(
			generalColumns, projectv1.GeneralColumn(projectv1.GeneralColumn_value[gc]))
	}
	// Get hyperpatameters columns
	hyperparameters := struct {
		Hyperparameters []string
	}{}
	err := db.Bun().
		NewSelect().Table("projects").Column("hyperparameters").Where(
		"id = ?", id).Scan(ctx, &hyperparameters)
	if err != nil {
		return nil, err
	}

	// Get metrics columns
	metricNames := []struct {
		Vname []string
	}{}
	metricColumns := make([]string, 0)
	err = db.Bun().
		NewSelect().
		TableExpr("exp_metrics_name").
		TableExpr("LATERAL json_array_elements_text(vname) AS vnames").
		ColumnExpr("array_to_json(array_agg(DISTINCT vnames)) AS vname").
		Where("project_id = ?", id).Scan(ctx, &metricNames)
	if err != nil {
		return nil, err
	}
	for _, mn := range metricNames {
		for _, mnv := range mn.Vname {
			metricColumns = append(metricColumns, fmt.Sprintf("%s.%s", "validation", mnv))
		}
	}

	return &apiv1.GetProjectColumnsResponse{
		General: generalColumns, Hyperparameters: hyperparameters.Hyperparameters, Metrics: metricColumns,
	}, nil
}

func (a *apiServer) getProjectAndCheckCanDoActions(
	ctx context.Context, projectID int32,
	canDoActions ...func(context.Context, model.User, *projectv1.Project) error,
) (*projectv1.Project, model.User, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, model.User{}, err
	}
	p, err := a.GetProjectByID(ctx, projectID, *curUser)
	if err != nil {
		return nil, model.User{}, err
	}

	for _, canDoAction := range canDoActions {
		if err = canDoAction(ctx, *curUser, p); err != nil {
			return nil, model.User{}, status.Error(codes.PermissionDenied, err.Error())
		}
	}
	return p, *curUser, nil
}

func (a *apiServer) CheckParentWorkspaceUnarchived(project *projectv1.Project) error {
	w := &workspacev1.Workspace{}
	err := a.m.db.QueryProto("get_workspace_from_project", w, project.Id)
	if err != nil {
		return fmt.Errorf("error fetching project (%v)'s workspace from database: %w", project.Id, err)
	}

	if w.Archived {
		//nolint: stylecheck.
		return fmt.Errorf("This project belongs to an archived workspace. " +
			"To make changes, first unarchive the workspace.")
	}
	return nil
}

func (a *apiServer) GetProject(
	ctx context.Context, req *apiv1.GetProjectRequest,
) (*apiv1.GetProjectResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	p, err := a.GetProjectByID(ctx, req.Id, *curUser)
	return &apiv1.GetProjectResponse{Project: p}, err
}

func (a *apiServer) GetProjectColumns(
	ctx context.Context, req *apiv1.GetProjectColumnsRequest,
) (*apiv1.GetProjectColumnsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	return a.getProjectColumnsByID(ctx, req.Id, *curUser)
}

func (a *apiServer) PostProject(
	ctx context.Context, req *apiv1.PostProjectRequest,
) (*apiv1.PostProjectResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	w, err := a.GetWorkspaceByID(ctx, req.WorkspaceId, *curUser, true)
	if err != nil {
		return nil, err
	}
	if err = project.AuthZProvider.Get().CanCreateProject(ctx, *curUser, w); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	p := &projectv1.Project{}
	err = a.m.db.QueryProto("insert_project", p, req.Name, req.Description,
		req.WorkspaceId, curUser.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating project %s in database: %w", req.Name, err)
	}

	return &apiv1.PostProjectResponse{Project: p}, nil
}

func (a *apiServer) AddProjectNote(
	ctx context.Context, req *apiv1.AddProjectNoteRequest,
) (*apiv1.AddProjectNoteResponse, error) {
	p, _, err := a.getProjectAndCheckCanDoActions(ctx, req.ProjectId,
		project.AuthZProvider.Get().CanSetProjectNotes)
	if err != nil {
		return nil, err
	}

	notes := p.Notes
	notes = append(notes, &projectv1.Note{
		Name:     req.Note.Name,
		Contents: req.Note.Contents,
	})

	newp := &projectv1.Project{}
	err = a.m.db.QueryProto("insert_project_note", newp, req.ProjectId, notes)
	if err != nil {
		return nil, fmt.Errorf("error adding project note: %w", err)
	}

	return &apiv1.AddProjectNoteResponse{Notes: newp.Notes}, nil
}

func (a *apiServer) PutProjectNotes(
	ctx context.Context, req *apiv1.PutProjectNotesRequest,
) (*apiv1.PutProjectNotesResponse, error) {
	_, _, err := a.getProjectAndCheckCanDoActions(ctx, req.ProjectId,
		project.AuthZProvider.Get().CanSetProjectNotes)
	if err != nil {
		return nil, err
	}

	newp := &projectv1.Project{}
	err = a.m.db.QueryProto("insert_project_note", newp, req.ProjectId, req.Notes)
	if err != nil {
		return nil, fmt.Errorf("error putting project notes: %w", err)
	}

	return &apiv1.PutProjectNotesResponse{Notes: newp.Notes}, nil
}

func (a *apiServer) PatchProject(
	ctx context.Context, req *apiv1.PatchProjectRequest,
) (*apiv1.PatchProjectResponse, error) {
	currProject, currUser, err := a.getProjectAndCheckCanDoActions(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if currProject.Archived {
		return nil, fmt.Errorf("project (%d) is archived and cannot have attributes updated.",
			currProject.Id)
	}
	if currProject.Immutable {
		return nil, fmt.Errorf("project (%v) is immutable and cannot have attributes updated.",
			currProject.Id)
	}

	madeChanges := false
	if req.Project.Name != nil && req.Project.Name.Value != currProject.Name {
		if err = project.AuthZProvider.Get().CanSetProjectName(ctx, currUser, currProject); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		log.Infof("project (%d) name changing from \"%s\" to \"%s\"",
			currProject.Id, currProject.Name, req.Project.Name.Value)
		madeChanges = true
		currProject.Name = req.Project.Name.Value
	}

	if req.Project.Description != nil && req.Project.Description.Value != currProject.Description {
		if err = project.AuthZProvider.Get().
			CanSetProjectDescription(ctx, currUser, currProject); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		log.Infof("project (%d) description changing from \"%s\" to \"%s\"",
			currProject.Id, currProject.Description, req.Project.Description.Value)
		madeChanges = true
		currProject.Description = req.Project.Description.Value
	}

	if !madeChanges {
		return &apiv1.PatchProjectResponse{Project: currProject}, nil
	}

	finalProject := &projectv1.Project{}
	err = a.m.db.QueryProto("update_project",
		finalProject, currProject.Id, currProject.Name, currProject.Description)
	if err != nil {
		return nil, fmt.Errorf("error updating project (%d) in database: %w", currProject.Id, err)
	}

	return &apiv1.PatchProjectResponse{Project: finalProject}, nil
}

func (a *apiServer) deleteProject(ctx context.Context, projectID int32,
	expList []*model.Experiment,
) (err error) {
	holder := &projectv1.Project{}
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		log.WithError(err).Errorf("failed to access user and delete project %d", projectID)
		_ = a.m.db.QueryProto("delete_fail_project", holder, projectID, err.Error())
		return err
	}

	log.Debugf("deleting project %d experiments", projectID)
	for _, exp := range expList {
		if err = a.deleteExperiment(exp, user); err != nil {
			log.WithError(err).Errorf("failed to delete experiment %d", exp.ID)
			_ = a.m.db.QueryProto("delete_fail_project", holder, projectID, err.Error())
			return err
		}
	}
	log.Debugf("project %d experiments deleted successfully", projectID)
	err = a.m.db.QueryProto("delete_project", holder, projectID)
	if err != nil {
		log.WithError(err).Errorf("failed to delete project %d", projectID)
		_ = a.m.db.QueryProto("delete_fail_project", holder, projectID, err.Error())
		return err
	}
	log.Debugf("project %d deleted successfully", projectID)
	return nil
}

func (a *apiServer) DeleteProject(
	ctx context.Context, req *apiv1.DeleteProjectRequest) (*apiv1.DeleteProjectResponse,
	error,
) {
	_, _, err := a.getProjectAndCheckCanDoActions(ctx, req.Id,
		project.AuthZProvider.Get().CanDeleteProject)
	if err != nil {
		return nil, err
	}

	holder := &projectv1.Project{}
	err = a.m.db.QueryProto("deletable_project", holder, req.Id)
	if err != nil || holder.Id == 0 {
		return nil, fmt.Errorf("project (%d) does not exist or not deletable by this user: %w",
			req.Id, err)
	}

	expList, err := a.m.db.ProjectExperiments(int(req.Id))
	if err != nil {
		return nil, err
	}

	if len(expList) == 0 {
		if err = a.m.db.QueryProto("delete_project", holder, req.Id); err != nil {
			return nil, fmt.Errorf("error deleting project (%d): %w", req.Id, err)
		}
		return &apiv1.DeleteProjectResponse{Completed: (err == nil)}, nil
	}
	go func() {
		_ = a.deleteProject(ctx, req.Id, expList)
	}()

	return &apiv1.DeleteProjectResponse{Completed: false}, nil
}

func (a *apiServer) MoveProject(
	ctx context.Context, req *apiv1.MoveProjectRequest) (*apiv1.MoveProjectResponse,
	error,
) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	p, err := a.GetProjectByID(ctx, req.ProjectId, *curUser)
	if err != nil { // Can view project?
		return nil, err
	}
	// Allow projects to be moved from immutable workspaces but not to immutable workspaces.
	from, err := a.GetWorkspaceByID(ctx, p.WorkspaceId, *curUser, false)
	if err != nil {
		return nil, err
	}
	to, err := a.GetWorkspaceByID(ctx, req.DestinationWorkspaceId, *curUser, true)
	if err != nil {
		return nil, err
	}
	if err = project.AuthZProvider.Get().CanMoveProject(ctx, *curUser, p, from, to); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	holder := &projectv1.Project{}
	err = a.m.db.QueryProto("move_project", holder, req.ProjectId, req.DestinationWorkspaceId)
	if err != nil {
		return nil, fmt.Errorf("error moving project (%d): %w", req.ProjectId, err)
	}
	if holder.Id == 0 {
		return nil, fmt.Errorf("project (%d) does not exist or not moveable by this user: %w",
			req.ProjectId, err)
	}

	return &apiv1.MoveProjectResponse{}, nil
}

func (a *apiServer) ArchiveProject(
	ctx context.Context, req *apiv1.ArchiveProjectRequest) (*apiv1.ArchiveProjectResponse,
	error,
) {
	p, _, err := a.getProjectAndCheckCanDoActions(ctx, req.Id,
		project.AuthZProvider.Get().CanArchiveProject)
	if err != nil {
		return nil, err
	}
	if err = a.CheckParentWorkspaceUnarchived(p); err != nil {
		return nil, err
	}

	holder := &projectv1.Project{}
	if err = a.m.db.QueryProto("archive_project", holder, req.Id, true); err != nil {
		return nil, fmt.Errorf("error archiving project (%d): %w", req.Id, err)
	}
	if holder.Id == 0 {
		return nil, fmt.Errorf("project (%d) is not archive-able by this user: %w",
			req.Id, err)
	}

	return &apiv1.ArchiveProjectResponse{}, nil
}

func (a *apiServer) UnarchiveProject(
	ctx context.Context, req *apiv1.UnarchiveProjectRequest) (*apiv1.UnarchiveProjectResponse,
	error,
) {
	p, _, err := a.getProjectAndCheckCanDoActions(ctx, req.Id,
		project.AuthZProvider.Get().CanUnarchiveProject)
	if err != nil {
		return nil, err
	}
	if err = a.CheckParentWorkspaceUnarchived(p); err != nil {
		return nil, err
	}

	holder := &projectv1.Project{}
	if err = a.m.db.QueryProto("archive_project", holder, req.Id, false); err != nil {
		return nil, fmt.Errorf("error unarchiving project (%d): %w", req.Id, err)
	}
	if holder.Id == 0 {
		return nil, fmt.Errorf("project (%d) is not unarchive-able by this user: %w",
			req.Id, err)
	}
	return &apiv1.UnarchiveProjectResponse{}, nil
}

func (a *apiServer) GetProjectsByUserActivity(
	ctx context.Context, req *apiv1.GetProjectsByUserActivityRequest,
) (*apiv1.GetProjectsByUserActivityResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	p := []*model.Project{}

	limit := req.Limit

	if limit > apiutils.MaxLimit {
		return nil, apiutils.ErrInvalidLimit
	}

	err = db.Bun().NewSelect().Model(p).NewRaw(`
	SELECT
		w.name AS workspace_name,
		u.username,
		p.id,
		p.name,
		p.archived,
		p.workspace_id,
		p.description,
		p.immutable,
		p.notes,
		p.user_id,
		'WORKSPACE_STATE_' || p.state AS state,
		p.error_message,
		COUNT(*) FILTER (WHERE e.project_id = p.id) AS num_experiments,
		COUNT(*) FILTER (WHERE e.project_id = p.id AND e.state = 'ACTIVE') AS num_active_experiments,
		MAX(e.start_time) FILTER (WHERE e.project_id = p.id) AS last_experiment_started_at
	FROM
		projects AS p
		INNER JOIN activity AS a ON p.id = a.entity_id AND a.user_id = ?
		LEFT JOIN users AS u ON u.id = p.user_id
		LEFT JOIN workspaces AS w ON w.id = p.workspace_id
		LEFT JOIN experiments AS e ON e.project_id = p.id
	GROUP BY
		p.id,
		u.username,
		w.name,
		a.activity_time
	ORDER BY
		a.activity_time DESC NULLS LAST
	LIMIT ?;
	`, curUser.ID, limit).
		Scan(ctx, &p)
	if err != nil {
		return nil, err
	}

	projects := model.ProjectsToProto(p)
	viewableProjects := []*projectv1.Project{}

	for _, pr := range projects {
		canView, err := project.AuthZProvider.Get().CanGetProject(ctx, *curUser, pr)
		if err != nil {
			return nil, err
		}
		if canView {
			viewableProjects = append(viewableProjects, pr)
		}
	}

	return &apiv1.GetProjectsByUserActivityResponse{Projects: viewableProjects}, nil
}
