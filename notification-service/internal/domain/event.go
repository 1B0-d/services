package domain

import "context"

type PaymentCompletedEvent struct {
	EventID       string `json:"event_id"`
	PaymentID     string `json:"payment_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type NotificationProvider interface {
	SendPaymentCompleted(ctx context.Context, event PaymentCompletedEvent) error
}
