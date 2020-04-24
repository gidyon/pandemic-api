package tracing

import (
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"google.golang.org/grpc"
)

// MessagingClientMock creates a mock for messaging API
type MessagingClientMock interface {
	location.MessagingClient
}

// AlertContactsStream is mock for grpc client stream
type AlertContactsStream interface {
	location.Messaging_AlertContactsClient
	grpc.ClientStream
}
