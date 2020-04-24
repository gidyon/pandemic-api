package services

// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"strings"
// 	"time"

// 	"google.golang.org/protobuf/proto"

// 	"github.com/gidyon/pandemic-api/pkg/api/location"

// 	"github.com/go-redis/redis"
// 	"github.com/golang/protobuf/ptypes/empty"
// 	"github.com/google/uuid"
// 	"github.com/jinzhu/gorm"
// 	"google.golang.org/grpc/codes"
// 	"google.golang.org/grpc/status"
// )

// const timeBoundary = time.Duration(5 * time.Minute)

// type userUpdates struct {
// 	UserID    string
// 	Number    int64
// 	CreatedAt time.Time
// 	UpdateAt  time.Time
// }

// type locationAPIServer struct {
// 	logsDB   *gorm.DB
// 	eventsDB *redis.Client
// }

// // Options contains parameters for NewLocationTracing
// type Options struct {
// 	LogsDB   *gorm.DB
// 	EventsDB *redis.Client
// }

// // NewLocationTracing creates a new location tracing API
// func NewLocationTracing(ctx context.Context, opt *Options) (location.LocationTracingAPIServer, error) {
// 	var err error
// 	// Validation
// 	switch {
// 	case ctx == nil:
// 		err = errors.New("context must not be nil")
// 	case opt.LogsDB == nil:
// 		err = errors.New("logsDB must not be nil")
// 	case opt.EventsDB == nil:
// 		err = errors.New("eventsDB must bot be nil")
// 	}
// 	if err != nil {
// 		return nil, err
// 	}

// 	locationManager := &locationAPIServer{
// 		logsDB:   opt.LogsDB,
// 		eventsDB: opt.EventsDB,
// 	}

// 	// Automigration
// 	err = locationManager.logsDB.AutoMigrate(&locationModel{}, &userModel{}).Error
// 	if err != nil {
// 		return nil, err
// 	}

// 	return locationManager, nil
// }

// var nilRequestError = func(request string) error {
// 	return status.Errorf(codes.FailedPrecondition, "%s must not be nil", request)
// }

// var missingFieldError = func(field string) error {
// 	return status.Errorf(codes.FailedPrecondition, "missing %s", field)
// }

// func getTimestampRange(locationPb *location.Location) (int64, int64) {
// 	min := locationPb.Timestamp / timeBoundary.Milliseconds()
// 	max := min + timeBoundary.Milliseconds()
// 	return min, max
// }

// func getCaseKey(maxTimestamp int64, geoFenceID string) string {
// 	// get timestamp
// 	return fmt.Sprintf("%d:fence_id:%s", maxTimestamp, geoFenceID)
// }

// const (
// 	sendLocatiosSet = "sendlocations"
// 	updatesStream   = "updates:stream"
// 	updatesList     = "updates:list"
// )

// func (locationManager *locationAPIServer) GetActions(
// 	ctx context.Context, getReq *location.GetActionsRequest,
// ) (*location.GetActionsResponse, error) {
// 	// Request must not be nil
// 	if getReq == nil {
// 		return nil, nilRequestError("GetActionsRequest")
// 	}

// 	// Validation
// 	if getReq.UserId == "" {
// 		return nil, missingFieldError("user id")
// 	}

// 	// Read from send locations set
// 	isMember, err := locationManager.eventsDB.SIsMember(sendLocatiosSet, getReq.UserId).Result()
// 	if err != nil {
// 		return nil, status.Errorf(codes.Internal, "failed to check for updates: %v", err)
// 	}

// 	// Read from updates channel
// 	l, err := locationManager.eventsDB.XLen(updatesStream).Result()
// 	if err != nil {
// 		return nil, status.Errorf(codes.Internal, "failed to check updates: %v", err)
// 	}

// 	// Get user meta
// 	updates := &userUpdates{}
// 	newUpdates := false
// 	err = locationManager.logsDB.First(updates, "user_id=?", getReq.UserId).Error
// 	if err != nil {
// 		return nil, status.Errorf(codes.Internal, "failed to get user updates: %v", err)
// 	}

// 	if updates.Number < l {
// 		// New updates
// 		newUpdates = true
// 	}

// 	// Populate response
// 	resp := &location.GetActionsResponse{}
// 	switch {
// 	case isMember && newUpdates:
// 		resp.Action = &location.GetActionsResponse_Both{
// 			Both: &location.TimeFilter{
// 				StartTimestampSec: updates.UpdateAt.Unix(),
// 				EndTimestampSec:   0,
// 			},
// 		}
// 	case isMember:
// 		resp.Action = &location.GetActionsResponse_GetUpdates{
// 			GetUpdates: &location.GetUpdatesPayload{
// 				Count:             l - updates.Number,
// 				StartTimestampSec: updates.UpdateAt.Unix(),
// 			},
// 		}
// 	case newUpdates:
// 		resp.Action = &location.GetActionsResponse_SendLocations{
// 			SendLocations: &location.TimeFilter{
// 				StartTimestampSec: updates.UpdateAt.Unix(),
// 				EndTimestampSec:   0,
// 			},
// 		}
// 	}

// 	return resp, nil
// }

// func (locationManager *locationAPIServer) GetLocationCases(
// 	ctx context.Context, getReq *location.Location,
// ) (*location.LocationCases, error) {
// 	// Request must not be nil
// 	if getReq == nil {
// 		return nil, nilRequestError("Location")
// 	}

// 	// Validation
// 	var err error
// 	switch {
// 	case getReq.GetLongitude() == 0.0:
// 		err = missingFieldError("location longitude")
// 	case getReq.GetLatitude() == 0.0:
// 		err = missingFieldError("location latitude")
// 	case getReq.GetGeoFenceId() == "":
// 		err = missingFieldError("location geofencd id")
// 	case getReq.Timestamp == 0:
// 		err = missingFieldError("timestamp")
// 	}
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Query for info
// 	_, max := getTimestampRange(getReq)
// 	key := getCaseKey(max, getReq.GeoFenceId)
// 	cases, err := locationManager.eventsDB.SMembers(key).Result()
// 	if err != nil {
// 		return nil, status.Errorf(codes.Internal, "failed to get fence activities: %v", err)
// 	}

// 	locationCases := make([]*location.LocationCase, 0, len(cases))

// 	for _, sharedCase := range cases {
// 		locationCase := &location.LocationCase{}
// 		err = proto.Unmarshal([]byte(sharedCase), locationCase)
// 		if err != nil {
// 			return nil, status.Errorf(codes.Internal, "failed to proto marshal: %v", err)
// 		}
// 		locationCases = append(locationCases, locationCase)
// 	}

// 	return &location.LocationCases{
// 		Cases: locationCases,
// 	}, nil
// }

// func (locationManager *locationAPIServer) GetUpdates(
// 	ctx context.Context, getReq *location.GetUpdatesRequest,
// ) (*location.LocationCases, error) {
// 	// Request must not be nil
// 	if getReq == nil {
// 		return nil, nilRequestError("GetUpdatesRequest")
// 	}

// 	// Validation
// 	var err error
// 	switch {
// 	case getReq.UserId == "":
// 		err = missingFieldError("user id")
// 	case getReq.StartIndex == 0:
// 		err = missingFieldError("start index")
// 	case getReq.Count == 0:
// 		err = missingFieldError("count")
// 	}
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Get the updates
// 	updates, err := locationManager.eventsDB.LRange(
// 		updatesList, getReq.StartIndex, getReq.StartIndex+getReq.Count,
// 	).Result()
// 	if err != nil {
// 		return nil, status.Errorf(codes.Internal, "failed to get updates: %v", err)
// 	}

// 	allLocationCases := make([]*location.LocationCase, 0, len(updates))

// 	for _, update := range updates {
// 		locationCases, err := locationManager.eventsDB.SMembers(update).Result()
// 		if err != nil {
// 			return nil, status.Errorf(codes.Internal, "failed to get updates: %v", err)
// 		}

// 		for _, sharedCase := range locationCases {
// 			locationCase := &location.LocationCase{}
// 			err = proto.Unmarshal([]byte(sharedCase), locationCase)
// 			if err != nil {
// 				return nil, status.Errorf(codes.Internal, "failed to proto marshal: %v", err)
// 			}
// 			allLocationCases = append(allLocationCases, locationCase)
// 		}
// 	}

// 	return &location.LocationCases{
// 		Cases: allLocationCases,
// 	}, nil
// }

// func validateLocationCase(locationCase *location.LocationCase) error {
// 	// Validation
// 	locationPB := locationCase.GetLocation()
// 	// Validation
// 	var err error
// 	switch {
// 	case locationCase == nil:
// 		err = missingFieldError("location case")
// 	case locationCase == nil:
// 		err = missingFieldError("location")
// 	case locationPB.GetLongitude() == 0.0:
// 		err = missingFieldError("location longitude")
// 	case locationPB.GetLatitude() == 0.0:
// 		err = missingFieldError("location latitude")
// 	case locationPB.GetGeoFenceId() == "":
// 		err = missingFieldError("location geo fence id")
// 	case locationPB.Timestamp == 0.0:
// 		err = missingFieldError("location timestamp")
// 	}
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (locationManager *locationAPIServer) validateAndSaveLocation(locationCase *location.LocationCase) error {
// 	err := validateLocationCase(locationCase)
// 	if err != nil {
// 		return err
// 	}

// 	// MArshal location case to bytes
// 	bs, err := proto.Marshal(locationCase)
// 	if err != nil {
// 		return status.Errorf(codes.Internal, "failed to marshal location: %v", err)
// 	}

// 	_, max := getTimestampRange(locationCase.GetLocation())
// 	key := getCaseKey(max, locationCase.GetLocation().GetGeoFenceId())

// 	// Save in events database
// 	_, err = locationManager.eventsDB.SAdd(key, bs).Result()
// 	if err != nil {
// 		return status.Errorf(codes.Internal, "failed to add to set: %v", err)
// 	}

// 	return nil
// }

// func (locationManager *locationAPIServer) AddLocationCase(
// 	ctx context.Context, addReq *location.LocationCase,
// ) (*empty.Empty, error) {
// 	// Request must not be nil
// 	if addReq == nil {
// 		return nil, nilRequestError("AddLocatiLocationCaseonCaseRequest")
// 	}

// 	err := locationManager.validateAndSaveLocation(addReq)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &empty.Empty{}, nil
// }

// func (locationManager *locationAPIServer) AddLocationCases(
// 	ctx context.Context, cases *location.LocationCases,
// ) (*empty.Empty, error) {
// 	// Request must not be nil
// 	if cases == nil {
// 		return nil, nilRequestError("LocationCases")
// 	}

// 	// Start pipeline
// 	txPipeliner := locationManager.eventsDB.TxPipeline()

// 	var err error
// 	for _, locationCase := range cases.GetCases() {
// 		// Validate
// 		err = validateLocationCase(locationCase)
// 		if err != nil {
// 			return nil, err
// 		}

// 		// MArshal location case to bytes
// 		bs, err := proto.Marshal(locationCase)
// 		if err != nil {
// 			return nil, status.Errorf(codes.Internal, "failed to marshal location: %v", err)
// 		}

// 		// Construct key
// 		_, max := getTimestampRange(locationCase.GetLocation())
// 		key := getCaseKey(max, locationCase.GetLocation().GetGeoFenceId())

// 		// Save event
// 		err = txPipeliner.SAdd(key, bs).Err()
// 		if err != nil {
// 			return nil, status.Errorf(codes.Internal, "failed to save shared log: %v", err)
// 		}

// 		// Save the case to list
// 		_, err = locationManager.eventsDB.LPush(updatesList, key).Result()
// 		if err != nil {
// 			return nil, status.Errorf(codes.Internal, "failed to add to cases: %v", err)
// 		}
// 	}

// 	// Execute pipeline
// 	_, err = txPipeliner.ExecContext(ctx)
// 	if err != nil {
// 		return nil, status.Errorf(codes.Internal, "failed to save all: %v", err)
// 	}

// 	return &empty.Empty{}, nil
// }

// func (locationManager *locationAPIServer) SendLocations(
// 	ctx context.Context, sendReq *location.SendLocationsRequest,
// ) (*empty.Empty, error) {
// 	// Request must not be nil
// 	if sendReq == nil {
// 		return nil, nilRequestError("SendLocationsRequest")
// 	}

// 	// Validation
// 	var err error
// 	switch {
// 	case sendReq.UserId == "":
// 		err = missingFieldError("user id")
// 	case len(sendReq.Locations) == 0:
// 		err = missingFieldError("locations")
// 	}
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Save individual locations
// 	for _, locationPB := range sendReq.GetLocations() {
// 		locationDB := getLocationDB(locationPB)
// 		err = locationManager.logsDB.Create(locationDB).Error
// 		if err != nil {
// 			return nil, status.Errorf(codes.Internal, "failed to save location: %v", err)
// 		}
// 	}

// 	return &empty.Empty{}, nil
// }

// const infectedUsers = "infeccted:users"

// func failedToBeginTx(err error) error {
// 	return status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
// }
// func failedCommitTx(err error) error {
// 	return status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
// }

// func (locationManager *locationAPIServer) UpdateUserStatus(
// 	ctx context.Context, updatReq *location.UpdateUserStatusRequest,
// ) (*empty.Empty, error) {
// 	// Request must not be nil
// 	if updatReq == nil {
// 		return nil, nilRequestError("UpdateUserStatusRequest")
// 	}

// 	// Validation
// 	var err error
// 	switch {
// 	case updatReq.UserId == "":
// 		err = missingFieldError("user id")
// 	case updatReq.Status == "":
// 		err = missingFieldError("status id")
// 	}
// 	if err != nil {
// 		return nil, err
// 	}

// 	tx := locationManager.logsDB.Begin()
// 	if tx.Error != nil {
// 		return nil, failedToBeginTx(err)
// 	}
// 	defer tx.Close()

// 	// Update status in database
// 	err = tx.Table(usersTable).Where("user_id=?", updatReq.UserId).
// 		Update("status", updatReq.Status).Error
// 	if err != nil {
// 		return nil, failedToBeginTx(err)
// 	}

// 	// Add user to list of users with COVID-19
// 	_, err = locationManager.eventsDB.LPush(infectedUsers, updatReq.UserId).Result()
// 	if err != nil {
// 		tx.Rollback()
// 		return nil, status.Errorf(codes.Internal, "failed to add users to list of infected users: %v", err)
// 	}

// 	err = tx.Commit().Error
// 	if err != nil {
// 		return nil, failedCommitTx(err)
// 	}

// 	return &empty.Empty{}, nil
// }

// func (locationManager *locationAPIServer) AddUser(
// 	ctx context.Context, addReq *location.AddUserRequest,
// ) (*empty.Empty, error) {
// 	// Request must not be  nil
// 	if addReq == nil {
// 		return nil, nilRequestError("AddUserRequest")
// 	}

// 	// Validation
// 	var err error
// 	switch {
// 	case addReq.PhoneNumber == "":
// 		err = missingFieldError("phone number")
// 	case addReq.StatusId == "":
// 		err = missingFieldError("status id")
// 	}
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Save user
// 	err = locationManager.logsDB.Create(&userModel{
// 		UserID:      uuid.New().String(),
// 		PhoneNumber: addReq.PhoneNumber,
// 		Status:      addReq.StatusId,
// 	}).Error
// 	switch {
// 	case err == nil:
// 	case strings.Contains(strings.ToLower(err.Error()), "duplicate"):
// 		err = status.Error(codes.ResourceExhausted, "phone already registred")
// 	default:
// 		err = status.Errorf(codes.Internal, "saving user failed: %v", err)
// 	}
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &empty.Empty{}, nil
// }
