package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"notification-service/internal/domain"
	"notification-service/internal/idempotency"
)

var ErrInvalidEvent = errors.New("invalid payment completed event")

type NotificationUsecase struct {
	store          idempotency.Store
	provider       domain.NotificationProvider
	retryCount     int
	retryBaseDelay time.Duration
}

func NewNotificationUsecase(store idempotency.Store, provider domain.NotificationProvider, retryCount int, retryBaseDelay time.Duration) *NotificationUsecase {
	if retryCount <= 0 {
		retryCount = 1
	}
	if retryBaseDelay <= 0 {
		retryBaseDelay = 2 * time.Second
	}

	return &NotificationUsecase{
		store:          store,
		provider:       provider,
		retryCount:     retryCount,
		retryBaseDelay: retryBaseDelay,
	}
}

func (u *NotificationUsecase) HandlePaymentCompleted(ctx context.Context, event domain.PaymentCompletedEvent) error {
	if event.EventID == "" || event.PaymentID == "" || event.OrderID == "" || event.CustomerEmail == "" {
		return ErrInvalidEvent
	}
	if u.provider == nil {
		return errors.New("notification provider is not configured")
	}

	processed, err := u.store.AlreadyProcessed(ctx, event.PaymentID)
	if err != nil {
		return fmt.Errorf("check notification idempotency for payment %s: %w", event.PaymentID, err)
	}
	if processed {
		log.Printf("[Notification] Duplicate payment %s ignored", event.PaymentID)
		return nil
	}

	if err := u.sendWithRetry(ctx, event); err != nil {
		return err
	}

	if err := u.store.MarkProcessed(ctx, event.PaymentID); err != nil {
		return fmt.Errorf("mark notification idempotency for payment %s: %w", event.PaymentID, err)
	}

	return nil
}

func (u *NotificationUsecase) sendWithRetry(ctx context.Context, event domain.PaymentCompletedEvent) error {
	var lastErr error
	for attempt := 1; attempt <= u.retryCount; attempt++ {
		err := u.provider.SendPaymentCompleted(ctx, event)
		if err == nil {
			return nil
		}

		lastErr = err
		if attempt == u.retryCount {
			break
		}

		delay := u.retryDelay(attempt)
		log.Printf(
			"[Notification] Provider failed for payment %s on attempt %d/%d: %v; retrying in %s",
			event.PaymentID,
			attempt,
			u.retryCount,
			err,
			delay,
		)

		if err := sleep(ctx, delay); err != nil {
			return err
		}
	}

	return fmt.Errorf("send notification for payment %s after %d attempts: %w", event.PaymentID, u.retryCount, lastErr)
}

func (u *NotificationUsecase) retryDelay(attempt int) time.Duration {
	multiplier := 1 << (attempt - 1)
	return u.retryBaseDelay * time.Duration(multiplier)
}

func sleep(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
