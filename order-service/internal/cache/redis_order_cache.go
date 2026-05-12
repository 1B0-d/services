package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"order-service/internal/domain"
)

const orderCacheTimeout = 2 * time.Second

type RedisOrderCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisOrderCache(client *redis.Client, ttl time.Duration) *RedisOrderCache {
	return &RedisOrderCache{
		client: client,
		ttl:    ttl,
	}
}

func (c *RedisOrderCache) GetByID(id string) (*domain.Order, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), orderCacheTimeout)
	defer cancel()

	body, err := c.client.Get(ctx, c.key(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var order domain.Order
	if err := json.Unmarshal(body, &order); err != nil {
		_ = c.DeleteByID(id)
		return nil, false, fmt.Errorf("decode cached order: %w", err)
	}

	return &order, true, nil
}

func (c *RedisOrderCache) Set(order *domain.Order) error {
	body, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("encode order cache value: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), orderCacheTimeout)
	defer cancel()

	return c.client.Set(ctx, c.key(order.ID), body, c.ttl).Err()
}

func (c *RedisOrderCache) DeleteByID(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), orderCacheTimeout)
	defer cancel()

	return c.client.Del(ctx, c.key(id)).Err()
}

func (c *RedisOrderCache) key(id string) string {
	return "order:" + id
}
