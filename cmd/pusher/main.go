package main

import (
	"context"
	"flag"
	"github.com/Sirupsen/logrus"
	"github.com/appleboy/go-fcm"
	"github.com/gidyon/config"
	"github.com/gidyon/micros"
	"github.com/gidyon/pandemic-api/internal/pusher"
	"os"
)

var interval = flag.Int("interval-seconds", 10, "interval to request for location")

func main() {
	ctx := context.Background()

	cfg, err := config.New()
	handleErr(err)

	service, err := micros.NewService(context.Background(), cfg, micros.NewLogger("pusher"))
	handleErr(err)

	// FCM client
	fcmClient, err := fcm.NewClient(os.Getenv("FCM_SERVER_KEY"))
	handleErr(err)

	service.Logger().Infoln("starting worker")

	// Starts pusher
	err = pusher.StartWorker(ctx, &pusher.Options{
		SQLDB:     service.GormDB(),
		FCMClient: fcmClient,
		Logger:    service.Logger(),
		Interval:  *interval,
	})
	handleErr(err)
}

func handleErr(err error) {
	if err != nil {
		logrus.Fatalln(err)
	}
}
