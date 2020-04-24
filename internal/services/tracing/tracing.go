package tracing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"

	"google.golang.org/grpc/grpclog"

	"github.com/google/uuid"

	"google.golang.org/genproto/googleapis/longrunning"

	"google.golang.org/grpc"

	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"

	"github.com/gidyon/pandemic-api/internal/services"
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"github.com/go-redis/redis"

	"github.com/jinzhu/gorm"

	spb "google.golang.org/genproto/googleapis/rpc/status"
)

type tracingAPIServer struct {
	failedDayContactChan chan *dayContact
	contactDataChan      chan *location.ContactData
	logger               grpclog.LoggerV2
	sqlDB                *gorm.DB
	redisDB              *redis.Client
	messagingClient      location.MessagingClient
}

// Options contains options for creating tracing API
type Options struct {
	SQLDB           *gorm.DB
	RedisClient     *redis.Client
	MessagingClient location.MessagingClient
	Logger          grpclog.LoggerV2
}

// NewContactTracingAPI creates a new contact tracing API server
func NewContactTracingAPI(
	ctx context.Context, opt *Options,
) (location.ContactTracingServer, error) {
	// Validation
	var err error
	switch {
	case opt.SQLDB == nil:
		err = errors.New("non-nil sqlDB is required")
	case opt.RedisClient == nil:
		err = errors.New("non-nil redis is required")
	case opt.MessagingClient == nil:
		err = errors.New("non-nil messaging client is required")
	case opt.Logger == nil:
		err = errors.New("non-nil logger is required")
	}
	if err != nil {
		return nil, err
	}

	ms := &tracingAPIServer{
		failedDayContactChan: make(chan *dayContact, 0),
		contactDataChan:      make(chan *location.ContactData, 0),
		sqlDB:                opt.SQLDB,
		redisDB:              opt.RedisClient,
		messagingClient:      opt.MessagingClient,
		logger:               opt.Logger,
	}

	return ms, nil
}

func getUserSetKey(userID string, t *time.Time) string {
	y, m, d := t.Date()
	date := fmt.Sprintf("%d:%d:%d", y, m, d)
	return fmt.Sprintf("%s:%s", userID, date)
}

type dayContact struct {
	UserPhone    string
	PatientPhone string
	Date         *time.Time
}

const runningOps = "longrunning:operations"

func (t *tracingAPIServer) TraceUserLocations(
	ctx context.Context, traceReq *location.TraceUserLocationsRequest,
) (*longrunning.Operation, error) {
	// Request must not be nil
	if traceReq == nil {
		return nil, services.NilRequestError("TraceUserLocationsRequest")
	}

	// Validation
	var err error
	switch {
	case traceReq.PhoneNumber == "":
		err = services.MissingFieldError("phone number")
	case traceReq.SinceDate == "":
		err = services.MissingFieldError("since date")
	}
	if err != nil {
		return nil, err
	}

	// Get user from db
	userDB := &services.UserModel{}
	err = t.sqlDB.Select("status").First(userDB, "phone_number=?", traceReq.PhoneNumber).Error
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, status.Errorf(codes.NotFound, "user with phone %s not found", traceReq.PhoneNumber)
	default:
		return nil, status.Errorf(codes.NotFound, "error happened: %v", err)
	}

	// User status must be positive
	if userDB.Status != int8(location.Status_POSITIVE) {
		return nil, status.Error(codes.FailedPrecondition, "user status must be positive")
	}

	// Get user logs for last n days
	sinceDate, err := time.Parse("2006-01-02", traceReq.SinceDate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse since date: %v", err)
	}
	todayDate := time.Now()

	if todayDate.Sub(sinceDate) <= 0 {
		return nil, status.Error(codes.FailedPrecondition, "since date cannot be greater than today")
	}

	// Create a long running operation
	longrunningOp := &longrunning.Operation{
		Name: fmt.Sprintf("TraceUserLocations::%s", uuid.New().String()),
		Done: false,
	}

	// Marshal to bytes
	bs, err := proto.Marshal(longrunningOp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal longrunning operation: %v", err)
	}

	// Save long running to cache
	err = t.redisDB.Set(longrunningOp.Name, bs, 12*time.Hour).Err()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save long running operation to cache: %v", err)
	}

	// Longrunning worker
	go t.traceUserWorker(longrunningOp.Name, userDB, &sinceDate)

	return longrunningOp, nil
}

const limit = 1000

func (t *tracingAPIServer) traceUserWorker(
	longrunningID string, userDB *services.UserModel, sinceDate *time.Time,
) {
	ctx := context.Background()

	// Get users with limit
	todayDate := time.Now()
	condition := true
	offset := 0

	// Set client ready
	client, err := t.messagingClient.AlertContacts(ctx, grpc.WaitForReady(true))
	if err != nil {
		errMsg := fmt.Sprintf("failed to create stream for sending messages: %v", err)
		t.logger.Errorf(errMsg)
		t.failLongRunningOperation(longrunningID, errMsg)
		return
	}

	usersDB := make([]*services.UserModel, 0, limit)
	for condition {
		err = t.sqlDB.Limit(limit).Offset(offset).Find(&usersDB).Error
		if err != nil {
			errMsg := fmt.Sprintf("failed to get users to send messages: %v", err)
			t.logger.Errorf(errMsg)
			t.failLongRunningOperation(longrunningID, errMsg)
			return
		}

		if len(usersDB) < limit {
			condition = false
		}

		wg := &sync.WaitGroup{}

		// For each user
		for _, suspect := range usersDB {
			wg.Add(1)

			suspect := suspect

			go func() {
				defer wg.Done()

				contactData := &location.ContactData{
					Count:         0,
					PatientPhone:  userDB.PhoneNumber,
					UserPhone:     suspect.PhoneNumber,
					DeviceToken:   userDB.DeviceToken,
					ContactPoints: make([]*location.ContactPoint, 0),
				}

				pipeliner := t.redisDB.Pipeline()

				for sinceDate.Unix() <= todayDate.Unix() {
					// Get union of contact points
					contacts, err := pipeliner.SUnion(
						getUserSetKey(suspect.PhoneNumber, sinceDate), getUserSetKey(userDB.PhoneNumber, sinceDate),
					).Result()
					*sinceDate = sinceDate.Add(time.Hour * 24)

					if err != nil {
						t.logger.Error("failed to get contact points: %v", err)
						// Send failed user id
						select {
						case <-time.After(5 * time.Second):
						case t.failedDayContactChan <- &dayContact{
							UserPhone:    suspect.PhoneNumber,
							PatientPhone: userDB.PhoneNumber,
							Date:         sinceDate,
						}:
						}
						continue
					}

					// Range individual contact points
					for _, contact := range contacts {
						contactPointData := strings.Split(contact, ":")
						if len(contactPointData) != 2 {
							t.logger.Warning("empty contact point data")
							continue
						}

						contactData.Count++
						contactData.ContactPoints = append(contactData.ContactPoints, &location.ContactPoint{
							GeoFenceId: contactPointData[0],
							TimeId:     contactPointData[1],
						})
					}
				}

				_, err := pipeliner.Exec()
				if err != nil {
					t.logger.Error("error while creating pipeline: %v", err)
					t.sendContactData(contactData)
					return
				}

				// Send contact data to messaging server
				err = client.Send(contactData)
				if err != nil {
					t.logger.Error("erro while sending to stream: %v", err)
					t.sendContactData(contactData)
					return
				}
			}()
		}

		wg.Wait()

		offset += limit
	}
}

func (t *tracingAPIServer) sendContactData(contactData *location.ContactData) {
	select {
	case <-time.After(5 * time.Second):
	case t.contactDataChan <- contactData:
	}
}

func (t *tracingAPIServer) failLongRunningOperation(longrunningID string, errMsg string) {
	// Get the longrunning operation
	longrunningOpStr, err := t.redisDB.Get(longrunningID).Result()
	if err != nil {
		t.logger.Errorf("failed to get longrunning operation from cache: %v", err)
		return
	}

	// Unmarshal to proto message
	longrunningOp := &longrunning.Operation{}
	err = proto.Unmarshal([]byte(longrunningOpStr), longrunningOp)
	if err != nil {
		t.logger.Errorf("failed to unmarshal longrunning operation: %v", err)
		return
	}

	// Update proto message
	longrunningOp.Done = true
	longrunningOp.Result = &longrunning.Operation_Error{
		Error: &spb.Status{
			Code:    int32(codes.Internal),
			Message: fmt.Sprintf("creating messages stream failed: %s", errMsg),
		},
	}

	// Marshal to bytes
	bs, err := proto.Marshal(longrunningOp)
	if err != nil {
		t.logger.Errorf("failed to marshal longrunning operation: %v", err)
		return
	}

	// Save back to cache
	err = t.redisDB.Set(longrunningID, bs, 12*time.Hour).Err()
	if err != nil {
		t.logger.Errorf("failed to set longrunning operation: %v", err)
		return
	}
}
