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

func (m *Master) EchoMetricsStream(c echo.Context) error {
	ids := c.QueryParam("trialIds")

	fmt.Println(ids)
	var trialIDs []int
	for _, id := range strings.Split(ids, ",") {
		trialID, err := strconv.Atoi(id)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf(
				"could not parse trialIds parameter expected comma seperated ints got %s"), ids)
		}
		trialIDs = append(trialIDs, trialID)
	}

	rows, err := db.Bun().QueryContext(c.Request().Context(), `SELECT jsonb_build_object(
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
			errors.Wrap(err, "error querying for trial training metrics"))
	}
	defer rows.Close()

	c.Response().Header().Set(echo.HeaderContentType, "application/x-ndjson")
	c.Response().WriteHeader(http.StatusOK)

	for rows.Next() {
		var b sql.RawBytes
		err := rows.Scan(&b)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				errors.Wrap(err, "error scanning result for trial training metrics"))
		}

		// TODO we do an O(n) copy everytime due to slice cap==len
		// and we append a newline. This still appears faster than two calls to Write().
		_, err = c.Response().Write(append(b, '\n'))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				errors.Wrap(err, "error writing response for trial training metrics"))
		}
		// break
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
