package location

import (
	"context"
	"errors"
	"fmt"
	"github.com/gidyon/pandemic-api/internal/auth"
	"github.com/gidyon/pandemic-api/internal/services/location/conversion"
	"github.com/gidyon/pandemic-api/pkg/api/messaging"
	hashids "github.com/speps/go-hashids"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
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

const (
	timeBoundary   = time.Duration(5 * time.Minute)
	alertRadius    = 10  // 10 meters
	socialDistance = 1.5 //meters
	durationRange  = 5 * time.Minute
)

type locationAPIServer struct {
	logsDB          *gorm.DB
	eventsDB        *redis.Client
	logger          grpclog.LoggerV2
	hasher          *hashids.HashID
	messagingClient messaging.MessagingClient
	realtimeAlerts  bool
	authorize       func(context.Context, string) error
	authenticate    func(context.Context) error
}

// Options contains parameters for NewLocationTracing
type Options struct {
	LogsDB          *gorm.DB
	EventsDB        *redis.Client
	Logger          grpclog.LoggerV2
	MessagingClient messaging.MessagingClient
	RealTimeAlerts  bool
}

func newHasher(salt string) (*hashids.HashID, error) {
	return nil, nil
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
	case opt.Logger == nil:
		err = errors.New("non-nil grpc logger is required")
	case opt.MessagingClient == nil:
		err = errors.New("non-nil messaging client is required")
	}
	if err != nil {
		return nil, err
	}

	lapi := &locationAPIServer{
		logsDB:          opt.LogsDB,
		eventsDB:        opt.EventsDB,
		logger:          opt.Logger,
		messagingClient: opt.MessagingClient,
		realtimeAlerts:  opt.RealTimeAlerts,
		authenticate:    auth.AuthenticateRequest,
		authorize:       auth.AuthenticateUser,
	}

	// Automigration
	err = lapi.logsDB.AutoMigrate(&services.LocationModel{}, &services.UserModel{}).Error
	if err != nil {
		return nil, err
	}

	// Create a full text search index
	err = services.CreateFullTextIndex(lapi.logsDB, services.UsersTable, "phone_number, full_name")
	if err != nil {
		return nil, fmt.Errorf("failed to create full text index: %v", err)
	}

	return lapi, nil
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
	case locationPB.Timestamp == 0.0:
		err = services.MissingFieldError("location timestamp")
	case locationPB.GeoFenceId == "":
		err = services.MissingFieldError("location geo fence id")
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

func (lapi *locationAPIServer) validateAndSaveLocation(
	ctx context.Context, sendReq *location.SendLocationRequest, notifyUser bool,
) error {
	locationPB := sendReq.GetLocation()
	err := validateLocation(locationPB)
	if err != nil {
		return err
	}

	// Update location on server
	locationPB.TimeId = conversion.GetTimeID(locationPB, durationRange)

	// Validate user id and status
	switch {
	case sendReq.UserId == "":
		return services.MissingFieldError("user id")
	case sendReq.StatusId.String() == "":
		return services.MissingFieldError("status id")
	}

	// save user log to redis
	key := getUserSetKeyToday(sendReq.UserId)

	// Add to set
	_, err = lapi.eventsDB.SAdd(
		ctx, key, fmt.Sprintf("%s:%s", locationPB.GetGeoFenceId(), locationPB.GetTimeId()),
	).Result()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to add location to set: %v", err)
	}

	if sendReq.StatusId == location.Status_POSITIVE {
		// Add to blacklist
		err = lapi.eventsDB.SAdd(
			ctx, getTimeKey(locationPB.GetTimeId()), getAlertGeoFenceID(locationPB),
		).Err()
		if err != nil {
			return status.Errorf(codes.Internal, "failed to add location to set: %v", err)
		}
	} else {
		if notifyUser {
			go lapi.sendUserAlert(locationPB, sendReq.UserId)
		}
	}

	// Save to database
	locationDB := services.GetLocationDB(locationPB)

	locationDB.UserID = sendReq.UserId

	// Add to database
	err = lapi.logsDB.Create(locationDB).Error
	if err != nil {
		return status.Errorf(codes.Internal, "failed to add location to db: %v", err)
	}

	return nil
}

func getTimeKey(timeID string) string {
	return fmt.Sprintf("time:%s", timeID)
}

func getAlertGeoFenceID(loc *location.Location) string {
	return conversion.GeoFenceID(loc, alertRadius)
}

func (lapi *locationAPIServer) sendUserAlert(loc *location.Location, phoneNumber string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	danger, err := lapi.eventsDB.SIsMember(
		ctx, getTimeKey(loc.GetTimeId()), getAlertGeoFenceID(loc),
	).Result()
	if err != nil {
		lapi.logger.Errorf("failed to get cases: %v", err)
		return
	}

	if danger {
		// Send message to user
		_, err = lapi.messagingClient.SendMessage(ctx, &messaging.Message{
			UserPhone:    phoneNumber,
			Title:        "COVID-19 Social Distancing Alert",
			Notification: "You are currently in an area with a COVID-19 cases. Ensure you maintain social distance",
			Timestamp:    time.Now().Unix(),
			Type:         messaging.MessageType_ALERT,
		}, grpc.WaitForReady(true))
		if err != nil {
			lapi.logger.Errorf("failed to send user message on aerial covid-19 case: %v", err)
			return
		}
	}
}

func (lapi *locationAPIServer) SendLocation(
	ctx context.Context, sendReq *location.SendLocationRequest,
) (*empty.Empty, error) {
	// Request must not be nil
	if sendReq == nil {
		return nil, services.NilRequestError("SendLocationsRequest")
	}

	// Authenticate request
	err := lapi.authenticate(ctx)
	if err != nil {
		return nil, err
	}

	err = lapi.validateAndSaveLocation(ctx, sendReq, lapi.realtimeAlerts)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

func (lapi *locationAPIServer) SendLocations(
	ctx context.Context, sendReq *location.SendLocationsRequest,
) (*empty.Empty, error) {
	// Request must not be nil
	if sendReq == nil {
		return nil, services.NilRequestError("SendLocationsRequest")
	}

	// Authenticate request
	err := lapi.authenticate(ctx)
	if err != nil {
		return nil, err
	}

	// Validation
	switch {
	case sendReq.GetStatusId().String() == "":
		err = services.MissingFieldError("status")
	case sendReq.GetUserId() == "":
		err = services.MissingFieldError("user id")
	}
	if err != nil {
		return nil, err
	}

	var (
		shouldNotify    bool
		approximateTime string
		placeMark       string
		maxTimestamp    int64
	)
	// Validate and save locations
	for _, locationPB := range sendReq.Locations {
		err = lapi.validateAndSaveLocation(ctx, &location.SendLocationRequest{
			UserId:   sendReq.UserId,
			StatusId: sendReq.StatusId,
			Location: locationPB,
		}, false)
		if err != nil {
			lapi.logger.Errorf("error while saving user locations: %v", err)
			continue
		}

		// Check if there exist case
		danger, err := lapi.eventsDB.SIsMember(
			ctx, getTimeKey(locationPB.GetTimeId()), getAlertGeoFenceID(locationPB),
		).Result()
		if err != nil {
			lapi.logger.Errorf("error checking for regional alerts: %v", err)
			continue
		}
		if danger {
			if locationPB.Timestamp > maxTimestamp {
				maxTimestamp = locationPB.Timestamp
				placeMark = locationPB.Placemark
				appTime := time.Unix(locationPB.Timestamp, 0)
				approximateTime = appTime.Format(time.RFC1123)
			}
			shouldNotify = true
		}
	}

	if shouldNotify && lapi.realtimeAlerts {

		// Send user a notification
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_, err := lapi.messagingClient.SendMessage(ctx, &messaging.Message{
				UserPhone: sendReq.UserId,
				Title:     "COVID-19 Social Distancing Warning",
				Notification: fmt.Sprintf(
					"You were in an area (10m diameter) with confirmed COVID-19 cases near %s at around %s. Always ensure you maintain social distance",
					placeMark, approximateTime,
				),
				Timestamp: time.Now().Unix(),
				Type:      messaging.MessageType_WARNING,
				Data:      map[string]string{"sender": "location_api"},
			}, grpc.WaitForReady(true))
			if err != nil {
				lapi.logger.Errorf("failed to send user message on aerial covid-19 case: %v", err)
			}
		}()
	}

	return &empty.Empty{}, nil
}

const infectedUsers = "infected:users"

func (lapi *locationAPIServer) UpdateUserStatus(
	ctx context.Context, updateReq *location.UpdateUserStatusRequest,
) (*empty.Empty, error) {
	// Request must not be nil
	if updateReq == nil {
		return nil, services.NilRequestError("UpdateUserStatusRequest")
	}

	// Authorization
	err := lapi.authenticate(ctx)
	if err != nil {
		return nil, err
	}

	// Validation
	switch {
	case updateReq.PhoneNumber == "":
		err = services.MissingFieldError("user phone")
	case updateReq.Status.String() == "":
		err = services.MissingFieldError("status id")
	}
	if err != nil {
		return nil, err
	}

	tx := lapi.logsDB.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
		}
	}()
	if tx.Error != nil {
		return nil, services.FailedToBeginTx(err)
	}

	// Update status in database
	err = tx.Table(services.UsersTable).Where("phone_number=?", updateReq.PhoneNumber).
		Update("status", int8(updateReq.Status)).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user status: %v", err)
	}

	// Add user to list of users with COVID-19
	if updateReq.GetStatus() == location.Status_POSITIVE {
		_, err = lapi.eventsDB.LPush(ctx, infectedUsers, updateReq.PhoneNumber).Result()
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

func (lapi *locationAPIServer) UpdateUser(
	ctx context.Context, updateReq *location.UpdateUserRequest,
) (*empty.Empty, error) {
	// Request must not be nil
	if updateReq == nil {
		return nil, services.NilRequestError("UpdateUserRequest")
	}

	// Authorization
	err := lapi.authenticate(ctx)
	if err != nil {
		return nil, err
	}

	// Validation
	switch {
	case updateReq.PhoneNumber == "":
		err = services.MissingFieldError("user phone")
	case updateReq.User == nil:
		err = services.MissingFieldError("user")
	}
	if err != nil {
		return nil, err
	}

	userDB, err := getUserDB(updateReq.GetUser())
	if err != nil {
		return nil, err
	}

	// Update status in database
	err = lapi.logsDB.Table(services.UsersTable).Omit("status").
		Where("phone_number=?", updateReq.PhoneNumber).Updates(userDB).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user status: %v", err)
	}

	return &empty.Empty{}, nil
}

func (lapi *locationAPIServer) AddUser(
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
	case addReq.User.County == "":
		err = services.MissingFieldError("county")
	}
	if err != nil {
		return nil, err
	}

	userModel, err := getUserDB(addReq.User)
	if err != nil {
		return nil, err
	}

	// If user already exists, performs an update
	alreadyExists := !lapi.logsDB.
		First(&services.UserModel{}, "phone_number=?", addReq.User.PhoneNumber).RecordNotFound()

	if alreadyExists {
		err = lapi.logsDB.Table(services.UsersTable).Where("phone_number=?", addReq.User.PhoneNumber).
			Omit("traced").Updates(userModel).Error
		switch {
		case err == nil:
		default:
			err = status.Errorf(codes.Internal, "saving user failed: %v", err)
		}
	} else {
		// Reset their status to unknown
		addReq.User.Status = location.Status_UNKNOWN
		err = lapi.logsDB.Create(userModel).Error
		switch {
		case err == nil:
		default:
			err = status.Errorf(codes.Internal, "creating user failed: %v", err)
		}

		// Send user a welcome notification
		_, err := lapi.messagingClient.SendMessage(ctx, &messaging.Message{
			UserPhone:    userModel.PhoneNumber,
			Title:        fmt.Sprintf("Hello %s", userModel.FullName),
			Notification: "Welcome to KoviTrace application.\nYou can do self-screening assessment, get qualitative information about COVID-19 and most important you will be notified in case you come into close contact with someone who has tested postive for COVID-19.",
			Timestamp:    time.Now().Unix(),
			Type:         messaging.MessageType_INFO,
			Data:         map[string]string{"sender": "location_api"},
		}, grpc.WaitForReady(true))
		if err != nil {
			lapi.logger.Errorf("failed to send user welcome message: %v", err)
		}
	}

	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

func (lapi *locationAPIServer) GetUser(
	ctx context.Context, getReq *location.GetUserRequest,
) (*location.User, error) {
	// Requets must not be nil
	if getReq == nil {
		return nil, services.NilRequestError("GetUserRequest")
	}

	// Authentication
	err := lapi.authenticate(ctx)
	if err != nil {
		return nil, err
	}

	if getReq.PhoneNumber == "" {
		return nil, services.MissingFieldError("phone number")
	}

	// Get from database
	userDB := &services.UserModel{}
	err = lapi.logsDB.First(userDB, "phone_number=?", getReq.PhoneNumber).Error
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		err = status.Errorf(codes.NotFound, "user not with phone number %s found: %v", getReq.PhoneNumber, err)
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

func (lapi *locationAPIServer) ListUsers(
	ctx context.Context, listReq *location.ListUsersRequest,
) (*location.Users, error) {
	// Request must not be nil
	if listReq == nil {
		return nil, services.NilRequestError("ListUsersRequest")
	}

	// Authenticate the request
	err := lapi.authenticate(ctx)
	if err != nil {
		return nil, err
	}

	// Normalize page
	pageToken, pageSize := normalizePageSize(listReq.PageToken, listReq.PageSize)

	db := lapi.logsDB.Order("id, created_at ASC").Where("id>?", pageToken).Limit(pageSize)
	if listReq.FilterStatus != location.Status_UNKNOWN {
		db = db.Where("status=?", int8(listReq.FilterStatus))
	}

	usersDB := make([]*services.UserModel, 0)
	err = db.Find(&usersDB).Error
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
		pageToken = int(userDB.ID)
	}

	return &location.Users{
		Users:         usersPB,
		NextPageToken: int32(pageToken),
	}, nil
}

func (lapi *locationAPIServer) SearchUsers(
	ctx context.Context, searchReq *location.SearchUsersRequest,
) (*location.Users, error) {
	// Request must not be nil
	if searchReq == nil {
		return nil, services.NilRequestError("SearchUsersRequest")
	}

	// Authenticate the request
	err := lapi.authenticate(ctx)
	if err != nil {
		return nil, err
	}

	// For empty queries
	if searchReq.Query == "" {
		return &location.Users{
			Users: []*location.User{},
		}, nil
	}

	// Normalize page
	pageToken, pageSize := normalizePageSize(searchReq.PageToken, searchReq.PageSize)

	searchReq.Query = strings.ReplaceAll(searchReq.Query, "+", "")

	parsedQuery := services.ParseQuery(searchReq.Query, "+", "users", "users")

	usersDB := make([]*services.UserModel, 0, pageSize)

	db := lapi.logsDB.Unscoped().Limit(pageSize).Order("id, created_at ASC").
		Where("id>?", pageToken)
	if searchReq.FilterStatus != location.Status_UNKNOWN {
		db = db.Where("status=?", int8(searchReq.FilterStatus))
	}

	err = db.Find(&usersDB, "MATCH(phone_number, full_name) AGAINST(? IN BOOLEAN MODE)", parsedQuery).Error
	switch {
	case err == nil:
	default:
		return nil, status.Errorf(codes.Internal, "failed to search users: %v", err)
	}

	// Populate response
	usersPB := make([]*location.User, 0, len(usersDB))

	for _, userDB := range usersDB {
		userPB, err := getUserPB(userDB)
		if err != nil {
			return nil, err
		}
		usersPB = append(usersPB, userPB)
	}

	return &location.Users{
		NextPageToken: int32(pageToken),
		Users:         usersPB,
	}, nil
}

const defaultPageSize = 10

func normalizePageSize(pageToken, pageSize int32) (int, int) {
	if pageToken <= 0 {
		pageToken = 0
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > defaultPageSize {
		pageSize = defaultPageSize
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
		PhoneNumber:      userDB.PhoneNumber,
		FullName:         userDB.FullName,
		County:           userDB.County,
		Status:           location.Status(userDB.Status),
		DeviceToken:      userDB.DeviceToken,
		Traced:           userDB.Traced,
		UpdatedTimestamp: userDB.UpdatedAt.Unix(),
	}
	return userPB, nil
}
