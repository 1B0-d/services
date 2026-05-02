package domain

import (
	"context"
	"time"
)

type Order struct {
	ID            string
	CustomerID    string
	CustomerEmail string
	ItemName      string
	Amount        int64
	Status        string
	CreatedAt     time.Time
}

const (
	OrderStatusPending   = "Pending"
	OrderStatusPaid      = "Paid"
	OrderStatusFailed    = "Failed"
	OrderStatusCancelled = "Cancelled"

	PaymentStatusAuthorized = "Authorized"
	PaymentStatusDeclined   = "Declined"
)

type OrderRepository interface {
	Create(order *Order) error
	GetByID(id string) (*Order, error)
	UpdateStatus(id string, status string) error
	GetByCustomerID(customerID string) ([]*Order, error)
}

type PaymentResult struct {
	Status        string
	TransactionID string
}

type PaymentService interface {
	CreatePayment(orderID, customerEmail string, amount int64) (*PaymentResult, error)
}

type OrderStatusPublisher interface {
	Publish(order *Order) error
	Subscribe(orderID string, ctx context.Context) (<-chan *Order, error)
}
