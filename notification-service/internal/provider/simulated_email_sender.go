package provider

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"notification-service/internal/domain"
)

type SimulatedEmailSender struct {
	latency     time.Duration
	failureRate float64
	rand        *rand.Rand
	mu          sync.Mutex
}

func NewSimulatedEmailSender(latency time.Duration, failureRate float64) *SimulatedEmailSender {
	if failureRate < 0 {
		failureRate = 0
	}
	if failureRate > 1 {
		failureRate = 1
	}

	return &SimulatedEmailSender{
		latency:     latency,
		failureRate: failureRate,
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *SimulatedEmailSender) SendPaymentCompleted(ctx context.Context, event domain.PaymentCompletedEvent) error {
	if s.latency > 0 {
		timer := time.NewTimer(s.latency)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	if s.shouldFail() {
		return fmt.Errorf("simulated provider failure for payment %s", event.PaymentID)
	}

	log.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%.2f", event.CustomerEmail, event.OrderID, float64(event.Amount)/100)
	return nil
}

func (s *SimulatedEmailSender) shouldFail() bool {
	if s.failureRate == 0 {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.rand.Float64() < s.failureRate
}
