package domain

import (
	"context"
	"time"

	"ap2_assignment/shared/events"
)

type Payment struct {
	ID            string    `json:"id"`
	OrderID       string    `json:"order_id"`
	CustomerEmail string    `json:"customer_email"`
	TransactionID string    `json:"transaction_id"`
	Amount        int64     `json:"amount"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type PaymentRepository interface {
	Save(payment *Payment) error
	GetByOrderID(orderID string) (*Payment, error)
}

type EventPublisher interface {
	PublishPaymentCompleted(ctx context.Context, event events.PaymentCompletedEvent) error
	Close() error
}

type PaymentUseCase interface {
	ProcessPayment(ctx context.Context, orderID string, customerEmail string, amount int64) (*Payment, error)
	GetPaymentStatus(orderID string) (*Payment, error)
}
