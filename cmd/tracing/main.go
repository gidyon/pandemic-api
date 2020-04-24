package main

import (
	"context"
	"strings"

	"google.golang.org/grpc"

	tracing_service "github.com/gidyon/pandemic-api/internal/services/tracing"

	"github.com/gidyon/pandemic-api/pkg/api/location"

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

	dopts := []grpc.DialOption{
		grpc.WithBlock(),
	}
	// Connect to external service
	cc, err := app.DialExternalService(ctx, "messaging", dopts)
	handleErr(err)

	messagingClient := location.NewMessagingClient(cc)

	app.Logger().Infoln("connected to messaging service")

	// Create location tracing instance
	tracingAPI, err := tracing_service.NewContactTracingAPI(ctx, &tracing_service.Options{
		SQLDB:           app.GormDB(),
		RedisClient:     app.RedisClient(),
		MessagingClient: messagingClient,
		Logger:          app.Logger(),
	})
	handleErr(err)

	// Initialize grpc server
	handleErr(app.InitGRPC(ctx))

	location.RegisterContactTracingServer(app.GRPCServer(), tracingAPI)

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
