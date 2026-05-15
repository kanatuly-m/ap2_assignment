package domain

import (
	"context"
	"time"
)

type Order struct {
	ID             string    `json:"id"`
	CustomerID     string    `json:"customer_id"`
	CustomerEmail  string    `json:"customer_email"`
	ItemName       string    `json:"item_name"`
	Amount         int64     `json:"amount"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	IdempotencyKey string    `json:"-"`
}

type OrderRepository interface {
	Save(order *Order) error
	UpdateStatus(id string, status string) error
	GetByID(id string) (*Order, error)
	GetByIdempotencyKey(key string) (*Order, error)
	GetByAmountRange(minAmount, maxAmount int64) ([]*Order, error)
}

type OrderCache interface {
	Get(ctx context.Context, id string) (*Order, bool, error)
	Set(ctx context.Context, order *Order, ttl time.Duration) error
	Delete(ctx context.Context, id string) error
}

type PaymentGateway interface {
	ProcessPayment(ctx context.Context, order *Order) (string, error)
}

type OrderUseCase interface {
	CreateOrder(ctx context.Context, customerID, customerEmail, itemName string, amount int64, idempKey string) (*Order, error)
	GetOrder(ctx context.Context, id string) (*Order, error)
	CancelOrder(ctx context.Context, id string) error
	GetOrdersByAmountRange(ctx context.Context, minAmount, maxAmount int64) ([]*Order, error)
}
