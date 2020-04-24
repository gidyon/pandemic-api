package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	services "github.com/gidyon/pandemic-api/internal/services"

	"github.com/gidyon/pandemic-api/pkg/api/location"

	"github.com/go-redis/redis"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jinzhu/gorm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const timeBoundary = time.Duration(5 * time.Minute)

type locationAPIServer struct {
	logsDB   *gorm.DB
	eventsDB *redis.Client
}

// Options contains parameters for NewLocationTracing
type Options struct {
	LogsDB   *gorm.DB
	EventsDB *redis.Client
}

// NewLocationTracing creates a new location tracing API
func NewLocationTracing(ctx context.Context, opt *Options) (location.LocationTracingAPIServer, error) {
	var err error
	// Validation
	switch {
	case ctx == nil:
		err = errors.New("non-nil context must not be nil")
	case opt.LogsDB == nil:
		err = errors.New("non-nil logsDB is required")
	case opt.EventsDB == nil:
		err = errors.New("non-nil eventsDB is required")
	}
	if err != nil {
		return nil, err
	}

	locationManager := &locationAPIServer{
		logsDB:   opt.LogsDB,
		eventsDB: opt.EventsDB,
	}

	// Automigration
	err = locationManager.logsDB.AutoMigrate(&services.LocationModel{}, &services.UserModel{}).Error
	if err != nil {
		return nil, err
	}

	return locationManager, nil
}

const (
	sendLocatiosSet = "sendlocations"
	updatesStream   = "updates:stream"
	updatesList     = "updates:list"
)

func validateLocation(locationPB *location.Location) error {
	// Validation
	var err error
	switch {
	case locationPB == nil:
		err = services.MissingFieldError("location")
	case locationPB.GetLongitude() == 0.0:
		err = services.MissingFieldError("location longitude")
	case locationPB.GetLatitude() == 0.0:
		err = services.MissingFieldError("location latitude")
	case locationPB.GetGeoFenceId() == "":
		err = services.MissingFieldError("location geo fence id")
	case locationPB.Timestamp == 0.0:
		err = services.MissingFieldError("location timestamp")
	case locationPB.TimeId == "":
		err = services.MissingFieldError("location time id")
	}
	if err != nil {
		return err
	}
	return nil
}

func getUserSetKeyToday(userID string) string {
	y, m, d := time.Now().Date()
	date := fmt.Sprintf("%d:%d:%d", y, m, d)
	return fmt.Sprintf("%s:%s", userID, date)
}

func (locationManager *locationAPIServer) validateAndSaveLocation(sendReq *location.SendLocationRequest) error {
	locationPB := sendReq.GetLocation()
	err := validateLocation(locationPB)
	if err != nil {
		return err
	}

	// Validate user id and status
	switch {
	case sendReq.UserId == "":
		return services.MissingFieldError("user id")
	case sendReq.StatusId.String() == "":
		return services.MissingFieldError("status id")
	}

	key := getUserSetKeyToday(sendReq.UserId)

	// Add to set
	_, err = locationManager.eventsDB.SAdd(
		key, fmt.Sprintf("%s:%s", locationPB.GetGeoFenceId(), locationPB.GetTimeId()),
	).Result()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to add location to set: %v", err)
	}

	locationDB := services.GetLocationDB(locationPB)

	// Add to database
	err = locationManager.logsDB.Create(locationDB).Error
	if err != nil {
		return status.Errorf(codes.Internal, "failed to add location to db: %v", err)
	}

	return nil
}

func (locationManager *locationAPIServer) SendLocation(
	ctx context.Context, sendReq *location.SendLocationRequest,
) (*empty.Empty, error) {
	// Request must not be nil
	if sendReq == nil {
		return nil, services.NilRequestError("SendLocationsRequest")
	}

	err := locationManager.validateAndSaveLocation(sendReq)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

func (locationManager *locationAPIServer) SendLocations(
	sendStream location.LocationTracingAPI_SendLocationsServer,
) error {
	// Request should not be nil
	if sendStream == nil {
		return services.NilRequestError("LocationTracingAPI_SendLocationsServer")
	}

streamLoop:
	for {
		sendReq, err := sendStream.Recv()
		switch {
		case err == nil:
		case errors.Is(err, io.EOF):
			break streamLoop
		default:
			return status.Errorf(codes.Unknown, "failed to receive from stream: %v", err)
		}

		// Request must not be nil
		if sendReq == nil {
			return services.NilRequestError("SendLocationsRequest")
		}

		// Validate and save location
		err = locationManager.validateAndSaveLocation(sendReq)
		if err != nil {
			return err
		}
	}

	return sendStream.SendAndClose(&empty.Empty{})
}

const infectedUsers = "infected:users"

func (locationManager *locationAPIServer) UpdateUserStatus(
	ctx context.Context, updatReq *location.UpdateUserStatusRequest,
) (*empty.Empty, error) {
	// Request must not be nil
	if updatReq == nil {
		return nil, services.NilRequestError("UpdateUserStatusRequest")
	}

	// Validation
	var err error
	switch {
	case updatReq.PhoneNumber == "":
		err = services.MissingFieldError("user phone")
	case updatReq.Status.String() == "":
		err = services.MissingFieldError("status id")
	}
	if err != nil {
		return nil, err
	}

	tx := locationManager.logsDB.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
		}
	}()
	if tx.Error != nil {
		return nil, services.FailedToBeginTx(err)
	}

	// Update status in database
	err = tx.Table(services.UsersTable).Where("phone_number=?", updatReq.PhoneNumber).
		Update("status", int8(updatReq.Status)).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user status: %v", err)
	}

	// Add user to list of users with COVID-19
	if updatReq.GetStatus() == location.Status_POSITIVE {
		_, err = locationManager.eventsDB.LPush(infectedUsers, updatReq.PhoneNumber).Result()
		if err != nil {
			tx.Rollback()
			return nil, status.Errorf(codes.Internal, "failed to add users to list of infected users: %v", err)
		}
	}

	err = tx.Commit().Error
	if err != nil {
		return nil, services.FailedToCommitTx(err)
	}

	return &empty.Empty{}, nil
}

func (locationManager *locationAPIServer) AddUser(
	ctx context.Context, sendReq *location.AddUserRequest,
) (*empty.Empty, error) {
	// Request must not be  nil
	if sendReq == nil {
		return nil, services.NilRequestError("AddUserRequest")
	}

	// Validation
	var err error
	switch {
	case sendReq.PhoneNumber == "":
		err = services.MissingFieldError("phone number")
	case sendReq.FullName == "":
		err = services.MissingFieldError("full name")
	case sendReq.DeviceToken == "":
		err = services.MissingFieldError("device token")
	case sendReq.County == "":
		err = services.MissingFieldError("county")
	}
	if err != nil {
		return nil, err
	}

	// Reset their status to unknown
	sendReq.StatusId = location.Status_UNKNOWN

	// Save user
	err = locationManager.logsDB.Create(&services.UserModel{
		PhoneNumber: sendReq.PhoneNumber,
		FullName:    sendReq.FullName,
		Status:      int8(sendReq.StatusId),
		DeviceToken: sendReq.DeviceToken,
	}).Error
	switch {
	case strings.Contains(strings.ToLower(err.Error()), "duplicate"):
		if strings.Contains(err.Error(), "phone_number") {
			err = status.Error(codes.ResourceExhausted, "phone number already registred")
		} else {
			err = status.Errorf(codes.ResourceExhausted, "user is already registred: %v", err)
		}
	default:
		err = status.Errorf(codes.Internal, "saving user failed: %v", err)
	}
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}
