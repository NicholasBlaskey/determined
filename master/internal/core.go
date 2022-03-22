package internal

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-systemd/activation"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/cluster"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/config"
	detContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/elastic"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/hpimportance"
	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/internal/prom"
	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/internal/template"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/master/version"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/masterv1"
)

const (
	maxConcurrentRestores = 10
	defaultAskTimeout     = 2 * time.Second
	webuiBaseRoute        = "/det"
)

// staticWebDirectoryPaths are the locations of static files that comprise the webui.
var staticWebDirectoryPaths = map[string]bool{
	"/docs":          true,
	webuiBaseRoute:   true,
	"/docs/rest-api": true,
}

// Master manages the Determined master state.
type Master struct {
	ClusterID string
	MasterID  string

	config   *config.Config
	taskSpec *tasks.TaskSpec

	logs          *logger.LogBuffer
	system        *actor.System
	echo          *echo.Echo
	rm            *actor.Ref
	rwCoordinator *actor.Ref
	db            *db.PgDB
	proxy         *actor.Ref
	taskLogger    *task.Logger
	hpImportance  *actor.Ref

	trialLogBackend TrialLogBackend
	taskLogBackend  task.LogBackend
}

// New creates an instance of the Determined master.
func New(logStore *logger.LogBuffer, config *config.Config) *Master {
	logger.SetLogrus(config.Log)
	return &Master{
		MasterID: uuid.New().String(),
		logs:     logStore,
		config:   config,
	}
}

func (m *Master) getConfig(echo.Context) (interface{}, error) {
	return m.config.Printable()
}

func (m *Master) getTaskContainerDefaults(poolName string) model.TaskContainerDefaultsConfig {
	// Always fall back to the top-level TaskContainerDefaults
	taskContainerDefaults := m.config.TaskContainerDefaults

	// Only look for pool settings with Agent resource managers.
	if m.config.ResourceManager.AgentRM != nil {
		// Iterate through configured pools looking for a TaskContainerDefaults setting.
		for _, pool := range m.config.ResourcePools {
			if poolName == pool.PoolName {
				if pool.TaskContainerDefaults == nil {
					break
				}
				taskContainerDefaults = *pool.TaskContainerDefaults
			}
		}
	}
	return taskContainerDefaults
}

// Info returns this master's information.
func (m *Master) Info() aproto.MasterInfo {
	telemetryInfo := aproto.TelemetryInfo{}

	if m.config.Telemetry.Enabled && m.config.Telemetry.SegmentWebUIKey != "" {
		// Only advertise a Segment WebUI key if a key has been configured and
		// telemetry is enabled.
		telemetryInfo.Enabled = true
		telemetryInfo.SegmentKey = m.config.Telemetry.SegmentWebUIKey
	}

	return aproto.MasterInfo{
		ClusterID:   m.ClusterID,
		MasterID:    m.MasterID,
		Version:     version.Version,
		Telemetry:   telemetryInfo,
		ClusterName: m.config.ClusterName,
	}
}

func (m *Master) getInfo(echo.Context) (interface{}, error) {
	return m.Info(), nil
}

func (m *Master) getMasterLogs(c echo.Context) (interface{}, error) {
	args := struct {
		LessThanID    *int `query:"less_than_id"`
		GreaterThanID *int `query:"greater_than_id"`
		Limit         *int `query:"tail"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}

	limit := -1
	if args.Limit != nil {
		limit = *args.Limit
	}

	startID := -1
	if args.GreaterThanID != nil {
		startID = *args.GreaterThanID + 1
	}

	endID := -1
	if args.LessThanID != nil {
		endID = *args.LessThanID
	}

	entries := m.logs.Entries(startID, endID, limit)
	if len(entries) == 0 {
		// Return a zero-length array here so the JSON encoding is `[]` rather than `null`.
		entries = make([]*logger.Entry, 0)
	}
	return entries, nil
}

// @Summary Get a detailed view of resource allocation during the given time period (CSV).
// @Tags Cluster
// @ID get-raw-resource-allocation-csv
// @Accept  json
// @Produce  text/csv
//nolint:lll
// @Param   timestamp_after query string true "Start time to get allocations for (YYYY-MM-DDTHH:MM:SSZ format)"
//nolint:lll
// @Param   timestamp_before query string true "End time to get allocations for (YYYY-MM-DDTHH:MM:SSZ format)"
//nolint:lll
// @Success 200 {} string "A CSV file containing the fields experiment_id,kind,username,labels,slots,start_time,end_time,seconds"
//nolint:godot
// @Router /allocation/raw [get]
// @Deprecated
func (m *Master) getRawResourceAllocation(c echo.Context) error {
	args := struct {
		Start string `query:"timestamp_after"`
		End   string `query:"timestamp_before"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return err
	}

	start, err := time.Parse("2006-01-02T15:04:05Z", args.Start)
	if err != nil {
		return errors.Wrap(err, "invalid start time")
	}
	end, err := time.Parse("2006-01-02T15:04:05Z", args.End)
	if err != nil {
		return errors.Wrap(err, "invalid end time")
	}
	if start.After(end) {
		return errors.New("start time cannot be after end time")
	}

	resp := &apiv1.ResourceAllocationRawResponse{}
	if err := m.db.QueryProto(
		"get_raw_allocation", &resp.ResourceEntries, start.UTC(), end.UTC(),
	); err != nil {
		return errors.Wrap(err, "error fetching allocation data")
	}

	c.Response().Header().Set("Content-Type", "text/csv")

	labelEscaper := strings.NewReplacer("\\", "\\\\", ",", "\\,")
	csvWriter := csv.NewWriter(c.Response())
	formatTimestamp := func(ts *timestamppb.Timestamp) string {
		if ts == nil {
			return ""
		}
		return ts.AsTime().Format(time.RFC3339Nano)
	}

	header := []string{
		"experiment_id", "kind", "username", "labels", "slots", "start_time", "end_time", "seconds",
	}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	for _, entry := range resp.ResourceEntries {
		var labels []string
		for _, label := range entry.Labels {
			labels = append(labels, labelEscaper.Replace(label))
		}
		fields := []string{
			strconv.Itoa(int(entry.ExperimentId)), entry.Kind, entry.Username, strings.Join(labels, ","),
			strconv.Itoa(int(entry.Slots)), formatTimestamp(entry.StartTime), formatTimestamp(entry.EndTime),
			fmt.Sprintf("%f", entry.Seconds),
		}
		if err := csvWriter.Write(fields); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return nil
}

func (m *Master) fetchAggregatedResourceAllocation(
	req *apiv1.ResourceAllocationAggregatedRequest,
) (*apiv1.ResourceAllocationAggregatedResponse, error) {
	resp := &apiv1.ResourceAllocationAggregatedResponse{}

	switch req.Period {
	case masterv1.ResourceAllocationAggregationPeriod_RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY:
		start, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			return nil, errors.Wrap(err, "invalid start date")
		}
		end, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			return nil, errors.Wrap(err, "invalid end date")
		}
		if start.After(end) {
			return nil, errors.New("start date cannot be after end date")
		}

		if err := m.db.QueryProto(
			"get_aggregated_allocation", &resp.ResourceEntries, start.UTC(), end.UTC(),
		); err != nil {
			return nil, errors.Wrap(err, "error fetching aggregated allocation data")
		}

		return resp, nil

	case masterv1.ResourceAllocationAggregationPeriod_RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY:
		start, err := time.Parse("2006-01", req.StartDate)
		if err != nil {
			return nil, errors.Wrap(err, "invalid start date")
		}
		end, err := time.Parse("2006-01", req.EndDate)
		if err != nil {
			return nil, errors.Wrap(err, "invalid end date")
		}
		end = end.AddDate(0, 1, -1)
		if start.After(end) {
			return nil, errors.New("start date cannot be after end date")
		}

		if err := m.db.QueryProto(
			"get_monthly_aggregated_allocation", &resp.ResourceEntries, start.UTC(), end.UTC(),
		); err != nil {
			return nil, errors.Wrap(err, "error fetching aggregated allocation data")
		}

		return resp, nil

	default:
		return nil, errors.New("no aggregation period specified")
	}
}

// @Summary Get an aggregated view of resource allocation during the given time period (CSV).
// @Tags Cluster
// @ID get-aggregated-resource-allocation-csv
// @Produce  text/csv
//nolint:lll
// @Param   start_date query string true "Start time to get allocations for (YYYY-MM-DD format for daily, YYYY-MM format for monthly)"
//nolint:lll
// @Param   end_date query string true "End time to get allocations for (YYYY-MM-DD format for daily, YYYY-MM format for monthly)"
//nolint:lll
// @Param   period query string true "Period to aggregate over (RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY or RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY)"
// @Success 200 {} string "aggregation_type,aggregation_key,date,seconds"
//nolint:godot
// @Router /allocation/aggregated [get]
func (m *Master) getAggregatedResourceAllocation(c echo.Context) error {
	args := struct {
		Start  string `query:"start_date"`
		End    string `query:"end_date"`
		Period string `query:"period"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return err
	}

	resp, err := m.fetchAggregatedResourceAllocation(&apiv1.ResourceAllocationAggregatedRequest{
		StartDate: args.Start,
		EndDate:   args.End,
		Period: masterv1.ResourceAllocationAggregationPeriod(
			masterv1.ResourceAllocationAggregationPeriod_value[args.Period],
		),
	})

	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Type", "text/csv")

	csvWriter := csv.NewWriter(c.Response())

	header := []string{"aggregation_type", "aggregation_key", "date", "seconds"}
	if err = csvWriter.Write(header); err != nil {
		return err
	}

	write := func(aggType, aggKey, start string, seconds float32) error {
		return csvWriter.Write([]string{aggType, aggKey, start, fmt.Sprintf("%f", seconds)})
	}

	for _, entry := range resp.ResourceEntries {
		writeAggType := func(agg string, vals map[string]float32) error {
			for key, seconds := range vals {
				if err = write(agg, key, entry.PeriodStart, seconds); err != nil {
					return err
				}
			}
			return nil
		}
		if err = writeAggType("experiment_label", entry.ByExperimentLabel); err != nil {
			return err
		}
		if err = writeAggType("username", entry.ByUsername); err != nil {
			return err
		}
		if err = writeAggType("resource_pool", entry.ByResourcePool); err != nil {
			return err
		}
		if err = writeAggType("agent_label", entry.ByAgentLabel); err != nil {
			return err
		}
		if err = writeAggType("total", map[string]float32{"total": entry.Seconds}); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return nil
}

func (m *Master) getSystemdListener() (net.Listener, error) {
	switch systemdListeners, err := activation.Listeners(); {
	case err != nil:
		return nil, errors.Wrap(err, "failed to find systemd listeners")
	case len(systemdListeners) == 0:
		return nil, nil
	case len(systemdListeners) == 1:
		return systemdListeners[0], nil
	default:
		return nil, errors.Errorf("expected at most 1 systemd listener, got %d", len(systemdListeners))
	}
}

func (m *Master) findListeningPort(listener net.Listener) (uint16, error) {
	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		return 0, errors.New("listener is not a TCP listener")
	}

	file, err := tcpListener.File()
	if err != nil {
		return 0, err
	}
	link, err := os.Readlink(fmt.Sprintf("/proc/self/fd/%d", file.Fd()))
	if err != nil {
		return 0, err
	}
	matches := regexp.MustCompile(`socket:\[(.*)\]`).FindStringSubmatch(link)
	inode := matches[1]
	tcp, err := os.Open("/proc/self/net/tcp")
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tcp.Close()
	}()

	lines := bufio.NewScanner(tcp)
	for lines.Scan() {
		fields := strings.Fields(lines.Text())
		if fields[9] == inode {
			addr := fields[1]
			port, err := strconv.ParseInt(strings.Split(addr, ":")[1], 16, 16)
			if err != nil {
				return 0, err
			}
			return uint16(port), nil
		}
	}

	return 0, errors.New("listener not found")
}

func (m *Master) startServers(ctx context.Context, cert *tls.Certificate) error {
	// Create the base socket listener by either fetching one passed to us from systemd or creating a
	// TCP listener manually.
	var baseListener net.Listener
	systemdListener, err := m.getSystemdListener()
	switch {
	case err != nil:
		return errors.Wrap(err, "failed to find systemd listeners")
	case systemdListener != nil:
		baseListener = systemdListener
		port, pErr := m.findListeningPort(systemdListener)
		if pErr != nil {
			return pErr
		}
		log.Infof("found port %d for systemd listener", port)
		m.config.Port = int(port)
	default:
		baseListener, err = net.Listen("tcp", fmt.Sprintf(":%d", m.config.Port))
		if err != nil {
			return err
		}
	}
	defer closeWithErrCheck("base", baseListener)

	// If configured, set up TLS wrapping.
	if cert != nil {
		baseListener = tls.NewListener(baseListener, &tls.Config{
			Certificates:             []tls.Certificate{*cert},
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
		})
	}

	// This must be before grpcutil.RegisterHTTPProxy is called since it may use stuff set up by the
	// gRPC server (logger initialization, maybe more). Found by --race.
	gRPCServer := grpcutil.NewGRPCServer(m.db, &apiServer{m: m},
		m.config.Observability.EnablePrometheus,
		&m.config.InternalConfig.ExternalSessions)

	err = grpcutil.RegisterHTTPProxy(ctx, m.echo, m.config.Port, cert)
	if err != nil {
		return errors.Wrap(err, "failed to register gRPC gateway")
	}

	// Initialize listeners and multiplexing.
	mux := cmux.New(baseListener)

	grpcListener := mux.MatchWithWriters(
		cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"),
	)
	defer closeWithErrCheck("grpc", grpcListener)

	httpListener := mux.Match(cmux.HTTP1(), cmux.HTTP2())
	defer closeWithErrCheck("http", httpListener)

	// Start all servers and return the first error. This leaks a channel, but the complexity of
	// perfectly handling cleanup and all the error cases doesn't seem worth it for a function that is
	// called exactly once and causes the whole process to exit immediately when it returns.
	errs := make(chan error)
	start := func(name string, run func() error) {
		go func() {
			errs <- errors.Wrap(run(), name+" failed")
		}()
	}
	start("gRPC server", func() error {
		// We should defer srv.Stop() here, but cmux does not unblock accept calls when underlying
		// listeners close and grpc-go depends on cmux unblocking and closing, Stop() blocks
		// indefinitely when using cmux.
		// To be fixed by https://github.com/soheilhy/cmux/pull/69 which makes cmux an io.Closer.
		return gRPCServer.Serve(grpcListener)
	})
	start("HTTP server", func() error {
		m.echo.Listener = httpListener
		m.echo.HidePort = true
		defer closeWithErrCheck("echo", m.echo)
		return m.echo.StartServer(m.echo.Server)
	})
	start("cmux listener", mux.Serve)

	if systemdListener != nil {
		log.Infof("accepting incoming connections on a socket inherited from systemd")
	} else {
		log.Infof("accepting incoming connections on port %d", m.config.Port)
	}
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func closeWithErrCheck(name string, closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Errorf("error closing closer %s: %s", name, err)
	}
}

func (m *Master) tryRestoreExperiment(sema chan struct{}, wg *sync.WaitGroup, e *model.Experiment) {
	sema <- struct{}{}
	defer func() { <-sema }()
	defer func() { wg.Done() }()

	if err := m.restoreExperiment(e); err != nil {
		log.WithError(err).Errorf("failed to restore experiment: %d", e.ID)
		e.State = model.ErrorState
		if err := m.db.TerminateExperimentInRestart(e.ID, e.State); err != nil {
			log.WithError(err).Error("failed to mark experiment as errored")
		}
		telemetry.ReportExperimentStateChanged(m.system, m.db, *e)
	}
}

// TODO XXX wait only for open experiments-trials.
func (m *Master) restoreNonTerminalExperiments() error {
	// Restore non-terminal experiments from the database.
	// Limit the number of concurrent restores at any time within the system to maxConcurrentRestores.
	// This has avoided resource exhaustion in the past (on the db connection pool) and probably is
	// good still to avoid overwhelming us on restart after a crash.
	sema := make(chan struct{}, maxConcurrentRestores)
	m.system.ActorOf(actor.Addr("experiments"), &actors.Group{})
	m.system.ActorOf(job.JobsActorAddr, job.NewJobs(sproto.GetCurrentRM(m.system)))
	toRestore, err := m.db.NonTerminalExperiments()
	if err != nil {
		return errors.Wrap(err, "couldn't retrieve experiments to restore")
	}

	wg := sync.WaitGroup{}
	for _, exp := range toRestore {
		wg.Add(1)
		go m.tryRestoreExperiment(sema, &wg, exp)
	}

	wg.Wait()

	// TODO XXX crutch: need to wait for experiment to restore trials and check if allocation can be restored.
	time.Sleep(1 * time.Second)

	return nil
}

// convertDBErrorsToNotFound helps reduce boilerplate in our handlers, by
// classifying database "not found" errors as HTTP "not found" errors.
func convertDBErrorsToNotFound(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if errors.Cause(err) == db.ErrNotFound {
			return echo.ErrNotFound
		}
		return err
	}
}

func (m *Master) rwCoordinatorWebSocket(socket *websocket.Conn, c echo.Context) error {
	c.Logger().Infof(
		"new connection for RW Coordinator from: %v, %s",
		socket.RemoteAddr(),
		c.Request().URL,
	)

	resourceName := c.Request().URL.Path
	query := c.Request().URL.Query()

	readLockString, ok := query["read_lock"]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Received request without specifying read_lock: %v", c.Request().URL))
	}

	var readLock bool
	if strings.EqualFold(readLockString[0], "True") {
		readLock = true
	} else {
		if !strings.EqualFold(readLockString[0], "false") {
			return echo.NewHTTPError(http.StatusBadRequest,
				fmt.Sprintf("Received request with invalid read_lock: %v", c.Request().URL))
		}
		readLock = false
	}

	socketActor := m.system.AskAt(actor.Addr("rwCoordinator"),
		resourceRequest{resourceName, readLock, socket})
	actorRef, ok := socketActor.Get().(*actor.Ref)
	if !ok {
		c.Logger().Error("failed to get websocket actor")
		return nil
	}

	// Wait for the websocket actor to terminate.
	return actorRef.AwaitTermination()
}

func updateClusterHeartbeat(ctx context.Context, db *db.PgDB) {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		currentTime := time.Now().UTC().Truncate(time.Millisecond)
		err := db.UpdateClusterHeartBeat(currentTime)
		if err != nil {
			log.Error(err.Error())
		}
		select {
		case <-t.C:
		case <-ctx.Done():
			return
		}
	}
}

func (m *Master) postTaskLogs(c echo.Context) (interface{}, error) {
	var logs []*model.TaskLog
	if err := json.NewDecoder(c.Request().Body).Decode(&logs); err != nil {
		return "", err
	}
	if err := m.taskLogBackend.AddTaskLogs(logs); err != nil {
		return "", errors.Wrap(err, "receiving task logs")
	}
	return "", nil
}

// Run causes the Determined master to connect the database and begin listening for HTTP requests.
func (m *Master) Run(ctx context.Context) error {
	log.Infof("Determined master %s (built with %s)", version.Version, runtime.Version())

	var err error

	if err = etc.SetRootPath(filepath.Join(m.config.Root, "static/srv")); err != nil {
		return errors.Wrap(err, "could not set static root")
	}

	m.db, err = db.Setup(&m.config.DB)
	if err != nil {
		return err
	}
	defer closeWithErrCheck("db", m.db)

	m.ClusterID, err = m.db.GetOrCreateClusterID()
	if err != nil {
		return errors.Wrap(err, "could not fetch cluster id from database")
	}
	cert, err := m.config.Security.TLS.ReadCertificate()
	if err != nil {
		return errors.Wrap(err, "failed to read TLS certificate")
	}
	m.taskSpec = &tasks.TaskSpec{
		ClusterID:             m.ClusterID,
		HarnessPath:           filepath.Join(m.config.Root, "wheels"),
		TaskContainerDefaults: m.config.TaskContainerDefaults,
		MasterCert:            cert,
		SSHRsaSize:            m.config.Security.SSH.RsaKeySize,
		SegmentEnabled:        m.config.Telemetry.Enabled && m.config.Telemetry.SegmentMasterKey != "",
		SegmentAPIKey:         m.config.Telemetry.SegmentMasterKey,
	}

	go m.cleanUpExperimentSnapshots()

	// Actor structure:
	// master system
	// +- Agent Group (actors.Group: agents)
	//     +- Agent (internal.agent: <agent-id>)
	//         +- Websocket (actors.WebSocket: <remote-address>)
	// +- ResourceManagers (scheduler.ResourceManagers: resourceManagers)
	// Exactly one of the resource managers is enabled at a time.
	// +- AgentResourceManager (resourcemanagers.AgentResourceManager: agentRM)
	//     +- Resource Pool (resourcemanagers.ResourcePool: <resource-pool-name>)
	//         +- Provisioner (provisioner.Provisioner: provisioner)
	// +- KubernetesResourceManager (scheduler.KubernetesResourceManager: kubernetesRM)
	// +- Service Proxy (proxy.Proxy: proxy)
	// +- RWCoordinator (internal.rw_coordinator: rwCoordinator)
	// +- Telemetry (telemetry.telemetry: telemetry)
	// +- TrialLogger (internal.trialLogger: trialLogger)
	// +- Experiments (actors.Group: experiments)
	//     +- Experiment (internal.experiment: <experiment-id>)
	//         +- Trial (internal.trial: <trial-request-id>)
	//             +- Websocket (actors.WebSocket: <remote-address>)
	m.system = actor.NewSystemWithRoot("master", actor.ActorFunc(root))

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		sErr := m.system.Ref.AwaitTermination()
		log.WithError(sErr).Error("actor system exited")
		cancel()
	}()

	switch {
	case m.config.Logging.DefaultLoggingConfig != nil:
		m.trialLogBackend = m.db
		m.taskLogBackend = m.db
	case m.config.Logging.ElasticLoggingConfig != nil:
		es, eErr := elastic.Setup(*m.config.Logging.ElasticLoggingConfig)
		if eErr != nil {
			return eErr
		}
		m.trialLogBackend = es
		m.taskLogBackend = es
	default:
		panic("unsupported logging backend")
	}
	m.taskLogger = task.NewLogger(m.system, m.taskLogBackend)

	userService, err := user.New(m.db, m.system, &m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return errors.Wrap(err, "cannot initialize user manager")
	}
	authFuncs := []echo.MiddlewareFunc{userService.ProcessAuthentication}

	m.proxy, _ = m.system.ActorOf(actor.Addr("proxy"), &proxy.Proxy{})

	task.InitAllocationMap()
	m.system.MustActorOf(actor.Addr("allocation-aggregator"), &allocationAggregator{db: m.db})

	hpi, err := hpimportance.NewManager(m.db, m.system, m.config.HPImportance, m.config.Root)
	if err != nil {
		return err
	}
	m.hpImportance, _ = m.system.ActorOf(actor.Addr(hpimportance.RootAddr), hpi)

	// Initialize the HTTP server and listen for incoming requests.
	m.echo = echo.New()
	m.echo.Use(middleware.Recover())

	gzipConfig := middleware.GzipConfig{
		Skipper: func(c echo.Context) bool {
			return !staticWebDirectoryPaths[c.Path()]
		},
	}
	m.echo.Use(middleware.GzipWithConfig(gzipConfig))

	m.echo.Use(middleware.AddTrailingSlashWithConfig(middleware.TrailingSlashConfig{
		Skipper: func(c echo.Context) bool {
			return !staticWebDirectoryPaths[c.Path()]
		},
		RedirectCode: http.StatusMovedPermanently,
	}))
	setupEchoRedirects(m)

	if m.config.EnableCors {
		m.echo.Use(api.CORSWithTargetedOrigin)
	}

	// Add resistance to common HTTP attacks.
	//
	// TODO(DET-1696): Enable Content Security Policy (CSP).
	secureConfig := middleware.SecureConfig{
		Skipper:            middleware.DefaultSkipper,
		XSSProtection:      "1; mode=block",
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "SAMEORIGIN",
	}
	m.echo.Use(middleware.SecureWithConfig(secureConfig))

	// Register middleware that extends default context.
	m.echo.Use(func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &detContext.DetContext{Context: c}
			return h(cc)
		}
	})

	m.echo.Use(convertDBErrorsToNotFound)

	if m.config.InternalConfig.AuditLoggingEnabled {
		m.echo.Use(auditLogMiddleware())
	}

	if m.config.Telemetry.OtelEnabled {
		configureOtel(m.config.Telemetry.OtelExportedOtlpEndpoint)
		m.echo.Use(otelecho.Middleware("determined-master"))
	}

	m.echo.Logger = logger.New()
	m.echo.HideBanner = true
	m.echo.HTTPErrorHandler = api.JSONErrorHandler

	// Resource Manager.
	agentOpts := &aproto.MasterSetAgentOptions{
		MasterInfo:     m.Info(),
		LoggingOptions: m.config.Logging,
	}
	m.rm = resourcemanagers.Setup(m.system, m.db, m.echo, m.config.ResourceConfig, agentOpts, cert)
	tasksGroup := m.echo.Group("/tasks", authFuncs...)
	tasksGroup.GET("", api.Route(m.getTasks))
	tasksGroup.GET("/:task_id", api.Route(m.getTask))

	// Distributed lock server.
	rwCoordinator := newRWCoordinator()
	m.rwCoordinator, _ = m.system.ActorOf(actor.Addr("rwCoordinator"), rwCoordinator)

	if err := m.restoreNonTerminalExperiments(); err != nil {
		return err
	}

	// End stats for dangling agents/instances in case of master crushed
	err = m.db.EndAllAgentStats()
	if err != nil {
		return errors.Wrap(err, "could not update end stats for agents")
	}
	err = m.db.EndAllInstanceStats()
	if err != nil {
		return errors.Wrap(err, "could not update end stats for instances")
	}

	if err = m.db.FailDeletingExperiment(); err != nil {
		return err
	}
	// TODO XXX crutch: only cleanup ones that has failed to restore.
	const CLOSE_OPEN_ALLOCATIONS bool = false
	if CLOSE_OPEN_ALLOCATIONS {
		if err = m.db.CloseOpenAllocations(); err != nil {
			return err
		}
		if err = task.WipeResourcesState(); err != nil {
			return err
		}
		if err = m.db.EndAllTaskStats(); err != nil {
			return err
		}
	}

	// The below function call is intentionally made after the call to CloseOpenAllocations.
	// This ensures that in the scenario where a cluster fails all open allocations are
	// set to the last cluster heartbeat when the cluster was running.
	cluster.InitTheLastBootClusterHeartbeat()
	go updateClusterHeartbeat(ctx, m.db)

	// Docs and WebUI.
	webuiRoot := filepath.Join(m.config.Root, "webui")
	reactRoot := filepath.Join(webuiRoot, "react")
	reactRootAbs, err := filepath.Abs(reactRoot)
	if err != nil {
		return errors.Wrap(err, "failed to get absolute path to react root")
	}
	reactIndex := filepath.Join(reactRoot, "index.html")

	// Docs.
	m.echo.Static("/docs/rest-api", filepath.Join(webuiRoot, "docs", "rest-api"))
	m.echo.Static("/docs", filepath.Join(webuiRoot, "docs"))

	webuiGroup := m.echo.Group(webuiBaseRoute)
	webuiGroup.File("/", reactIndex)
	webuiGroup.GET("/*", func(c echo.Context) error {
		groupPath := strings.TrimPrefix(c.Request().URL.Path, webuiBaseRoute+"/")
		requestedFile := filepath.Join(reactRoot, groupPath)
		// We do a simple check against directory traversal attacks.
		requestedFileAbs, fErr := filepath.Abs(requestedFile)
		if fErr != nil {
			log.WithError(fErr).Error("failed to get absolute path to requested file")
			return c.File(reactIndex)
		}
		isInReactDir := strings.HasPrefix(requestedFileAbs, reactRootAbs)
		if !isInReactDir {
			return echo.NewHTTPError(http.StatusForbidden)
		}

		var hasMatchingFile bool
		stat, oErr := os.Stat(requestedFile)
		switch {
		case os.IsNotExist(oErr):
		case os.IsPermission(oErr):
			hasMatchingFile = false
		case oErr != nil:
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to check if file exists")
		default:
			hasMatchingFile = !stat.IsDir()
		}
		if hasMatchingFile {
			return c.File(requestedFile)
		}

		return c.File(reactIndex)
	})

	m.echo.Static("/api/v1/api.swagger.json",
		filepath.Join(m.config.Root, "swagger/determined/api/v1/api.swagger.json"))

	m.echo.GET("/config", api.Route(m.getConfig))
	m.echo.GET("/info", api.Route(m.getInfo))
	m.echo.GET("/logs", api.Route(m.getMasterLogs), authFuncs...)

	experimentsGroup := m.echo.Group("/experiments", authFuncs...)
	experimentsGroup.GET("/:experiment_id/model_def", m.getExperimentModelDefinition)
	experimentsGroup.GET("/:experiment_id/preview_gc", api.Route(m.getExperimentCheckpointsToGC))
	experimentsGroup.PATCH("/:experiment_id", api.Route(m.patchExperiment))
	experimentsGroup.POST("", api.Route(m.postExperiment))

	searcherGroup := m.echo.Group("/searcher", authFuncs...)
	searcherGroup.POST("/preview", api.Route(m.getSearcherPreview))

	trialsGroup := m.echo.Group("/trials", authFuncs...)
	trialsGroup.GET("/:trial_id", api.Route(m.getTrial))
	trialsGroup.GET("/:trial_id/metrics", api.Route(m.getTrialMetrics))

	resourcesGroup := m.echo.Group("/resources", authFuncs...)
	resourcesGroup.GET("/allocation/raw", m.getRawResourceAllocation)
	resourcesGroup.GET("/allocation/aggregated", m.getAggregatedResourceAllocation)

	m.echo.POST("/task-logs", api.Route(m.postTaskLogs))

	// used in as a part of the data layer API (to be removed) in harness/determined/_data_layer
	// see https://docs.determined.ai/latest/training-apis/data-layer.html#using-the-data-layer-api
	m.echo.GET("/ws/data-layer/*",
		api.WebSocketRoute(m.rwCoordinatorWebSocket))

	m.echo.Any("/debug/pprof/*", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	m.echo.Any("/debug/pprof/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	m.echo.Any("/debug/pprof/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	m.echo.Any("/debug/pprof/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	m.echo.Any("/debug/pprof/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))

	if m.config.Observability.EnablePrometheus {
		p := prometheus.NewPrometheus("echo", nil)
		p.Use(m.echo)
		m.echo.Any("/debug/prom/metrics", echo.WrapHandler(promhttp.Handler()))
		m.echo.Any("/prom/det-state-metrics",
			echo.WrapHandler(promhttp.HandlerFor(prom.DetStateMetrics, promhttp.HandlerOpts{})))
		m.echo.Any("/prom/det-http-sd-config",
			api.Route(m.getPrometheusTargets))
	}

	handler := m.system.AskAt(actor.Addr("proxy"), proxy.NewProxyHandler{ServiceID: "service"})
	m.echo.Any("/proxy/:service/*", handler.Get().(echo.HandlerFunc))

	user.RegisterAPIHandler(m.echo, userService, authFuncs...)
	command.RegisterAPIHandler(
		m.system,
		m.echo,
		m.db,
		m.taskLogger,
		authFuncs...,
	)
	template.RegisterAPIHandler(m.echo, m.db, authFuncs...)

	telemetry.Setup(
		m.system,
		m.db,
		m.ClusterID,
		m.config.Telemetry,
	)

	return m.startServers(ctx, cert)
}
