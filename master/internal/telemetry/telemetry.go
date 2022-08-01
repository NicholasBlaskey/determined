package telemetry

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/version"
)

const (
	minTickIntervalMins = 1
	maxTickIntervalMins = 2
)

type telemetryTick struct{}

// TelemetryActor manages gathering and sending telemetry data.
type TelemetryActor struct {
	db        db.DB
	client    analytics.Client
	clusterID string
}

// New creates an actor to handle collecting and sending telemetry information.
func New(
	db db.DB,
	clusterID string,
	segmentKey string,
) (*TelemetryActor, error) {
	fmt.Println("NEW ACTOR")
	client, err := analytics.NewWithConfig(
		segmentKey,
		analytics.Config{Logger: debugLogger{}},
	)
	if err != nil {
		return nil, err
	}

	if err := client.Enqueue(analytics.Identify{
		UserId: clusterID,
		Traits: analytics.Traits{
			"master_version": version.Version,
		},
	}); err != nil {
		logrus.WithError(err).Warnf("failed to enqueue identity %s", clusterID)
	}

	return &TelemetryActor{db, client, clusterID}, nil
}

// Receive implements the actor.Actor interface.
func (s *TelemetryActor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, 0, telemetryTick{})

	case analytics.Track:
		msg.UserId = s.clusterID
		if err := s.client.Enqueue(msg); err != nil {
			ctx.Log().WithError(err).Warnf("failed to enqueue track %s", msg.Event)
		}

	case telemetryTick:
		fmt.Println("TICKET")
		// Tick in a random interval.
		//nolint:gosec // Weak RNG is fine here.
		randNum := rand.Intn(maxTickIntervalMins-minTickIntervalMins) + minTickIntervalMins
		_ = randNum
		actors.NotifyAfter(ctx, time.Duration(15)*time.Second, telemetryTick{})

		ReportMasterTick(ctx.Self().System(), s.db)

	case actor.PostStop:
		_ = s.client.Close()
	}

	return nil
}
