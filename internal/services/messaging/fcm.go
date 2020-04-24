package messaging

import (
	"context"

	"github.com/appleboy/go-fcm"
)

type fcmClient interface {
	SendWithContext(ctx context.Context, msg *fcm.Message) (*fcm.Response, error)
	SendWithRetry(msg *fcm.Message, retryAttempts int) (*fcm.Response, error)
	Send(msg *fcm.Message) (*fcm.Response, error)
}
