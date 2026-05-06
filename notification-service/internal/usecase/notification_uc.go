package usecase

import (
	"context"
	"errors"
	"log"

	"ap2_assignment/shared/events"
	"notification-service/internal/domain"
)

type notificationUseCase struct {
	store domain.ProcessedEventStore
}

func NewNotificationUseCase(store domain.ProcessedEventStore) domain.NotificationUseCase {
	return &notificationUseCase{store: store}
}

func (u *notificationUseCase) HandlePaymentCompleted(ctx context.Context, event events.PaymentCompletedEvent) (bool, error) {
	if event.CustomerEmail == "fail@example.com" {
		return false, errors.New("simulated permanent email provider failure")
	}

	inserted, err := u.store.TryMarkProcessed(ctx, event.EventID, event.OrderID)
	if err != nil {
		return false, err
	}
	if !inserted {
		log.Printf("[Notification] Duplicate event skipped: %s", event.EventID)
		return false, nil
	}

	log.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%s", event.CustomerEmail, event.OrderID, event.AmountAsDollars())
	return true, nil
}
