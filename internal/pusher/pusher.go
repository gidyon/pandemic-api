package pusher

import (
	"context"
	"errors"
	"fmt"
	"github.com/appleboy/go-fcm"
	"github.com/gidyon/pandemic-api/internal/services"
	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	"google.golang.org/grpc/grpclog"
	"sync"
	"time"
)

type deviceManager struct {
	sqlDB     *gorm.DB
	redisDB   *redis.Client
	fcmClient *fcm.Client
	logger    grpclog.LoggerV2
	timeout   time.Duration
	devices   map[int][]string
}

// Options contains options that is passed to StartWorker
type Options struct {
	SQLDB     *gorm.DB
	FCMClient *fcm.Client
	Logger    grpclog.LoggerV2
	Interval  int
}

const (
	devicesList1     = "devices:registrations:1"
	devicesList2     = "devices:registrations:2"
	initializedSet   = "devices:set"
	senderWorkerChan = "device:workers:channel"
	maxDevices       = 1000
	doneChannel      = "workers:done"
)

// StartWorker starts service that sends notification to millions of devices
func StartWorker(ctx context.Context, opt *Options) error {
	// Validation
	var err error
	switch {
	case opt.FCMClient == nil:
		err = errors.New("fcm client must not be nil")
	case opt.Logger == nil:
		err = errors.New("logger cannot be nil")
	case opt.SQLDB == nil:
		err = errors.New("sqlDB cannot be nil")
	}
	if err != nil {
		return err
	}

	if opt.Interval == 0 {
		opt.Interval = 3
	}

	dm := &deviceManager{
		sqlDB:     opt.SQLDB,
		fcmClient: opt.FCMClient,
		logger:    opt.Logger,
		timeout:   time.Duration(opt.Interval) * time.Second,
		devices:   make(map[int][]string, 0),
	}

	// Ato migrate
	err = dm.sqlDB.AutoMigrate(&services.UserModel{}).Error
	if err != nil {
		return fmt.Errorf("failed to automigrate: %v", err)
	}

	// Load devices to list
	var devices []string
	condition := true
	index := 1
	limit := 1000
	offset := 0
	deviceToken := ""

	for condition {
		devices = make([]string, 0, limit)
		rows, err := dm.sqlDB.Table(services.UsersTable).Limit(limit).Offset(offset).Select("device_token").
			Where("phone_number=?", "+254716484395").Rows()
		if err != nil {
			return fmt.Errorf("failed to get rows: %v", err)
		}

		for rows.Next() {
			err = rows.Scan(&deviceToken)
			if err != nil {
				return fmt.Errorf("failed to scan row: %v", err)
			}
			devices = append(devices, deviceToken)
		}

		offset += len(devices)
		if len(devices) < limit {
			condition = false
		}

		dm.devices[index] = devices
	}

	return dm.startWorker(ctx)
}

func (dm *deviceManager) startWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(dm.timeout):
			dm.logger.Infoln("received request to send messages to devices")

			// Send to devices
			wg := &sync.WaitGroup{}

			for _, devices := range dm.devices {
				wg.Add(1)

				dm.logger.Infoln(len(devices))

				go func(devices []string) {
					defer wg.Done()

					dm.logger.Infof("devices: %d", len(devices))
					dm.logger.Infof("device 0: %s", devices[0])

					_, err := dm.fcmClient.Send(&fcm.Message{
						RegistrationIDs: devices,
						Data: map[string]interface{}{
							"type": "UPDATE",

							"time": time.Now().String(),
						},
						// Notification: &fcm.Notification{
						// 	Title: "Hello User",
						// 	Body:  "Don't forget to take precaution",
						// },
						Priority: "high",
					})
					if err != nil {
						dm.logger.Errorf("failed to send message to device: %v", err)
						return
					}

					dm.logger.Infoln("semding was susccessful")
				}(devices)
			}

			wg.Wait()
		}
	}
}
