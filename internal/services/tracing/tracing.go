package tracing

import (
	"context"
	"encoding/json"
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

	"google.golang.org/genproto/googleapis/longrunning"

	"google.golang.org/grpc"

	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"

	"github.com/gidyon/pandemic-api/internal/services"
	"github.com/go-redis/redis"

	"github.com/jinzhu/gorm"

	spb "google.golang.org/genproto/googleapis/rpc/status"
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

	// Create a long running operation
	longrunningOp := &longrunning.Operation{
		Name: fmt.Sprintf("TraceUserLocations::%s", uuid.New().String()),
		Done: false,
	}

	// Marshal to bytes json
	bs, err := json.Marshal(longrunningOp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal longrunning operation: %v", err)
	}

	// Save operation
	operationDB := &services.ContactTracingOperation{
		County:      userDB.County,
		Description: fmt.Sprintf("%s - %s", userDB.FullName, userDB.PhoneNumber),
		Done:        false,
		Payload:     bs,
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
	)

	days := int(todayDate.Sub(*sinceDate).Hours()/24) + 1

	var usersDB []*services.UserModel

	for condition {
		usersDB = make([]*services.UserModel, 0, limit)

		db := t.sqlDB.Limit(limit).Offset(offset)

		if len(traceCounties) > 0 {
			db = db.Where("county IN(?)", traceCounties)
		}

		err = db.Find(&usersDB).Error
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
			if suspect.ID == userDB.ID {
				continue
			}

			wg.Add(1)

			go func(suspect *services.UserModel) {
				defer wg.Done()

				contactData := &messaging.ContactData{
					Count:         0,
					PatientPhone:  userDB.PhoneNumber,
					UserPhone:     suspect.PhoneNumber,
					FullName:      suspect.FullName,
					DeviceToken:   suspect.DeviceToken,
					ContactPoints: make([]*messaging.ContactPoint, 0),
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
					t.logger.Error("error while executing pipeline: %v", err)
					t.sendContactData(contactData)
					return
				}

				for res := range resChan {

					contacts, err := res.Result()
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
						if len(contactPointData) != 3 {
							t.logger.Warning("empty contact point data")
							continue
						}

						contactData.Count++
						contactData.ContactPoints = append(contactData.ContactPoints, &messaging.ContactPoint{
							GeoFenceId: contactPointData[0] + contactPointData[1],
							TimeId:     contactPointData[2],
						})
					}
				}

				if contactData.Count > 0 {
					// Change their status to suspected
					err = t.sqlDB.Table(services.UsersTable).Where("phone_number=?", suspect.PhoneNumber).
						Update("status", int8(location.Status_SUSPECTED)).Error
					if err != nil {
						t.logger.Error("error while updating user status: %v", err)
						t.sendContactData(contactData)
					}

					// Send contact data to messaging server
					err = messagingStream.Send(contactData)
					switch {
					case err == nil:
					case errors.Is(err, io.EOF):
						break
					default:
						t.logger.Error("error while sending to stream: %v", err)
						t.sendContactData(contactData)
						return
					}
				}
			}(suspect)
		}

		wg.Wait()

		offset += len(usersDB)
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

	// Unmarshal to proto message
	longrunningOp := &longrunning.Operation{}
	err = json.Unmarshal(operationDB.Payload, longrunningOp)
	if err != nil {
		t.logger.Errorf("failed to unmarshal longrunning operation: %v", err)
		return
	}

	// Update proto message
	longrunningOp.Done = true
	longrunningOp.Result = &longrunning.Operation_Error{
		Error: &spb.Status{
			Code:    int32(codes.Internal),
			Message: fmt.Sprintf("the operation failed: %s", errMsg),
		},
	}

	// Marshal to bytes
	bs, err := json.Marshal(longrunningOp)
	if err != nil {
		t.logger.Errorf("failed to marshal longrunning operation: %v", err)
		return
	}

	operationDB.Payload = bs
	operationDB.Done = true

	// Save back to cache
	err = t.sqlDB.Table(services.ContactTracingOperationTable).Where("id=?", longrunningID).
		Updates(operationDB).Error
	if err != nil {
		t.logger.Errorf("failed to update longrunning operation: %v", err)
		return
	}
}

func (t *tracingAPIServer) TraceUsersLocations(
	ctx context.Context, traceReq *contact_tracing.TraceUsersLocationsRequest,
) (*contact_tracing.ContactTracingResponse, error) {
	// request must not be nil
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
	client, err := t.messagingClient.AlertContacts(ctx, grpc.WaitForReady(true))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create stream for sending messages: %v", err)
	}

	// Create a long running operation
	longrunningOp := &longrunning.Operation{
		Name: fmt.Sprintf("TraceUserLocations::%s", uuid.New().String()),
		Done: false,
	}

	// Marshal to bytes json
	bs, err := json.Marshal(longrunningOp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal longrunning operation: %v", err)
	}

	// Save operation
	county := strings.Join(traceReq.Counties, ", ")
	operationDB := &services.ContactTracingOperation{
		County:      county,
		Description: fmt.Sprintf("Cases from %s county", county),
		Done:        false,
		Payload:     bs,
	}
	err = t.sqlDB.Create(operationDB).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save operation: %v", err)
	}

	db := t.sqlDB.Select("status, full_name, phone_number, county")
	if len(traceReq.Counties) > 0 {
		db = db.Where("county IN(?)", traceReq.Counties)
	}

	go func() {
		condition := true
		limit := 1000
		offset := 0
		counties := []string{}

		usersDB := make([]*services.UserModel, 0, limit)

		for condition {
			err = db.Find(usersDB, "status=?", int8(location.Status_POSITIVE)).Error
			switch {
			case err == nil:
			default:
				t.failLongRunningOperation(operationDB.ID, err.Error())
				return
			}

			if len(usersDB) < limit {
				condition = false
			}

			wg := &sync.WaitGroup{}

			for _, userDB := range usersDB {
				if userDB.Traced {
					continue
				}
				// Longrunning worker
				wg.Add(1)
				go func() {
					defer wg.Done()
					t.traceUserWorker(client, operationDB.ID, counties, userDB, &sinceDate)
				}()
			}

			wg.Wait()

			offset += len(usersDB)
		}
	}()

	return &contact_tracing.ContactTracingResponse{
		OperationId: int64(operationDB.ID),
	}, nil
}

func (t *tracingAPIServer) ListOperations(
	ctx context.Context, listReq *contact_tracing.ListOperationsRequest,
) (*contact_tracing.ListOperationsResponse, error) {
	// Request must not be nil
	if listReq == nil {
		return nil, services.NilRequestError("ListOperationsRequest")
	}

	// Validation
	var err error
	switch {
	case listReq.County == "":
		err = services.MissingFieldError("county")
	}
	if err != nil {
		return nil, err
	}

	// Parse page size and page token
	pageNumber, pageSize := services.NormalizePage(listReq.PageToken, listReq.PageSize)
	offset := pageNumber*pageSize - pageSize

	operationsDB := make([]*services.ContactTracingOperation, 0)
	err = t.sqlDB.Order("created_at DESC").Offset(offset).Limit(pageSize).
		Find(&operationsDB, "county=?", listReq.County).Error
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
		County:      operationDB.County,
		Description: operationDB.Description,
		Timestamp:   operationDB.CreatedAt.Unix(),
		Payload:     &longrunning.Operation{},
	}

	if len(operationDB.Payload) > 0 {
		err := json.Unmarshal(operationDB.Payload, &operationPB.Payload)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to unmarshal longrunning json")
		}
	}

	return operationPB, nil
}

func (t *tracingAPIServer) sendContactData(contactData *messaging.ContactData) {
	select {
	case <-time.After(5 * time.Second):
	case t.contactDataChan <- contactData:
	}
}
