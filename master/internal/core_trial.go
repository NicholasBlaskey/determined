package internal

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/uptrace/bun"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	detContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
)

func echoCanGetTrial(c echo.Context, m *Master, trialID string) error {
	id, err := strconv.Atoi(trialID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "trial ID must be numeric got %s", trialID)
	}

	curUser := c.(*detContext.DetContext).MustGetUser()
	trialNotFound := echo.NewHTTPError(http.StatusNotFound, "trial not found: %d", id)
	exp, err := m.db.ExperimentByTrialID(id)
	if errors.Is(err, db.ErrNotFound) {
		return trialNotFound
	} else if err != nil {
		return err
	}
	var ok bool
	ctx := c.Request().Context()
	if ok, err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, curUser, exp); err != nil {
		return err
	} else if !ok {
		return trialNotFound
	}

	if err = expauth.AuthZProvider.Get().CanGetExperimentArtifacts(ctx, curUser, exp); err != nil {
		return echo.NewHTTPError(http.StatusForbidden, err.Error())
	}
	return nil
}

func (m *Master) parseAndRBACTrialIDs(c echo.Context, in string) ([]int, error) {
	if len(in) == 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest,
			"expected at least one trial ID in the query param 'trialIds'")
	}

	var trialIDs []int
	for _, id := range strings.Split(in, ",") {
		if err := echoCanGetTrial(c, m, id); err != nil {
			return nil, err
		}

		trialID, err := strconv.Atoi(id)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf(
				"could not parse trialIds parameter expected comma separated ints got %s", in))
		}
		trialIDs = append(trialIDs, trialID)
	}

	return trialIDs, nil
}

//	@Summary	Stream one or more trial's training metrics.
//	@Tags		Trials
//	@ID			stream-training-metrics
//	@Produce	application/x-ndjson
//	@Param		trialIds	query	string	true	"Comma delimited trial IDs"
//	@Success	200					{}		string	"A new line delimated JSON stream with the following fields trialId,endTime,metrics,totalBatches,trialRunId,id"
//	@Router		/trials/metrics/training [get]
//
// nolint:lll
func (m *Master) streamTrainingMetrics(c echo.Context) error {
	trialIDs, err := m.parseAndRBACTrialIDs(c, c.QueryParam("trialIds"))
	if err != nil {
		return err
	}

	err = streamQueryAsNDJSON(c, `SELECT jsonb_build_object(
		'trialId', trial_id,
		'endTime', end_time,
		'metrics', metrics,
		'totalBatches', total_batches,
		'trialRunId', trial_run_id,
		'id', id
	) FROM steps WHERE trial_id IN (?) ORDER BY trial_id, trial_run_id, total_batches`,
		bun.In(trialIDs))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			errors.Wrap(err, "error streaming training metrics"))
	}
	return nil
}

//	@Summary	Stream one or more trial's validation metrics.
//	@Tags		Trials
//	@ID			stream-validation-metrics
//	@Produce	application/x-ndjson
//	@Param		trialIds	query	string	true	"Comma delimited trial IDs"
//	@Success	200					{}		string	"A new line delimated JSON stream with the following fields trialId,endTime,metrics,totalBatches,trialRunId,id"
//	@Router		/trials/metrics/validation [get]
//
// nolint:lll
func (m *Master) streamValidationMetrics(c echo.Context) error {
	trialIDs, err := m.parseAndRBACTrialIDs(c, c.QueryParam("trialIds"))
	if err != nil {
		return err
	}

	err = streamQueryAsNDJSON(c, `SELECT jsonb_build_object(
		'trialId', trial_id,
		'endTime', end_time,
		'metrics', metrics,
		'totalBatches', total_batches,
		'trialRunId', trial_run_id,
		'id', id
	) FROM validations WHERE trial_id IN (?) ORDER BY trial_id, trial_run_id, total_batches`,
		bun.In(trialIDs))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			errors.Wrap(err, "error streaming training metrics"))
	}
	return nil
}

func streamQueryAsNDJSON(c echo.Context, query string, args ...any) error {
	rows, err := db.Bun().QueryContext(c.Request().Context(), query, args...)
	if err != nil {
		return errors.Wrap(err, "error running query to stream")
	}
	defer rows.Close()

	c.Response().Header().Set(echo.HeaderContentType, "application/x-ndjson")
	c.Response().WriteHeader(http.StatusOK)

	for rows.Next() {
		var b sql.RawBytes
		err := rows.Scan(&b)
		if err != nil {
			return errors.Wrap(err, "error scanning results")
		}

		// TODO we do an O(n) copy everytime due to slice cap==len
		// when we append a newline. This still appears faster than two calls to Write().
		_, err = c.Response().Write(append(b, '\n'))
		if err != nil {
			return errors.Wrap(err, "error writing response")
		}
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "error running query to stream")
	}

	return nil
}

// TODO(ilia): These APIs are deprecated and will be removed in a future release.
func (m *Master) getTrial(c echo.Context) (interface{}, error) {
	if err := echoCanGetTrial(c, m, c.Param("trial_id")); err != nil {
		return nil, err
	}

	return m.db.RawQuery("get_trial", c.Param("trial_id"))
}

func (m *Master) getTrialMetrics(c echo.Context) (interface{}, error) {
	if err := echoCanGetTrial(c, m, c.Param("trial_id")); err != nil {
		return nil, err
	}

	return m.db.RawQuery("get_trial_metrics", c.Param("trial_id"))
}
