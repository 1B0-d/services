package idempotency

import "sync"

type MemoryStore struct {
	mu        sync.Mutex
	processed map[string]struct{}
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		processed: make(map[string]struct{}),
	}
}

func (s *MemoryStore) AlreadyProcessed(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.processed[id]
	return ok
}

func (s *MemoryStore) MarkProcessed(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.processed[id] = struct{}{}
}
