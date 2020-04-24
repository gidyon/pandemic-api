package main

import (
	"context"
	"strings"

	"github.com/appleboy/go-fcm"

	messaging_app "github.com/gidyon/pandemic-api/internal/services/messaging"

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

	// FCM client
	fcmClient, err := fcm.NewClient(
		"AAAApoeNiqU:APA91bH7JMT0ITyGESfWtKzP8901ja834A_u4DP6rXw92OgujEPVJzqlL2fRyMjfU6yakaDGiGVaBBRfW-lwX7AGtBd_Ub1YZP4RMaIqCLkEZ18TD55oEReMu2ge5no1RQ5d7frrkEYW",
	)
	handleErr(err)

	// Create location tracing instance
	messagingAPI, err := messaging_app.NewMessagingServer(ctx, &messaging_app.Options{
		SQLDB:     app.GormDB(),
		FCMClient: fcmClient,
		Logger:    app.Logger(),
	})
	handleErr(err)

	// Initialize grpc server
	handleErr(app.InitGRPC(ctx))

	location.RegisterMessagingServer(app.GRPCServer(), messagingAPI)

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
