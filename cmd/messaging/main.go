package main

import (
	"context"
	"github.com/gidyon/micros/utils/healthcheck"
	"os"
	"strings"

	"github.com/appleboy/go-fcm"

	messaging_app "github.com/gidyon/pandemic-api/internal/services/messaging"

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
	app.AddEndpoint("/api/v1/messaging/readyq/", healthcheck.RegisterProbe(&healthcheck.ProbeOptions{
		Service:      app,
		Type:         healthcheck.ProbeReadiness,
		AutoMigrator: func() error { return nil },
	}))

	// Liveness health check
	app.AddEndpoint("/api/v1/messaging/liveq/", healthcheck.RegisterProbe(&healthcheck.ProbeOptions{
		Service:      app,
		Type:         healthcheck.ProbeLiveNess,
		AutoMigrator: func() error { return nil },
	}))

	// FCM client
	fcmClient, err := fcm.NewClient(os.Getenv("FCM_SERVER_KEY"))
	handleErr(err)

	// Create messaging tracing instance
	messagingAPI, err := messaging_app.NewMessagingServer(ctx, &messaging_app.Options{
		SQLDB:     app.GormDB(),
		FCMClient: fcmClient,
		Logger:    app.Logger(),
	})
	handleErr(err)

	// Initialize grpc server
	handleErr(app.InitGRPC(ctx))

	messaging.RegisterMessagingServer(app.GRPCServer(), messagingAPI)
	handleErr(messaging.RegisterMessagingHandlerServer(ctx, app.RuntimeMux(), messagingAPI))

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
