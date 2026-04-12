package repository

import (
	"database/sql"
	"errors"
	"payment-service/internal/domain"
)

type postgresPaymentRepo struct {
	db *sql.DB
}

func NewPostgresPaymentRepository(db *sql.DB) domain.PaymentRepository {
	return &postgresPaymentRepo{db: db}
}

func (r *postgresPaymentRepo) Save(payment *domain.Payment) error {
	query := `INSERT INTO payments (id, order_id, transaction_id, amount, status) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.Exec(query, payment.ID, payment.OrderID, payment.TransactionID, payment.Amount, payment.Status)
	return err
}

func (r *postgresPaymentRepo) GetByOrderID(orderID string) (*domain.Payment, error) {
	payment := &domain.Payment{}
	err := r.db.QueryRow(`SELECT id, order_id, transaction_id, amount, status FROM payments WHERE order_id = $1`, orderID).
		Scan(&payment.ID, &payment.OrderID, &payment.TransactionID, &payment.Amount, &payment.Status)
	if err == sql.ErrNoRows {
		return nil, errors.New("payment not found")
	}
	return payment, err
}
