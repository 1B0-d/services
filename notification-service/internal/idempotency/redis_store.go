package idempotency

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const processedStatus = "sent"

type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisStore(client *redis.Client, ttl time.Duration) *RedisStore {
	return &RedisStore{
		client: client,
		ttl:    ttl,
	}
}

func (s *RedisStore) AlreadyProcessed(ctx context.Context, paymentID string) (bool, error) {
	status, err := s.client.Get(ctx, s.key(paymentID)).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return status == processedStatus, nil
}

func (s *RedisStore) MarkProcessed(ctx context.Context, paymentID string) error {
	return s.client.Set(ctx, s.key(paymentID), processedStatus, s.ttl).Err()
}

func (s *RedisStore) key(paymentID string) string {
	return "notification:payment:" + paymentID
}
