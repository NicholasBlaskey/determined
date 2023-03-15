package grpcutil

import (
	"context"
	"crypto/tls"
	"fmt"
	"runtime/debug"
	"time"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpclogrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	proto "github.com/determined-ai/determined/proto/pkg/apiv1"
)

const jsonPretty = "application/json+pretty"

var kaep = keepalive.EnforcementPolicy{
	MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
	PermitWithoutStream: true,            // Allow pings even when there are no active streams
}

var kasp = keepalive.ServerParameters{
	MaxConnectionIdle:     15 * time.Second, // If a client is idle for 15 seconds, send a GOAWAY
	MaxConnectionAge:      30 * time.Second, // If any connection is alive for more than 30 seconds, send a GOAWAY
	MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before forcibly closing connections
	Time:                  5 * time.Second,  // Ping the client if it is idle for 5 seconds to ensure the connection is still active
	Timeout:               1 * time.Second,  // Wait 1 second for the ping ack before assuming the connection is dead
}

// NewGRPCServer creates a Determined gRPC service.
func NewGRPCServer(db *db.PgDB, srv proto.DeterminedServer, enablePrometheus bool,
	extConfig *model.ExternalSessions,
) *grpc.Server {
	// In go-grpc, the INFO log level is used primarily for debugging
	// purposes, so omit INFO messages from the master log.
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	logEntry := logrus.NewEntry(logger)
	grpclogrus.ReplaceGrpcLogger(logEntry)

	opts := []grpclogrus.Option{
		grpclogrus.WithLevels(grpcCodeToLogrusLevel),
	}

	streamInterceptors := []grpc.StreamServerInterceptor{
		grpclogrus.StreamServerInterceptor(logEntry, opts...),
		grpcrecovery.StreamServerInterceptor(),
		streamAuthInterceptor(db, extConfig),
	}

	unaryInterceptors := []grpc.UnaryServerInterceptor{
		grpclogrus.UnaryServerInterceptor(logEntry, opts...),
		grpcrecovery.UnaryServerInterceptor(grpcrecovery.WithRecoveryHandler(
			func(p interface{}) (err error) {
				logEntry.Error(string(debug.Stack()))
				return status.Errorf(codes.Internal, "%s", p)
			},
		)),
		unaryAuthInterceptor(db, extConfig),
		authZInterceptor(),
	}

	if enablePrometheus {
		streamInterceptors = append(streamInterceptors, grpc_prometheus.StreamServerInterceptor)
		unaryInterceptors = append(unaryInterceptors, grpc_prometheus.UnaryServerInterceptor)
		grpc_prometheus.EnableHandlingTimeHistogram()
	}

	grpcS := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
		grpc.StreamInterceptor(grpcmiddleware.ChainStreamServer(streamInterceptors...)),
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(unaryInterceptors...)),
		// Allow receiving messages _slightly_ larger than the maximum allowed context
		// directory. We should either just move these back to echo or have a chunker for
		// .tar.gz files long term.
		grpc.MaxRecvMsgSize(1024*1024*1024),
		grpc.MaxSendMsgSize(1024*1024*1024),
		grpc.MaxMsgSize(1024*1024*1024),
	)
	proto.RegisterDeterminedServer(grpcS, srv)
	return grpcS
}

// newGRPCGatewayMux creates a new gRPC server mux.
func newGRPCGatewayMux() *runtime.ServeMux {
	serverOpts := []runtime.ServeMuxOption{
		runtime.WithMarshalerOption(jsonPretty,
			&runtime.JSONPb{EmitDefaults: true, Indent: "    "}),
		runtime.WithMarshalerOption(runtime.MIMEWildcard,
			&runtime.JSONPb{EmitDefaults: true}),
		runtime.WithProtoErrorHandler(errorHandler),
		runtime.WithForwardResponseOption(userTokenResponse),
	}
	return runtime.NewServeMux(serverOpts...)
}

var kacp = keepalive.ClientParameters{
	Time:                10 * time.Second, // send pings every 10 seconds if there is no activity
	Timeout:             time.Second,      // wait 1 second for ping ack before considering the connection dead
	PermitWithoutStream: true,             // send pings even without active streams
}

// RegisterHTTPProxy registers grpc-gateway with the master echo server.
func RegisterHTTPProxy(ctx context.Context, e *echo.Echo, port int, cert *tls.Certificate) error {
	addr := fmt.Sprintf(":%d", port)
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024 * 1024 * 1024)),
		grpc.WithNoProxy(),
		grpc.WithKeepaliveParams(kacp),
	}
	if cert == nil {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		// Since this connection is coming directly back to this process, we can skip verification.
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		})))
	}
	mux := newGRPCGatewayMux()
	err := proto.RegisterDeterminedHandlerFromEndpoint(ctx, mux, addr, opts)
	if err != nil {
		return err
	}
	handler := func(c echo.Context) error {
		request := c.Request()
		if c.Request().Header.Get("Authorization") == "" {
			if cookie, err := c.Cookie(cookieName); err == nil {
				request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cookie.Value))
			}
		}
		if _, ok := request.URL.Query()["pretty"]; ok {
			request.Header.Set("Accept", jsonPretty)
		}
		mux.ServeHTTP(c.Response(), request)
		return nil
	}
	apiV1 := e.Group("/api/v1")
	apiV1.Any("/*", handler, middleware.RemoveTrailingSlash())
	return nil
}
