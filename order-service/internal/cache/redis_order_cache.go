package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"order-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

type RedisOrderCache struct {
	client *redis.Client
}

func NewRedisOrderCache(client *redis.Client) domain.OrderCache {
	return &RedisOrderCache{client: client}
}

func (c *RedisOrderCache) Get(ctx context.Context, id string) (*domain.Order, bool, error) {
	value, err := c.client.Get(ctx, c.key(id)).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var order domain.Order
	if err := json.Unmarshal([]byte(value), &order); err != nil {
		_ = c.client.Del(ctx, c.key(id)).Err()
		return nil, false, fmt.Errorf("invalid cached order JSON: %w", err)
	}
	return &order, true, nil
}

func (c *RedisOrderCache) Set(ctx context.Context, order *domain.Order, ttl time.Duration) error {
	body, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(order.ID), body, ttl).Err()
}

func (c *RedisOrderCache) Delete(ctx context.Context, id string) error {
	return c.client.Del(ctx, c.key(id)).Err()
}

func (c *RedisOrderCache) key(id string) string {
	return "order:" + id
}
