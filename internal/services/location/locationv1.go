package location

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	ctx context.Context, addReq *location.AddUserRequest,
) (*empty.Empty, error) {
	// Request must not be  nil
	if addReq == nil {
		return nil, services.NilRequestError("AddUserRequest")
	}

	// Validation
	var err error
	switch {
	case addReq.User == nil:
		err = services.MissingFieldError("user")
	case addReq.User.PhoneNumber == "":
		err = services.MissingFieldError("phone number")
	case addReq.User.FullName == "":
		err = services.MissingFieldError("full name")
	case addReq.User.DeviceToken == "":
		err = services.MissingFieldError("device token")
	case addReq.User.County == "":
		err = services.MissingFieldError("county")
	}
	if err != nil {
		return nil, err
	}

	// Reset their status to unknown
	addReq.User.Status = location.Status_UNKNOWN

	userModel, err := getUserDB(addReq.User)
	if err != nil {
		return nil, err
	}

	// If user already exists, performs an update
	alreadyExists := !locationManager.logsDB.
		First(&services.UserModel{}, "phone_number=?", addReq.User.PhoneNumber).RecordNotFound()

	if alreadyExists {
		err = locationManager.logsDB.Table(services.UsersTable).Where("phone_number=?", addReq.User.PhoneNumber).
			Omit("status", "traced").Updates(userModel).Error
		switch {
		case err == nil:
		default:
			err = status.Errorf(codes.Internal, "saving user failed: %v", err)
		}
	} else {
		// Save user
		err = locationManager.logsDB.Save(userModel).Error
		switch {
		case err == nil:
		// case strings.Contains(strings.ToLower(err.Error()), "duplicate"):
		// 	if strings.Contains(err.Error(), "phone_number") {
		// 		err = status.Error(codes.ResourceExhausted, "phone number already registred")
		// 	} else {
		// 		err = status.Errorf(codes.ResourceExhausted, "user is already registred: %v", err)
		// 	}
		default:
			err = status.Errorf(codes.Internal, "saving user failed: %v", err)
		}
	}

	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

func (locationManager *locationAPIServer) GetUser(
	ctx context.Context, getReq *location.GetUserRequest,
) (*location.User, error) {
	// Requets must not be nil
	if getReq == nil {
		return nil, services.NilRequestError("GetUserRequest")
	}

	// Validation
	var err error
	switch {
	case getReq.PhoneNumber == "":
		err = services.MissingFieldError("phone number")
	}
	if err != nil {
		return nil, err
	}

	// Get from database
	userDB := &services.UserModel{}
	err = locationManager.logsDB.First(userDB, "phone_number=?", getReq.PhoneNumber).Error
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		err = status.Errorf(codes.NotFound, "user not found: %v", err)
	default:
		err = status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if err != nil {
		return nil, err
	}

	userPB, err := getUserPB(userDB)
	if err != nil {
		return nil, err
	}

	return userPB, nil
}

func (locationManager *locationAPIServer) ListUsers(
	ctx context.Context, listReq *location.ListUsersRequest,
) (*location.ListUsersResponse, error) {
	// Request must not be nil
	if listReq == nil {
		return nil, services.NilRequestError("ListUsersRequest")
	}

	pageNumber, pageSize := normalizePage(listReq.PageToken, listReq.PageSize)
	offset := pageNumber*pageSize - pageSize

	condition := fmt.Sprintf("status=%d", listReq.FilterStatus)
	if listReq.FilterStatus == location.Status_UNKNOWN {
		condition = ""
	}

	usersDB := make([]*services.UserModel, 0)
	err := locationManager.logsDB.Offset(offset).Limit(pageSize).
		Find(&usersDB, condition).Error
	switch {
	case err == nil:
	default:
		return nil, status.Errorf(codes.Internal, "failed to get users from db: %v", err)
	}

	usersPB := make([]*location.User, 0, len(usersDB))
	for _, userDB := range usersDB {
		userPB, err := getUserPB(userDB)
		if err != nil {
			return nil, err
		}
		usersPB = append(usersPB, userPB)
	}

	return &location.ListUsersResponse{
		Users:         usersPB,
		NextPageToken: int32(pageNumber + 1),
	}, nil
}

const defaultPageSize = 10

func normalizePage(pageToken, pageSize int32) (int, int) {
	if pageToken <= 0 {
		pageToken = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > 20 {
		pageSize = 20
	}
	return int(pageToken), int(pageSize)
}

func getUserDB(userPB *location.User) (*services.UserModel, error) {
	userDB := &services.UserModel{
		PhoneNumber: userPB.PhoneNumber,
		FullName:    userPB.FullName,
		County:      userPB.County,
		Status:      int8(userPB.Status),
		DeviceToken: userPB.DeviceToken,
		Traced:      userPB.Traced,
	}
	return userDB, nil
}

func getUserPB(userDB *services.UserModel) (*location.User, error) {
	userPB := &location.User{
		PhoneNumber: userDB.PhoneNumber,
		FullName:    userDB.FullName,
		County:      userDB.County,
		Status:      location.Status(userDB.Status),
		DeviceToken: userDB.DeviceToken,
		Traced:      userDB.Traced,
	}
	return userPB, nil
}
