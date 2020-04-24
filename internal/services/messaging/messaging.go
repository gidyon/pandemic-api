package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"

	"google.golang.org/grpc/grpclog"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/jinzhu/gorm"

	"github.com/appleboy/go-fcm"
	"github.com/gidyon/pandemic-api/internal/services"
	"github.com/gidyon/pandemic-api/pkg/api/location"
)

type messagingServer struct {
	failedSend chan *fcmErrFDetails
	sqlDB      *gorm.DB
	fcmClient  fcmClient
	logger     grpclog.LoggerV2
}

// Options contains options passed while calling NewMessagingServer
type Options struct {
	SQLDB     *gorm.DB
	FCMClient fcmClient
	Logger    grpclog.LoggerV2
}

type fcmErrFDetails struct {
	payload *location.ContactData
	err     error
	sending bool
}

// NewMessagingServer creates a new fcm MessagingServer server
func NewMessagingServer(
	ctx context.Context, opt *Options,
) (location.MessagingServer, error) {
	// Validation
	var err error
	switch {
	case opt.SQLDB == nil:
		err = errors.New("active sqlDB is required")
	case opt.FCMClient == nil:
		err = errors.New("fcm client is required")
	case opt.Logger == nil:
		err = errors.New("logger is required")
	}
	if err != nil {
		return nil, err
	}

	ms := &messagingServer{
		failedSend: make(chan *fcmErrFDetails, 0),
		sqlDB:      opt.SQLDB,
		fcmClient:  opt.FCMClient,
		logger:     opt.Logger,
	}

	// Auto migration
	err = ms.sqlDB.AutoMigrate(&services.GeneralMessageData{}, &services.UserModel{}, &services.ContactMessageData{}).Error
	if err != nil {
		return nil, fmt.Errorf("failed to automigrate: %v", err)
	}

	return ms, nil
}

func (s *messagingServer) AlertContacts(
	msgStream location.Messaging_AlertContactsServer,
) error {
	// Request must not be nil
	if msgStream == nil {
		return services.NilRequestError("Messaging_DispatchContactMessageServer")
	}

msgStreamLoop:
	for {
		contactData, err := msgStream.Recv()
		switch {
		case err == nil:
		case errors.Is(err, io.EOF):
			break msgStreamLoop
		default:
			continue
		}

		// Send message to device
		go s.alertContact(contactData)
	}

	return msgStream.SendAndClose(&empty.Empty{})
}

func (s *messagingServer) alertContact(
	contactData *location.ContactData,
) {
	dataPayloadDB := &services.ContactMessageData{
		UserPhone:    contactData.UserPhone,
		PatientPhone: contactData.PatientPhone,
		Message: fmt.Sprintf(
			"Hello %s, you have been in contact %d times with a person who has now tested positive for COVID-19",
			contactData.FullName,
			contactData.Count,
		),
		ContactsCount: contactData.Count,
		Sent:          true,
	}

	// Start a transaction
	tx := s.sqlDB.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
		}
	}()
	err := tx.Error
	if err != nil {
		s.sendError(contactData, err, false)
		return
	}

	// Save data to database
	err = tx.Create(dataPayloadDB).Error
	if err != nil {
		tx.Rollback()
		s.sendError(contactData, err, false)
		return
	}

	// Send message and notification
	_, err = s.fcmClient.SendWithRetry(&fcm.Message{
		To:   contactData.DeviceToken,
		Data: map[string]interface{}{},
		Notification: &fcm.Notification{
			Title: "COVID-19 Alert",
			Body:  dataPayloadDB.Message,
		},
	}, 5)
	if err != nil {
		tx.Rollback()
		s.sendError(contactData, err, true)
		return
	}

	// Commit transaction
	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		s.sendError(contactData, err, false)
		return
	}
}

func (s *messagingServer) BroadCastMessage(
	ctx context.Context, req *location.BroadCastMessageRequest,
) (*location.BroadCastMessageResponse, error) {
	// Request must not be nil
	if req == nil {
		return nil, services.NilRequestError("BroadCastMessageRequest")
	}

	// Validation
	var err error
	switch {
	case req.Title == "":
		err = services.MissingFieldError("title")
	case req.Message == "":
		err = services.MissingFieldError("message")
	case req.Payload == nil:
		err = services.MissingFieldError("payload")
	case len(req.Topics) == 0:
		err = services.MissingFieldError("topics")
	}
	if err != nil {
		return nil, err
	}

	// Broadcast message id
	messageID := uuid.New().String()

	go s.broadCastMessage(req, messageID)

	return &location.BroadCastMessageResponse{
		BroadcastMessageId: messageID,
	}, nil
}

func (s *messagingServer) broadCastMessage(
	req *location.BroadCastMessageRequest, messageID string,
) {
	topics := req.GetTopics()

	// Start transaction
	tx := s.sqlDB.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
		}
	}()
	err := tx.Error
	if err != nil {
		s.logger.Errorf("failed to start broadcast transaction: %v", err)
		return
	}

	for _, filter := range req.Filters {
		switch filter {
		case location.BroadCastMessageFilter_ALL:
		case location.BroadCastMessageFilter_POSITIVES:
			tx = tx.Where("status = ?", location.Status_POSITIVE)
		case location.BroadCastMessageFilter_NEGATIVES:
			tx = tx.Where("status = ?", location.Status_NEGATIVE)
		case location.BroadCastMessageFilter_BY_COUNTY:
			tx = tx.Where("county IN(?)", topics)
		}
	}

	// Message payload
	bs, err := json.Marshal(req.Payload)
	if err != nil {
		tx.Rollback()
		s.logger.Errorf("failed to marshal payload: %v", err)
		return
	}

	// FCM payload
	payload := map[string]interface{}{}
	for key, value := range req.Payload {
		payload[key] = value
	}

	condition := true
	limit := 1000
	offset := 0

	usersDB := make([]*services.UserModel, 0)

	for condition {
		deviceTokens := make([]string, 0, limit)

		// Get device tokens
		err = tx.Offset(offset).Limit(limit).Select("device_token, phone_number").Find(&usersDB).Error
		if err != nil {
			tx.Rollback()
			s.logger.Errorf("failed to get device token: %v", err)
			return
		}

		if len(usersDB) < limit {
			condition = false
		}

		for _, userDB := range usersDB {
			deviceTokens = append(deviceTokens, userDB.DeviceToken)

			// Save user message
			userMsg := &services.GeneralMessageData{
				MessageID: messageID,
				UserPhone: userDB.PhoneNumber,
				Title:     req.Title,
				Data:      bs,
				Sent:      true,
			}

			err = tx.Create(userMsg).Error
			if err != nil {
				s.logger.Errorf("failed to save user broadcast message: %v", err)
				continue
			}
		}

		// Send message to devices
		_, err = s.fcmClient.SendWithRetry(&fcm.Message{
			RegistrationIDs: deviceTokens,
			Data:            payload,
			Notification: &fcm.Notification{
				Title: req.Title,
				Body:  req.Message,
			},
		}, 5)
		if err != nil {
			tx.Rollback()
			s.logger.Errorf("failed to send users message and notifications: %v", err)
			return
		}

		// Commit the transaction
		err = tx.Commit().Error
		if err != nil {
			tx.Rollback()
			return
		}
	}
}

func (s *messagingServer) SendMessage(
	ctx context.Context, req *location.SendMessageRequest,
) (*location.SendMessageResponse, error) {
	// Request must not be nil
	if req == nil {
		return nil, services.NilRequestError("SendMessageRequest")
	}

	// Validation
	var err error
	switch {
	case req.UserPhone == "":
		err = services.MissingFieldError("user phone")
	case req.Title == "":
		err = services.MissingFieldError("title")
	case req.Message == "":
		err = services.MissingFieldError("message")
	case req.Payload == nil:
		err = services.MissingFieldError("payload")
	}
	if err != nil {
		return nil, err
	}

	// Get user device token
	userDB := &services.UserModel{}
	err = s.sqlDB.Table(services.UsersTable).First(userDB, "phone_number=?", req.UserPhone).Error
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, status.Errorf(codes.NotFound, "user with phone %s not found", req.UserPhone)
	default:
		return nil, status.Errorf(codes.NotFound, "error happened: %v", err)
	}

	// Message payload
	bs, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal payload: %v", err)
	}

	bid := uuid.New().String()

	// Start transaction
	tx := s.sqlDB.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
		}
	}()
	err = tx.Error
	if err != nil {
		return nil, services.FailedToBeginTx(err)
	}

	// Save user message
	userMsg := &services.GeneralMessageData{
		MessageID: bid,
		UserPhone: userDB.PhoneNumber,
		Title:     req.Title,
		Data:      bs,
		Sent:      false,
	}
	err = tx.Create(userMsg).Error
	if err != nil {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "failed to save user broadcast message: %v", err)
	}

	paylod := map[string]interface{}{}
	for key, value := range req.Payload {
		paylod[key] = value
	}

	// Send notification and message
	_, err = s.fcmClient.SendWithRetry(&fcm.Message{
		To:   userDB.DeviceToken,
		Data: paylod,
		Notification: &fcm.Notification{
			Title: req.Title,
			Body:  req.Message,
		},
	}, 5)
	if err != nil {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "failed to send user message and notification: %v", err)
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return nil, services.FailedToCommitTx(err)
	}

	return &location.SendMessageResponse{
		MessageId: bid,
	}, nil
}

func (s *messagingServer) sendError(payload *location.ContactData, err error, sending bool) {
	select {
	case <-time.After(10 * time.Second):
	case s.failedSend <- &fcmErrFDetails{
		payload: payload,
		err:     err,
		sending: sending,
	}:
	}
}
