package domain

import (
	"context"
	"time"

	"ap2_assignment/shared/events"
)

type NotificationJobStore interface {
	GetStatus(ctx context.Context, paymentID string) (string, error)
	SetStatus(ctx context.Context, paymentID string, status string, ttl time.Duration) error
}

type EmailSender interface {
	Send(ctx context.Context, event events.PaymentCompletedEvent) error
}

type NotificationUseCase interface {
	HandlePaymentCompleted(ctx context.Context, event events.PaymentCompletedEvent) (bool, error)
}
