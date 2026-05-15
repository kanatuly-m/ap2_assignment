package usecase

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"ap2_assignment/shared/events"
	"notification-service/internal/domain"
)

type notificationUseCase struct {
	store     domain.NotificationJobStore
	sender    domain.EmailSender
	statusTTL time.Duration
}

func NewNotificationUseCase(store domain.NotificationJobStore, sender domain.EmailSender, statusTTL time.Duration) domain.NotificationUseCase {
	return &notificationUseCase{store: store, sender: sender, statusTTL: statusTTL}
}

func (u *notificationUseCase) HandlePaymentCompleted(ctx context.Context, event events.PaymentCompletedEvent) (bool, error) {
	paymentID := strings.TrimSpace(event.EffectivePaymentID())
	if paymentID == "" {
		return false, errors.New("payment_id is required for notification job idempotency")
	}

	status, err := u.store.GetStatus(ctx, paymentID)
	if err != nil {
		return false, err
	}
	if status == "sent" {
		log.Printf("[Notification] Duplicate payment job skipped: payment_id=%s", paymentID)
		return false, nil
	}

	if err := u.store.SetStatus(ctx, paymentID, "processing", u.statusTTL); err != nil {
		return false, err
	}

	if err := u.sender.Send(ctx, event); err != nil {
		_ = u.store.SetStatus(ctx, paymentID, "failed", u.statusTTL)
		return false, err
	}

	if err := u.store.SetStatus(ctx, paymentID, "sent", u.statusTTL); err != nil {
		return false, err
	}

	log.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%s | payment_id=%s", event.CustomerEmail, event.OrderID, event.AmountAsDollars(), paymentID)
	return true, nil
}
