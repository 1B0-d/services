package pubsub

import (
	"context"
	"sync"

	"order-service/internal/domain"
)

type OrderStatusBroadcaster struct {
	mu          sync.Mutex
	subscribers map[string]map[chan *domain.Order]struct{}
}

func NewOrderStatusBroadcaster() *OrderStatusBroadcaster {
	return &OrderStatusBroadcaster{
		subscribers: make(map[string]map[chan *domain.Order]struct{}),
	}
}

func (b *OrderStatusBroadcaster) Publish(order *domain.Order) error {
	b.mu.Lock()
	subscribers := b.subscribers[order.ID]
	channels := make([]chan *domain.Order, 0, len(subscribers))
	for ch := range subscribers {
		channels = append(channels, ch)
	}
	b.mu.Unlock()

	for _, ch := range channels {
		select {
		case ch <- order:
		default:
		}
	}

	return nil
}

func (b *OrderStatusBroadcaster) Subscribe(orderID string, ctx context.Context) (<-chan *domain.Order, error) {
	ch := make(chan *domain.Order, 10)

	b.mu.Lock()
	if b.subscribers[orderID] == nil {
		b.subscribers[orderID] = make(map[chan *domain.Order]struct{})
	}
	b.subscribers[orderID][ch] = struct{}{}
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		delete(b.subscribers[orderID], ch)
		if len(b.subscribers[orderID]) == 0 {
			delete(b.subscribers, orderID)
		}
		b.mu.Unlock()
		close(ch)
	}()

	return ch, nil
}
