package repository

import (
	"context"
	"time"

	"notification-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

type RedisNotificationJobStore struct {
	client *redis.Client
}

func NewRedisNotificationJobStore(client *redis.Client) domain.NotificationJobStore {
	return &RedisNotificationJobStore{client: client}
}

func (s *RedisNotificationJobStore) GetStatus(ctx context.Context, paymentID string) (string, error) {
	value, err := s.client.Get(ctx, s.key(paymentID)).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *RedisNotificationJobStore) SetStatus(ctx context.Context, paymentID string, status string, ttl time.Duration) error {
	return s.client.Set(ctx, s.key(paymentID), status, ttl).Err()
}

func (s *RedisNotificationJobStore) key(paymentID string) string {
	return "notification:payment:" + paymentID
}
