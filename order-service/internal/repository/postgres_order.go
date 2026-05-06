package repository

import (
	"database/sql"
	"errors"

	"order-service/internal/domain"
)

type postgresOrderRepo struct {
	db *sql.DB
}

func NewPostgresOrderRepository(db *sql.DB) domain.OrderRepository {
	return &postgresOrderRepo{db: db}
}

func (r *postgresOrderRepo) Save(order *domain.Order) error {
	query := `
		INSERT INTO orders (id, customer_id, customer_email, item_name, amount, status, created_at, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''))`
	_, err := r.db.Exec(
		query,
		order.ID,
		order.CustomerID,
		order.CustomerEmail,
		order.ItemName,
		order.Amount,
		order.Status,
		order.CreatedAt,
		order.IdempotencyKey,
	)
	return err
}

func (r *postgresOrderRepo) UpdateStatus(id string, status string) error {
	result, err := r.db.Exec(`UPDATE orders SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("order not found")
	}
	return nil
}

func (r *postgresOrderRepo) GetByID(id string) (*domain.Order, error) {
	order := &domain.Order{}
	err := r.db.QueryRow(`
		SELECT id, customer_id, customer_email, item_name, amount, status, created_at
		FROM orders
		WHERE id = $1`, id).
		Scan(&order.ID, &order.CustomerID, &order.CustomerEmail, &order.ItemName, &order.Amount, &order.Status, &order.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("order not found")
	}
	return order, err
}

func (r *postgresOrderRepo) GetByIdempotencyKey(key string) (*domain.Order, error) {
	if key == "" {
		return nil, errors.New("empty key")
	}
	order := &domain.Order{}
	err := r.db.QueryRow(`
		SELECT id, customer_id, customer_email, item_name, amount, status, created_at
		FROM orders
		WHERE idempotency_key = $1`, key).
		Scan(&order.ID, &order.CustomerID, &order.CustomerEmail, &order.ItemName, &order.Amount, &order.Status, &order.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return order, err
}

func (r *postgresOrderRepo) GetByAmountRange(minAmount, maxAmount int64) ([]*domain.Order, error) {
	query := `
		SELECT id, customer_id, customer_email, item_name, amount, status, created_at
		FROM orders
		WHERE amount >= $1 AND amount <= $2
		ORDER BY created_at DESC`
	rows, err := r.db.Query(query, minAmount, maxAmount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]*domain.Order, 0)
	for rows.Next() {
		order := &domain.Order{}
		if err := rows.Scan(&order.ID, &order.CustomerID, &order.CustomerEmail, &order.ItemName, &order.Amount, &order.Status, &order.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}
