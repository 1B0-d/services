package domain

type Payment struct {
	ID            string
	OrderID       string
	TransactionID string
	Amount        int64
	Status        string
	CustomerEmail string
}

const (
	PaymentStatusAuthorized = "Authorized"
	PaymentStatusDeclined   = "Declined"
)

type PaymentRepository interface {
	Create(payment *Payment) error
	GetByOrderID(orderID string) (*Payment, error)
	FindByAmountRange(minAmount, maxAmount int64) ([]*Payment, error)
}

type PaymentCompletedEvent struct {
	EventID       string `json:"event_id"`
	PaymentID     string `json:"payment_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type PaymentEventPublisher interface {
	PublishPaymentCompleted(event PaymentCompletedEvent) error
	Close() error
}
