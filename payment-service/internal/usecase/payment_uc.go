package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"ap2_assignment/shared/events"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

type paymentUseCase struct {
	repo      domain.PaymentRepository
	publisher domain.EventPublisher
}

func NewPaymentUseCase(repo domain.PaymentRepository, publisher domain.EventPublisher) domain.PaymentUseCase {
	return &paymentUseCase{repo: repo, publisher: publisher}
}

func (u *paymentUseCase) ProcessPayment(ctx context.Context, orderID string, customerEmail string, amount int64) (*domain.Payment, error) {
	if strings.TrimSpace(orderID) == "" {
		return nil, errors.New("order_id is required")
	}
	if strings.TrimSpace(customerEmail) == "" {
		return nil, errors.New("customer_email is required")
	}

	existingPayment, err := u.repo.GetByOrderID(orderID)
	if err == nil && existingPayment != nil {
		if existingPayment.Status == "Authorized" {
			if err := u.publishAuthorizedPayment(ctx, existingPayment); err != nil {
				return existingPayment, err
			}
		}
		return existingPayment, nil
	}

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		CustomerEmail: customerEmail,
		TransactionID: uuid.New().String(),
		Amount:        amount,
		Status:        "Authorized",
		CreatedAt:     time.Now().UTC(),
	}

	if amount <= 0 || amount > 100000 {
		payment.Status = "Declined"
	}

	if err := u.repo.Save(payment); err != nil {
		return nil, err
	}

	if payment.Status == "Authorized" {
		if err := u.publishAuthorizedPayment(ctx, payment); err != nil {
			return payment, err
		}
	}

	return payment, nil
}

func (u *paymentUseCase) publishAuthorizedPayment(ctx context.Context, payment *domain.Payment) error {
	event := events.PaymentCompletedEvent{
		EventID:       payment.ID,
		OrderID:       payment.OrderID,
		Amount:        payment.Amount,
		CustomerEmail: payment.CustomerEmail,
		Status:        payment.Status,
		CreatedAt:     time.Now().UTC(),
	}
	return u.publisher.PublishPaymentCompleted(ctx, event)
}

func (u *paymentUseCase) GetPaymentStatus(orderID string) (*domain.Payment, error) {
	return u.repo.GetByOrderID(orderID)
}
