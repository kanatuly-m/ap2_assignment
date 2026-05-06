package events

import (
	"fmt"
	"time"
)

type PaymentCompletedEvent struct {
	EventID       string    `json:"event_id"`
	OrderID       string    `json:"order_id"`
	Amount        int64     `json:"amount"`
	CustomerEmail string    `json:"customer_email"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

func (e PaymentCompletedEvent) AmountAsDollars() string {
	return fmt.Sprintf("%.2f", float64(e.Amount)/100.0)
}
