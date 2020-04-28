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
	err = ms.sqlDB.AutoMigrate(&services.Message{}, &services.UserModel{}).Error
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
	messageData := map[string]interface{}{
		"patient_phone":  contactData.PatientPhone,
		"contact_points": contactData.Count,
	}

	data, err := json.Marshal(messageData)
	if err != nil {
		s.sendError(contactData, err, false)
		return
	}

	messageModel := &services.Message{
		UserPhone: contactData.UserPhone,
		Title:     "COVID-19 Alert!",
		DateTime:  time.Now().Local().String(),
		Message: fmt.Sprintf(
			"Hello %s, you have been in contact %d times with a person who has now tested positive for COVID-19",
			contactData.FullName,
			contactData.Count,
		),
		Sent: true,
		Type: int8(location.MessageType_ALERT),
		Data: data,
	}

	// Start a transaction
	tx := s.sqlDB.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
		}
	}()
	err = tx.Error
	if err != nil {
		s.sendError(contactData, err, false)
		return
	}

	// Save data to database
	err = tx.Create(messageModel).Error
	if err != nil {
		tx.Rollback()
		s.sendError(contactData, err, false)
		return
	}

	// Send message and notification
	_, err = s.fcmClient.SendWithRetry(&fcm.Message{
		To:   contactData.DeviceToken,
		Data: messageData,
		Notification: &fcm.Notification{
			Title: messageModel.Title,
			Body:  messageModel.Message,
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
			userMsg := &services.Message{
				UserPhone: userDB.PhoneNumber,
				Title:     req.Title,
				Message:   req.Message,
				DateTime:  time.Now().Local().String(),
				Data:      bs,
				Sent:      true,
				Type:      int8(req.Type),
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
	ctx context.Context, msg *location.Message,
) (*location.SendMessageResponse, error) {
	// Request must not be nil
	if msg == nil {
		return nil, services.NilRequestError("Message")
	}

	// Validation
	var err error
	switch {
	case msg.UserPhone == "":
		err = services.MissingFieldError("user phone")
	case msg.Title == "":
		err = services.MissingFieldError("title")
	case msg.Notification == "":
		err = services.MissingFieldError("notification")
	case msg.Data == nil:
		err = services.MissingFieldError("data")
	}
	if err != nil {
		return nil, err
	}

	// Get user device token
	userDB := &services.UserModel{}
	err = s.sqlDB.Table(services.UsersTable).First(userDB, "phone_number=?", msg.UserPhone).Error
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, status.Errorf(codes.NotFound, "user with phone %s not found", msg.UserPhone)
	default:
		return nil, status.Errorf(codes.NotFound, "error happened: %v", err)
	}

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
	userMsg, err := getMessageDB(msg)
	if err != nil {
		return nil, err
	}

	userMsg.Sent = true

	err = tx.Create(userMsg).Error
	if err != nil {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "failed to save user broadcast message: %v", err)
	}

	data := map[string]interface{}{}
	for key, value := range msg.Data {
		data[key] = value
	}

	// Send notification and message
	_, err = s.fcmClient.SendWithRetry(&fcm.Message{
		To:   userDB.DeviceToken,
		Data: data,
		Notification: &fcm.Notification{
			Title: msg.Title,
			Body:  msg.Notification,
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
		MessageId: fmt.Sprint(userMsg.ID),
	}, nil
}

func (s *messagingServer) GetMessages(
	ctx context.Context, getReq *location.GetMessagesRequest,
) (*location.Messages, error) {
	// Requst must not be nil
	if getReq == nil {
		return nil, services.NilRequestError("GetMessagesRequest")
	}

	// Validation
	var err error
	switch {
	case getReq.PhoneNumber == "":
		err = services.MissingFieldError("user phone")
	}
	if err != nil {
		return nil, err
	}

	// Normalize page
	pageNumber, pageSize := normalizePage(getReq.GetPageToken(), getReq.GetPageSize())
	offset := pageNumber*pageSize - pageSize

	messagesDB := make([]*services.Message, 0)
	err = s.sqlDB.Order("created_at, date_time DESC").Offset(offset).Limit(pageSize).
		Find(&messagesDB, "user_phone=?", getReq.PhoneNumber).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get messages: %v", err)
	}

	messagesPB := make([]*location.Message, 0, len(messagesDB))

	for _, messageDB := range messagesDB {
		messagePB, err := getMessagePB(messageDB)
		if err != nil {
			return nil, err
		}

		messagesPB = append(messagesPB, messagePB)
	}

	return &location.Messages{
		Messages: messagesPB,
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

func getMessageDB(messagePB *location.Message) (*services.Message, error) {
	messageDB := &services.Message{
		UserPhone: messagePB.UserPhone,
		Title:     messagePB.Title,
		Message:   messagePB.Notification,
		DateTime:  messagePB.DateTime,
		Sent:      messagePB.Sent,
		Type:      int8(messagePB.Type),
	}

	if len(messagePB.Data) != 0 {
		data, err := json.Marshal(messagePB.Data)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal: %v", err)
		}
		messageDB.Data = data
	}

	return messageDB, nil
}

func getMessagePB(messageDB *services.Message) (*location.Message, error) {
	messagePB := &location.Message{
		MessageId:    fmt.Sprint(messageDB.ID),
		UserPhone:    messageDB.UserPhone,
		Title:        messageDB.Title,
		Notification: messageDB.Message,
		DateTime:     messageDB.DateTime,
		Sent:         messageDB.Sent,
		Type:         location.MessageType(messageDB.Type),
	}

	if len(messageDB.Data) != 0 {
		data := make(map[string]string, 0)
		err := json.Unmarshal(messageDB.Data, &data)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to json unmarshal: %v", err)
		}
		messagePB.Data = data
	}

	return messagePB, nil
}
