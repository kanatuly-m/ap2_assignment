package domain

type Payment struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"` // Authorized, Declined
}

type PaymentRepository interface {
	Save(payment *Payment) error
	GetByOrderID(orderID string) (*Payment, error)
	ListByStatus(status string) ([]*Payment, error)
}

type PaymentUseCase interface {
	ProcessPayment(orderID string, amount int64) (*Payment, error)
	GetPaymentStatus(orderID string) (*Payment, error)
	ListPayments(status string) ([]*Payment, error)
}
