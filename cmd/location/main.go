package main

import (
	"context"
	"github.com/gidyon/micros/utils/healthcheck"
	"google.golang.org/grpc"
	"os"
	"strings"

	location_app "github.com/gidyon/pandemic-api/internal/services/location"

	"github.com/gidyon/pandemic-api/pkg/api/location"
	"github.com/gidyon/pandemic-api/pkg/api/messaging"

	"github.com/gidyon/config"
	"github.com/gidyon/micros"

	"github.com/Sirupsen/logrus"
)

func main() {
	cfg, err := config.New()
	handleErr(err)

	ctx := context.Background()

	app, err := micros.NewService(ctx, cfg, nil)
	handleErr(err)

	// Readiness health check
	app.AddEndpoint("/api/v1/locations/readyq/", healthcheck.RegisterProbe(&healthcheck.ProbeOptions{
		Service:      app,
		Type:         healthcheck.ProbeReadiness,
		AutoMigrator: func() error { return nil },
	}))

	// Liveness health check
	app.AddEndpoint("/api/v1/locations/liveq/", healthcheck.RegisterProbe(&healthcheck.ProbeOptions{
		Service:      app,
		Type:         healthcheck.ProbeLiveNess,
		AutoMigrator: func() error { return nil },
	}))

	// Connect to external service
	dopts := []grpc.DialOption{
		grpc.WithBlock(),
	}
	cc, err := app.DialExternalService(ctx, "messaging", dopts)
	handleErr(err)

	messagingClient := messaging.NewMessagingClient(cc)

	app.Logger().Infoln("connected to messaging service")

	// Create location tracing instance
	locationAPI, err := location_app.NewLocationTracing(ctx, &location_app.Options{
		LogsDB:          app.GormDB(),
		EventsDB:        app.RedisClient(),
		MessagingClient: messagingClient,
		Logger:          app.Logger(),
		RealTimeAlerts:  os.Getenv("ENABLE_REALTIME_ALERTS") == "true",
	})
	handleErr(err)

	// Initialize grpc server
	handleErr(app.InitGRPC(ctx))

	location.RegisterLocationTracingAPIServer(app.GRPCServer(), locationAPI)
	handleErr(location.RegisterLocationTracingAPIHandlerServer(ctx, app.RuntimeMux(), locationAPI))

	handleErr(app.Run(ctx))
}

func setIfempty(val1, val2 string, swap ...bool) string {
	if len(swap) > 0 && swap[0] {
		if strings.TrimSpace(val2) == "" {
			return val1
		}
		return val2
	}
	if strings.TrimSpace(val1) == "" {
		return val2
	}
	return val1
}

func handleErr(err error) {
	if err != nil {
		logrus.Fatalln(err)
	}
}
