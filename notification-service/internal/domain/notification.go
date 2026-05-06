package domain

import (
	"context"

	"ap2_assignment/shared/events"
)

type ProcessedEventStore interface {
	TryMarkProcessed(ctx context.Context, eventID string, orderID string) (bool, error)
}

type NotificationUseCase interface {
	HandlePaymentCompleted(ctx context.Context, event events.PaymentCompletedEvent) (bool, error)
}
