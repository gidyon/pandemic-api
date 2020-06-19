package main

import (
	"context"
	"encoding/json"
	"github.com/gidyon/micros/utils/healthcheck"
	location_app "github.com/gidyon/pandemic-api/internal/services/location"
	http_error "github.com/gidyon/pandemic-api/pkg/errors"
	"google.golang.org/grpc"
	"net/http"
	"os"
	"strings"

	"github.com/gidyon/pandemic-api/pkg/api/location"
	"github.com/gidyon/pandemic-api/pkg/api/messaging"

	"github.com/gidyon/micros"
	"github.com/gidyon/micros/pkg/config"

	"github.com/Sirupsen/logrus"
)

func main() {
	cfg, err := config.New()
	handleErr(err)

	ctx := context.Background()

	app, err := micros.NewService(ctx, cfg, nil)
	handleErr(err)

	// Readiness health check
	app.AddEndpoint("/api/v1/locations/health/ready", healthcheck.RegisterProbe(&healthcheck.ProbeOptions{
		Service:      app,
		Type:         healthcheck.ProbeReadiness,
		AutoMigrator: func() error { return nil },
	}))

	// Liveness health check
	app.AddEndpoint("/api/v1/locations/health/live", healthcheck.RegisterProbe(&healthcheck.ProbeOptions{
		Service:      app,
		Type:         healthcheck.ProbeLiveNess,
		AutoMigrator: func() error { return nil },
	}))

	// Token endpoint
	app.AddEndpointFunc("/api/v1/users/token/", func(w http.ResponseWriter, r *http.Request) {
		phone := r.URL.Query().Get("phone_number")
		deviceID := r.URL.Query().Get("device_token")

		if phone == "" || deviceID == "" {
			http_error.Write(w, &http_error.Error{
				Message: "missing phone or device id",
				Details: "missing phone or device id",
				Code:    http.StatusBadRequest,
			})
			return
		}

		token, err := getToken(app.GormDB(), phone, deviceID)
		if err != nil {
			http_error.Write(w, &http_error.Error{
				Message: "failed to get token",
				Details: err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	})

	app.Start(ctx, func() error {
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

		location.RegisterLocationTracingAPIServer(app.GRPCServer(), locationAPI)
		handleErr(location.RegisterLocationTracingAPIHandlerServer(ctx, app.RuntimeMux(), locationAPI))

		return nil
	})
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
