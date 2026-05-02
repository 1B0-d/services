package usecase

import (
	"errors"
	"fmt"
	"log"

	"notification-service/internal/domain"
	"notification-service/internal/idempotency"
)

var ErrInvalidEvent = errors.New("invalid payment completed event")

type NotificationUsecase struct {
	store     *idempotency.MemoryStore
	failEmail string
}

func NewNotificationUsecase(store *idempotency.MemoryStore, failEmail string) *NotificationUsecase {
	return &NotificationUsecase{
		store:     store,
		failEmail: failEmail,
	}
}

func (u *NotificationUsecase) HandlePaymentCompleted(event domain.PaymentCompletedEvent) error {
	if event.EventID == "" || event.OrderID == "" || event.CustomerEmail == "" {
		return ErrInvalidEvent
	}

	if u.store.AlreadyProcessed(event.EventID) {
		log.Printf("[Notification] Duplicate event %s ignored", event.EventID)
		return nil
	}

	if u.failEmail != "" && event.CustomerEmail == u.failEmail {
		return fmt.Errorf("simulated permanent notification failure for %s", event.CustomerEmail)
	}

	log.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%.2f", event.CustomerEmail, event.OrderID, float64(event.Amount)/100)
	u.store.MarkProcessed(event.EventID)

	return nil
}
