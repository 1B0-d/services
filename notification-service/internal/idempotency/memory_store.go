package idempotency

import (
	"context"
	"sync"
)

type Store interface {
	AlreadyProcessed(ctx context.Context, paymentID string) (bool, error)
	MarkProcessed(ctx context.Context, paymentID string) error
}

type MemoryStore struct {
	mu        sync.Mutex
	processed map[string]struct{}
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		processed: make(map[string]struct{}),
	}
}

func (s *MemoryStore) AlreadyProcessed(ctx context.Context, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.processed[id]
	return ok, nil
}

func (s *MemoryStore) MarkProcessed(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.processed[id] = struct{}{}
	return nil
}
