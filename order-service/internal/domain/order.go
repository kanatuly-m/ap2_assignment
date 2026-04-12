package domain

import "time"

type Order struct {
	ID             string    `json:"id"`
	CustomerID     string    `json:"customer_id"`
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

type OrderUseCase interface {
	CreateOrder(customerID, itemName string, amount int64, idempKey string) (*Order, error)
	GetOrder(id string) (*Order, error)
	CancelOrder(id string) error
	GetOrdersByAmountRange(minAmount, maxAmount int64) ([]*Order, error)
}
