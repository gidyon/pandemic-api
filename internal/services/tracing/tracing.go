package tracing

import (
	"context"
	"errors"
	"fmt"
	"github.com/gidyon/pandemic-api/pkg/api/contact_tracing"
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"github.com/gidyon/pandemic-api/pkg/api/messaging"
	"io"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/grpclog"

	"github.com/google/uuid"

	"google.golang.org/grpc"

	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"

	"github.com/gidyon/pandemic-api/internal/services"
	"github.com/go-redis/redis"

	"github.com/jinzhu/gorm"
)

type tracingAPIServer struct {
	failedDayContactChan chan *dayContact
	contactDataChan      chan *messaging.ContactData
	logger               grpclog.LoggerV2
	sqlDB                *gorm.DB
	redisDB              *redis.Client
	messagingClient      messaging.MessagingClient
}

// Options contains options for creating tracing API
type Options struct {
	SQLDB           *gorm.DB
	RedisClient     *redis.Client
	MessagingClient messaging.MessagingClient
	Logger          grpclog.LoggerV2
}

// NewContactTracingAPI creates a new contact tracing API server
func NewContactTracingAPI(
	ctx context.Context, opt *Options,
) (contact_tracing.ContactTracingServer, error) {
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
		contactDataChan:      make(chan *messaging.ContactData, 0),
		sqlDB:                opt.SQLDB,
		redisDB:              opt.RedisClient,
		messagingClient:      opt.MessagingClient,
		logger:               opt.Logger,
	}

	// Automigration
	err = ms.sqlDB.AutoMigrate(&services.ContactTracingOperation{}).Error
	if err != nil {
		return nil, fmt.Errorf("failed to automigrate: %v", err)
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
	ctx context.Context, traceReq *contact_tracing.TraceUserLocationsRequest,
) (*contact_tracing.ContactTracingResponse, error) {
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
	err = t.sqlDB.Select("status, full_name, phone_number, county, id").
		First(userDB, "phone_number=?", traceReq.PhoneNumber).Error
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, status.Errorf(codes.NotFound, "user with phone %s not found", traceReq.PhoneNumber)
	default:
		return nil, status.Errorf(codes.NotFound, "error happened: %v", err)
	}

	// User status must be positive
	if userDB.Status != int8(location.Status_POSITIVE) {
		return nil, status.Error(codes.FailedPrecondition, "user must be infected with COVID-19")
	}

	// Get user logs for the last n days
	sinceDate, err := time.Parse("2006-01-02", traceReq.SinceDate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse since date: %v", err)
	}
	todayDate := time.Now()

	if todayDate.Sub(sinceDate) <= 0 {
		return nil, status.Error(codes.FailedPrecondition, "since date cannot be greater than today")
	}

	// Create stream to messaging
	client, err := t.messagingClient.AlertContacts(context.Background(), grpc.WaitForReady(true))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create stream for sending messages: %v", err)
	}

	// Save operation
	operationDB := &services.ContactTracingOperation{
		County:      userDB.County,
		Description: fmt.Sprintf("%s - %s", userDB.FullName, userDB.PhoneNumber),
		Status:      int8(contact_tracing.OperationStatus_PENDING),
		Name:        fmt.Sprintf("TraceUserLocations::%s", uuid.New().String()),
	}
	err = t.sqlDB.Create(operationDB).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save operation: %v", err)
	}

	// Longrunning worker
	go t.traceUserWorker(client, operationDB.ID, traceReq.GetCounties(), userDB, &sinceDate)

	return &contact_tracing.ContactTracingResponse{
		OperationId: int64(operationDB.ID),
	}, nil
}

const limit = 1000

func (t *tracingAPIServer) traceUserWorker(
	messagingStream messaging.Messaging_AlertContactsClient,
	longrunningID uint,
	traceCounties []string,
	userDB *services.UserModel,
	sinceDate *time.Time,
) {
	defer func() {
		_, err := messagingStream.CloseAndRecv()
		if err != nil {
			t.logger.Errorf("error while closing stream: %v", err)
		}
	}()

	if sinceDate == nil {
		last2WeeksDur := 7 * 24 * time.Hour
		last2WeeksTime := time.Now().Add(-last2WeeksDur)
		sinceDate = &last2WeeksTime
	}

	var (
		todayDate = time.Now()
		condition = true
		offset    = 0
		err       error
		days      = int(todayDate.Sub(*sinceDate).Hours()/24) + 1
		usersDB   []*services.UserModel
		mu        = &sync.Mutex{}
		complete  = true
	)

	for condition {
		usersDB = make([]*services.UserModel, 0, limit)

		// Only those whose current status is not known
		db := t.sqlDB.Limit(limit).Offset(offset).Where("status=?", int8(location.Status_UNKNOWN))

		if len(traceCounties) > 0 {
			db = db.Where("county IN(?)", traceCounties)
		}

		err = db.Find(&usersDB).Error
		if err != nil {
			errMsg := fmt.Sprintf("failed to get users to send messages: %v", err)
			t.logger.Error(errMsg)
			complete = false
			t.failLongRunningOperation(longrunningID, errMsg)
			return
		}

		if len(usersDB) < limit {
			condition = false
		}

		wg := &sync.WaitGroup{}

		// For each user
		for _, suspect := range usersDB {
			if suspect.ID == userDB.ID {
				continue
			}

			wg.Add(1)

			go func(suspect *services.UserModel) {
				defer wg.Done()

				contactData := &messaging.ContactData{
					Count:        0,
					PatientPhone: userDB.PhoneNumber,
					UserPhone:    suspect.PhoneNumber,
					FullName:     suspect.FullName,
					DeviceToken:  suspect.DeviceToken,
					ContactTime:  sinceDate.String(),
				}

				pipeliner := t.redisDB.Pipeline()

				since := *sinceDate

				resChan := make(chan *redis.StringSliceCmd, days)

				for since.Unix() <= todayDate.Unix() {
					// Get union of contact points
					resChan <- pipeliner.SInter(
						getUserSetKey(suspect.PhoneNumber, &since), getUserSetKey(userDB.PhoneNumber, &since),
					)

					since = since.Add(time.Hour * 24)
				}

				close(resChan)

				_, err := pipeliner.Exec()
				if err != nil {
					mu.Lock()
					complete = false
					mu.Unlock()
					errMsg := fmt.Sprintf("error while executing pipeline: %v", err)
					t.logger.Error(errMsg)
					t.failLongRunningOperation(longrunningID, errMsg)
					return
				}

				for res := range resChan {

					contacts, err := res.Result()
					if err != nil {
						mu.Lock()
						complete = false
						mu.Unlock()
						errMsg := fmt.Sprintf("failed to get contact points: %v", err)
						t.logger.Error(errMsg)
						t.failLongRunningOperation(longrunningID, errMsg)
						return
					}

					// Range individual contact points
					if len(contacts) > 0 {
						contactPointData := strings.Split(contacts[0], ":")
						if len(contactPointData) != 3 {
							mu.Lock()
							complete = false
							mu.Unlock()
							errMsg := "empty contact point data"
							t.logger.Error(errMsg)
							t.failLongRunningOperation(longrunningID, errMsg)
							return
						}

						contactData.ContactTime = contactPointData[2]
						contactData.Count = int32(len(contacts))
					}
				}

				if contactData.Count > 0 {
					// Change their status to suspected
					err = t.sqlDB.Table(services.UsersTable).Where("phone_number=?", suspect.PhoneNumber).
						Update("status", int8(location.Status_SUSPECTED)).Error
					if err != nil {
						mu.Lock()
						complete = false
						mu.Unlock()
						errMsg := fmt.Sprintf("error while updating user status: %v", err)
						t.logger.Error(errMsg)
						t.failLongRunningOperation(longrunningID, errMsg)
						return
					}

					// Send contact data to messaging server
					err = messagingStream.Send(contactData)
					switch {
					case err == nil:
					case errors.Is(err, io.EOF):
						break
					default:
						mu.Lock()
						complete = false
						mu.Unlock()
						errMsg := fmt.Sprintf("error while sending to stream: %v", err)
						t.logger.Error(errMsg)
						t.failLongRunningOperation(longrunningID, errMsg)
						return
					}
				}
			}(suspect)
		}

		wg.Wait()

		offset += len(usersDB)
	}

	if complete {
		err = t.completeLongRunningOperation(longrunningID)
		if err != nil {
			errMsg := fmt.Sprintf("failed to mark long running operation as complete: %v", err)
			t.failLongRunningOperation(longrunningID, errMsg)
			return
		}

		// Update status of user to traced
		err = t.sqlDB.Table(services.UsersTable).Where("phone_number=?", userDB.PhoneNumber).
			Update("traced", true).Error
		if err != nil {
			errMsg := fmt.Sprintf("failed to update user traced status: %v", err)
			t.logger.Error(errMsg)
			t.failLongRunningOperation(longrunningID, errMsg)
			return
		}
	}
}

func (t *tracingAPIServer) failLongRunningOperation(longrunningID uint, errMsg string) {
	// Get the longrunning operation
	operationDB := &services.ContactTracingOperation{}
	err := t.sqlDB.First(operationDB, "id=?", longrunningID).Error
	if err != nil {
		t.logger.Errorf("failed to get longrunning operation from database: %v", err)
		return
	}

	operationDB.Result = fmt.Sprintf("the operation failed: %s", errMsg)
	operationDB.Status = int8(contact_tracing.OperationStatus_FAILED)

	// Save back to cache
	err = t.sqlDB.Table(services.ContactTracingOperationTable).Where("id=?", longrunningID).
		Updates(operationDB).Error
	if err != nil {
		t.logger.Errorf("failed to update longrunning operation: %v", err)
		return
	}
}

func (t *tracingAPIServer) completeLongRunningOperation(longrunningID uint) error {
	// Get the longrunning operation
	operationDB := &services.ContactTracingOperation{}
	err := t.sqlDB.First(operationDB, "id=?", longrunningID).Error
	if err != nil {
		t.logger.Errorf("failed to get longrunning operation from database: %v", err)
		return err
	}

	operationDB.Result = "operation completed successfully"
	operationDB.Status = int8(contact_tracing.OperationStatus_COMPLETED)

	// Save back to cache
	err = t.sqlDB.Table(services.ContactTracingOperationTable).Where("id=?", longrunningID).
		Updates(operationDB).Error
	if err != nil {
		t.logger.Errorf("failed to update longrunning operation: %v", err)
		return err
	}

	return nil
}

func (t *tracingAPIServer) TraceUsersLocations(
	ctx context.Context, traceReq *contact_tracing.TraceUsersLocationsRequest,
) (*contact_tracing.ContactTracingResponse, error) {
	// Request must not be nil
	if traceReq == nil {
		return nil, services.MissingFieldError("TraceUsersLocationsRequest")
	}

	// Validation
	var err error
	switch {
	case traceReq.SinceDate == "":
		err = services.MissingFieldError("since date")
	}
	if err != nil {
		return nil, err
	}

	// Get users logs for last n days
	sinceDate, err := time.Parse("2006-01-02", traceReq.SinceDate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse since date: %v", err)
	}
	todayDate := time.Now()

	if todayDate.Sub(sinceDate) <= 0 {
		return nil, status.Error(codes.FailedPrecondition, "since date cannot be greater than today")
	}

	// Create stream to messaging
	client, err := t.messagingClient.AlertContacts(context.Background(), grpc.WaitForReady(true))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create stream for sending messages: %v", err)
	}

	limit := 1000
	offset := 0
	counties := []string{}
	condition := true

	usersDB := make([]*services.UserModel, 0, limit)

	for condition {
		db := t.sqlDB.Select("status, id, full_name, phone_number, county")
		if len(traceReq.Counties) > 0 {
			db = db.Where("county IN(?)", traceReq.Counties)
		}
		err = db.Find(usersDB, "status=?", int8(location.Status_POSITIVE)).Error
		switch {
		case err == nil:
		default:
			return nil, status.Errorf(codes.Internal, "failed to find users: %v", err)
		}

		if len(usersDB) < limit {
			condition = false
		}

		wg := &sync.WaitGroup{}

		for _, userDB := range usersDB {
			if userDB.Traced {
				continue
			}

			// Save operation
			county := strings.Join(traceReq.Counties, ", ")
			operationDB := &services.ContactTracingOperation{
				County:      county,
				Description: fmt.Sprintf("Cases from %s county", county),
				Status:      int8(contact_tracing.OperationStatus_PENDING),
				Name:        fmt.Sprintf("TraceUserLocations::%s", uuid.New().String()),
			}
			err = t.sqlDB.Create(operationDB).Error
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to save operation: %v", err)
			}

			// Longrunning worker
			wg.Add(1)
			go func(userDB *services.UserModel) {
				defer wg.Done()
				t.traceUserWorker(client, operationDB.ID, counties, userDB, &sinceDate)
			}(userDB)
		}

		wg.Wait()

		offset += len(usersDB)
	}

	return &contact_tracing.ContactTracingResponse{}, nil
}

func (t *tracingAPIServer) ListOperations(
	ctx context.Context, listReq *contact_tracing.ListOperationsRequest,
) (*contact_tracing.ListOperationsResponse, error) {
	// Request must not be nil
	if listReq == nil {
		return nil, services.NilRequestError("ListOperationsRequest")
	}

	// Parse page size and page token
	pageNumber, pageSize := services.NormalizePage(listReq.PageToken, listReq.PageSize)
	offset := pageNumber*pageSize - pageSize

	db := t.sqlDB.Order("created_at DESC").Offset(offset).Limit(pageSize)
	if len(listReq.Counties) > 0 {
		db = db.Where("county IN(?)", listReq.Counties)
	}

	operationsDB := make([]*services.ContactTracingOperation, 0)

	err := db.Find(&operationsDB).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get operations: %v", err)
	}

	operationsPB := make([]*contact_tracing.ContactTracingOperation, 0, len(operationsDB))

	for _, operationDB := range operationsDB {
		operationPB, err := getOperationPB(operationDB)
		if err != nil {
			return nil, err
		}
		operationsPB = append(operationsPB, operationPB)
	}

	return &contact_tracing.ListOperationsResponse{
		Operations:    operationsPB,
		NextPageToken: int32(pageNumber + 1),
	}, nil
}

func getOperationPB(operationDB *services.ContactTracingOperation) (*contact_tracing.ContactTracingOperation, error) {
	operationPB := &contact_tracing.ContactTracingOperation{
		Id:          int64(operationDB.ID),
		Status:      contact_tracing.OperationStatus(operationDB.Status),
		County:      operationDB.County,
		Description: operationDB.Description,
		Name:        operationDB.Name,
		Result:      operationDB.Result,
		Timestamp:   operationDB.CreatedAt.Unix(),
	}

	return operationPB, nil
}

func (t *tracingAPIServer) sendContactData(contactData *messaging.ContactData) {
	select {
	case <-time.After(5 * time.Second):
	case t.contactDataChan <- contactData:
	}
}
