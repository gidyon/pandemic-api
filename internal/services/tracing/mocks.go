package tracing

import (
	"github.com/gidyon/pandemic-api/pkg/api/messaging"
)

// MessagingClientMock creates a mock for messaging API
type MessagingClientMock interface {
	messaging.MessagingClient
}

// AlertContactsStream is mock for grpc client stream
type AlertContactsStream interface {
	messaging.Messaging_AlertContactsClient
}
