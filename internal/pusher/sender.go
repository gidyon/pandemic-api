package pusher

// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"github.com/appleboy/go-fcm"
// 	"github.com/gidyon/pandemic-api/internal/services"
// 	"github.com/go-redis/redis"
// 	"github.com/jinzhu/gorm"
// 	"google.golang.org/grpc/grpclog"
// 	"time"
// )

// type deviceManager struct {
// 	sqlDB     *gorm.DB
// 	redisDB   *redis.Client
// 	fcmClient *fcm.Client
// 	logger    grpclog.LoggerV2
// 	timeout   time.Duration
// }

// // Options contains options that is passed to StartWorker
// type Options struct {
// 	SQLDB     *gorm.DB
// 	RedisDB   *redis.Client
// 	FCMClient *fcm.Client
// 	Logger    grpclog.LoggerV2
// 	Interval  int
// }

// const (
// 	devicesList1     = "devices:registrations:1"
// 	devicesList2     = "devices:registrations:2"
// 	initializedSet   = "devices:set"
// 	senderWorkerChan = "device:workers:channel"
// 	maxDevices       = 1000
// 	doneChannel      = "workers:done"
// )

// // StartWorker starts service that sends notification to millions of devices
// func StartWorker(ctx context.Context, opt *Options) error {
// 	// Validation
// 	var err error
// 	switch {
// 	case opt.FCMClient == nil:
// 		err = errors.New("fcm client must not be nil")
// 	case opt.Logger == nil:
// 		err = errors.New("logger cannot be nil")
// 	case opt.RedisDB == nil:
// 		err = errors.New("redisdb cannot be nil")
// 	case opt.SQLDB == nil:
// 		err = errors.New("sqlDB cannot be nil")
// 	}
// 	if err != nil {
// 		return err
// 	}

// 	if opt.Interval == 0 {
// 		opt.Interval = 3
// 	}

// 	dm := &deviceManager{
// 		sqlDB:     opt.SQLDB,
// 		redisDB:   opt.RedisDB,
// 		fcmClient: opt.FCMClient,
// 		logger:    opt.Logger,
// 		timeout:   time.Duration(opt.Interval) * time.Second,
// 	}

// 	// Ato migrate
// 	err = dm.sqlDB.AutoMigrate(&services.UserModel{}).Error
// 	if err != nil {
// 		return fmt.Errorf("failed to automigrate: %v", err)
// 	}

// 	res, err := dm.redisDB.SAdd(initializedSet, "FIRST").Result()
// 	switch {
// 	case err == nil:
// 	case errors.Is(err, redis.Nil):
// 	default:
// 		return fmt.Errorf("failed to check if devices are loaded: %v", err)
// 	}

// 	iStarted := make(chan struct{}, 0)

// 	if res > 0 {
// 		// Load devices to list
// 		var devices []string
// 		condition := true
// 		limit := 1000
// 		offset := 0
// 		deviceToken := ""

// 		close(iStarted)

// 		for condition {
// 			devices = make([]string, 0, limit)
// 			rows, err := dm.sqlDB.Table(services.UsersTable).Limit(limit).Offset(offset).Select("device_token").Rows()
// 			if err != nil {
// 				return fmt.Errorf("failed to get rows: %v", err)
// 			}

// 			for rows.Next() {
// 				err = rows.Scan(&deviceToken)
// 				if err != nil {
// 					return fmt.Errorf("failed to scan row: %v", err)
// 				}
// 				devices = append(devices, deviceToken)
// 			}

// 			offset += len(devices)
// 			if len(devices) < limit {
// 				condition = false
// 			}
// 			// Add devices to redis
// 			err = dm.redisDB.LPush(devicesList1, devices).Err()
// 			if err != nil {
// 				return fmt.Errorf("failed to add devices to device list: %v", err)
// 			}
// 		}

// 		err = dm.redisDB.Publish(senderWorkerChan, "START").Err()
// 		if err != nil {
// 			return fmt.Errorf("failed to add devices to device list: %v", err)
// 		}

// 		// Be notifying other workers
// 		go func(ctx context.Context) {
// 			var dones int
// 			for {
// 				// Naive dustributed interval
// 				select {
// 				case <-dm.redisDB.Subscribe(doneChannel):
// 					done++
// 				case <-ctx.Done():
// 					return
// 				case <-time.After(dm.timeout):
// 					for i := 0; i < 5; i++ {
// 						err = dm.redisDB.Publish(senderWorkerChan, "CONTINUE").Err()
// 						if err != nil {
// 							dm.logger.Errorf("failed to publish CONTINUE: %v\n", err)
// 							continue
// 						}
// 						break
// 					}
// 					dm.logger.Infoln("published sending")
// 				}
// 			}
// 		}(ctx)
// 	}

// 	// Only when told to start should other worker start
// 	startChan := dm.redisDB.Subscribe(senderWorkerChan).ChannelSize(100)
// 	select {
// 	case <-iStarted:
// 		dm.startWorker(ctx, startChan)
// 	case <-startChan:
// 		dm.startWorker(ctx, startChan)
// 	case <-time.After(5 * time.Minute):
// 		return errors.New("waited too long for producer to send signal")
// 	}

// 	return nil
// }

// func (dm *deviceManager) startWorker(ctx context.Context, resumeChan <-chan *redis.Message) {
// 	var (
// 		popList  = devicesList1
// 		pushList = devicesList2
// 		devices  = make([]string, 0)
// 	)

// 	// Get all device tokens
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case <-resumeChan:
// 			dm.logger.Infoln("received request to send messages to devices")

// 			// Devices list
// 			devices = make([]string, 0, 1000)

// 		loopB:
// 			for {
// 				// Get device token
// 				device, err := dm.redisDB.BRPopLPush(popList, pushList, 0).Result()
// 				switch {
// 				case err == nil:
// 				case errors.Is(err, redis.Nil) || device == "":
// 					// Check if the list is empty
// 					n, _ := dm.redisDB.LLen(popList).Result()
// 					if n == 0 {
// 						list := popList
// 						popList = pushList
// 						pushList = list

// 						if len(devices) > 0 {
// 							dm.redisDB.Publish(doneChannel, "DONE")
// 							dm.logger.Infof("sending messages to %d devices", len(devices))
// 							// Send message
// 							_, err = dm.fcmClient.Send(&fcm.Message{
// 								RegistrationIDs: devices,
// 								Data: map[string]interface{}{
// 									"type": "UPDATE",
// 									"time": time.Now().String(),
// 								},
// 								Priority: "high",
// 							})
// 							if err != nil {
// 								dm.logger.Errorf("failed to send message to device: %v", err)
// 							}
// 						}

// 						break loopB
// 					}

// 					continue
// 				default:
// 					dm.logger.Errorf("failed to brpoplpush from list: %v", err)
// 					continue
// 				}

// 				devices = append(devices, device)

// 				if len(devices) >= maxDevices {
// 					dm.logger.Infof("sending messages to %d devices", len(devices))
// 					_, err = dm.fcmClient.Send(&fcm.Message{
// 						RegistrationIDs: devices,
// 						Data: map[string]interface{}{
// 							"type": "UPDATE",
// 							"time": time.Now().String(),
// 						},
// 						Priority: "high",
// 					})
// 					if err != nil {
// 						dm.logger.Errorf("failed to send message to device: %v", err)
// 					}
// 				}
// 			}
// 		}
// 	}
// }
