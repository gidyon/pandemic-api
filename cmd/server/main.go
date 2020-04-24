package main

import (
	"context"
	"strings"

	location_service "github.com/gidyon/pandemic-api/internal/services/location"

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

	registerHandlers(app)

	// Create location tracing instance
	locationAPI, err := location_service.NewLocationTracing(ctx, &location_service.Options{
		LogsDB:   app.GormDB(),
		EventsDB: app.RedisClient(),
	})
	handleErr(err)

	// Initialize grpc server
	handleErr(app.InitGRPC(ctx))

	location.RegisterLocationTracingAPIServer(app.GRPCServer(), locationAPI)

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
