package repository

import (
	"context"
	"database/sql"
)

type PostgresProcessedEventStore struct {
	db *sql.DB
}

func NewPostgresProcessedEventStore(db *sql.DB) *PostgresProcessedEventStore {
	return &PostgresProcessedEventStore{db: db}
}

func (s *PostgresProcessedEventStore) TryMarkProcessed(ctx context.Context, eventID string, orderID string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO processed_events (event_id, order_id)
		VALUES ($1, $2)
		ON CONFLICT (event_id) DO NOTHING`, eventID, orderID)
	if err != nil {
		return false, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows == 1, nil
}
