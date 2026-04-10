package domain

type Payment struct {
	ID            string
	OrderID       string
	TransactionID string
	Amount        int64
	Status        string
}

const (
	PaymentStatusAuthorized = "Authorized"
	PaymentStatusDeclined   = "Declined"
)

type PaymentRepository interface {
	Create(payment *Payment) error
	GetByOrderID(orderID string) (*Payment, error)
}
