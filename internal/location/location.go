package location

import (
	"context"
	"errors"
	"fmt"
	"github.com/gidyon/fightcovid19/pkg/api/location"
	"github.com/go-redis/redis"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math"
	"strings"
	"time"
)

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
		err = errors.New("context must not be nil")
	case opt.LogsDB == nil:
		err = errors.New("logsDB must not be nil")
	case opt.EventsDB == nil:
		err = errors.New("eventsDB must bot be nil")
	}
	if err != nil {
		return nil, err
	}

	locationManager := &locationAPIServer{
		logsDB:   opt.LogsDB,
		eventsDB: opt.EventsDB,
	}

	return locationManager, nil
}

var nilRequestError = func(request string) error {
	return status.Errorf(codes.FailedPrecondition, "%s must not be nil", request)
}

var missingFieldError = func(field string) error {
	return status.Errorf(codes.FailedPrecondition, "missing %s", field)
}

func (locationManager *locationAPIServer) validateAndSaveLocation(userID string, locationPB *location.Location) error {
	// Validation
	var err error
	switch {
	case userID == "":
		err = missingFieldError("user id")
	case locationPB == nil:
		err = missingFieldError("location")
	case locationPB.GetLongitude() == 0.0:
		err = missingFieldError("location longitude")
	case locationPB.GetLatitude() == 0.0:
		err = missingFieldError("location latitude")
	}
	if err != nil {
		return err
	}

	// Save in database
	locationDB := getLocationDB(locationPB)
	locationDB.UserID = userID

	err = locationManager.logsDB.Create(locationDB).Error
	if err != nil {
		return status.Errorf(codes.Internal, "failed to save to database: %v", err)
	}
	return nil
}

func (locationManager *locationAPIServer) SaveCurrentLocation(
	ctx context.Context, saveReq *location.SaveCurrentLocationRequest,
) (*empty.Empty, error) {
	// Request must not be nil
	if saveReq == nil {
		return nil, nilRequestError("SaveCurrentLocationRequest")
	}

	err := locationManager.validateAndSaveLocation(saveReq.GetUserId(), saveReq.GetLocation())
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

const saveIntervalSeconds = 60 * 5

func getTimeBlock(hours, minutes int) int {
	allMinutes := (hours*60 + minutes)
	return int(math.Trunc(float64(allMinutes*60) / (saveIntervalSeconds)))
}

func validateSharedData(sharedData *location.Shared) error {
	// Validation
	var err error
	longitude := sharedData.GetLocation().GetLongitude()
	latitude := sharedData.GetLocation().GetLatitude()
	switch {
	case sharedData == nil:
		err = missingFieldError("shared data")
	case sharedData.StatusId == "":
		err = missingFieldError("status id")
	case sharedData.GetLocation() == nil:
		err = missingFieldError("location")
	case longitude == 0.0:
		err = missingFieldError("location longitude")
	case latitude == 0.0:
		err = missingFieldError("location latitude")
	}
	if err != nil {
		return err
	}
	return nil
}

func getGeoFenceID(lat, long float32) string {
	return fmt.Sprintf("lat:%f long:%f", lat, long)
}

func getGeoFenceKey(lat, long float32) string {
	now := time.Now()
	return fmt.Sprintf(
		"%d:%d:fence:%s",
		now.Day(),
		getTimeBlock(now.Hour(), now.Minute()),
		getGeoFenceID(lat, long),
	)
}

func getLocationValue(statusID, userID string) string {
	return fmt.Sprintf("%s:%s", statusID, userID)
}
func (locationManager *locationAPIServer) ShareCurrentLocation(
	ctc context.Context, shareReq *location.ShareCurrentLocationRequest,
) (*empty.Empty, error) {
	// Request must not be nil
	if shareReq == nil {
		return nil, nilRequestError("ShareCurrentLocationRequest")
	}

	// Validation
	var err error
	if shareReq.UserId == "" {
		return nil, missingFieldError("user id")
	}
	err = validateSharedData(shareReq.Data)
	if err != nil {
		return nil, err
	}

	// Construct key
	now := time.Now()
	key := fmt.Sprintf(
		"%d:%d:fence:%s",
		now.Day(),
		getTimeBlock(now.Hour(), now.Minute()),
		getGeoFenceID(
			shareReq.GetData().GetLocation().GetLatitude(),
			shareReq.GetData().GetLocation().GetLongitude(),
		),
	)

	// Save to set; val => status_is:user_id
	err = locationManager.eventsDB.SAdd(
		key, getLocationValue(shareReq.Data.GetStatusId(), shareReq.GetUserId()),
	).Err()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save shared log: %v", err)
	}

	return &empty.Empty{}, nil
}

func (locationManager *locationAPIServer) BatchSaveLocation(
	ctx context.Context, batchReq *location.BatchSaveLocationRequest,
) (*empty.Empty, error) {
	// Request must not be nil
	if batchReq == nil {
		return nil, nilRequestError("BatchSaveLocationRequest")
	}

	// Validation
	var err error
	switch {
	case batchReq.UserId == "":
		err = missingFieldError("user id")
	case len(batchReq.Locations) == 0:
		err = missingFieldError("locations")
	}
	if err != nil {
		return nil, err
	}

	// Save individual locations
	for _, locationPB := range batchReq.GetLocations() {
		err = locationManager.validateAndSaveLocation(batchReq.UserId, locationPB)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (locationManager *locationAPIServer) BatchShareLocation(
	ctx context.Context, batchReq *location.BatchShareLocationRequest,
) (*empty.Empty, error) {
	// Request must not be nil
	if batchReq == nil {
		return nil, nilRequestError("BatchShareLocationRequest")
	}

	// Validation
	var err error
	switch {
	case batchReq.UserId == "":
		err = missingFieldError("user id")
	case len(batchReq.Datas) == 0:
		err = missingFieldError("datas")
	}
	if err != nil {
		return nil, err
	}

	// Start pipeline
	txPipeliner := locationManager.eventsDB.TxPipeline()

	for _, sharedData := range batchReq.GetDatas() {
		// Validate
		err = validateSharedData(sharedData)
		if err != nil {
			return nil, err
		}

		// Construct key and value
		key := getGeoFenceKey(
			sharedData.GetLocation().GetLatitude(),
			sharedData.GetLocation().GetLongitude(),
		)
		val := getLocationValue(sharedData.GetStatusId(), batchReq.GetUserId())

		// Save event
		err = txPipeliner.SAdd(key, val).Err()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to save shared log: %v", err)
		}
	}

	// Execute pipeline
	_, err = txPipeliner.ExecContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save all: %v", err)
	}

	return &empty.Empty{}, nil
}

func (locationManager *locationAPIServer) GetLocationCases(
	ctx context.Context, getReq *location.Location,
) (*location.SharedDatas, error) {
	// Request must not be nil
	if getReq == nil {
		return nil, nilRequestError("Location")
	}

	// Validation
	var err error
	switch {
	case getReq.GetLongitude() == 0.0:
		err = missingFieldError("location longitude")
	case getReq.GetLatitude() == 0.0:
		err = missingFieldError("location latitude")
	}
	if err != nil {
		return nil, err
	}

	// Query for info
	key := getGeoFenceKey(getReq.Latitude, getReq.Longitude)
	cases, err := locationManager.eventsDB.SMembers(key).Result()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get fence activities: %v", err)
	}

	datas := make([]*location.SharedDatas_SharedData, 0, len(cases))

	for _, sharedCase := range cases {
		shared := strings.Split(sharedCase, ":")
		datas = append(datas, &location.SharedDatas_SharedData{
			UserId:   shared[0],
			StatusId: shared[1],
		})
	}

	return &location.SharedDatas{
		Datas: datas,
	}, nil
}

func (locationManager *locationAPIServer) AddUser(
	ctx context.Context, addReq *location.AddUserRequest,
) (*empty.Empty, error) {
	// Request must not be  nil
	if addReq == nil {
		return nil, nilRequestError("AddUserRequest")
	}

	// Validation
	var err error
	switch {
	case addReq.PhoneNumber == "":
		err = missingFieldError("phone number")
	case addReq.StatusId == "":
		err = missingFieldError("status id")
	}
	if err != nil {
		return nil, err
	}

	// Save user
	err = locationManager.logsDB.Create(&userModel{
		UserID:      uuid.New().String(),
		PhoneNumber: addReq.PhoneNumber,
		Status:      addReq.StatusId,
	}).Error
	switch {
	case err == nil:
	case strings.Contains(strings.ToLower(err.Error()), "duplicate"):
		err = status.Error(codes.ResourceExhausted, "phone already registred")
	default:
		err = status.Errorf(codes.Internal, "saving user failed: %v", err)
	}
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

func (locationManager *locationAPIServer) UpdateUserStatus(
	ctx context.Context, updatReq *location.UpdateUserStatusRequest,
) (*empty.Empty, error) {
	// Request must not be nil
	if updatReq == nil {
		return nil, nilRequestError("UpdateUserStatusRequest")
	}

	// Validation
	var err error
	switch {
	case updatReq.UserId == "":
		err = missingFieldError("user id")
	case updatReq.Status == "":
		err = missingFieldError("status id")
	}
	if err != nil {
		return nil, err
	}

	// Update status in database
	err = locationManager.logsDB.Table(usersTable).Where("user_id=?", updatReq.UserId).
		Update("status", updatReq.Status).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update: %v", err)
	}

	return &empty.Empty{}, nil
}
