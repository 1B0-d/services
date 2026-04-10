package usecase

import (
	"errors"

	"github.com/google/uuid"

	"payment-service/internal/domain"
)

var ErrInvalidAmount = errors.New("amount must be greater than 0")
var ErrPaymentNotFound = errors.New("payment not found")
type PaymentUsecase struct {
	repo domain.PaymentRepository
}

func NewPaymentUsecase(repo domain.PaymentRepository) *PaymentUsecase {
	return &PaymentUsecase{repo: repo}
}

func (u *PaymentUsecase) CreatePayment(orderID string, amount int64) (*domain.Payment, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	status := domain.PaymentStatusAuthorized
	transactionID := uuid.NewString()

	if amount > 100000 {
		status = domain.PaymentStatusDeclined
		transactionID = ""
	}

	payment := &domain.Payment{
		ID:            uuid.NewString(),
		OrderID:       orderID,
		TransactionID: transactionID,
		Amount:        amount,
		Status:        status,
	}

	if err := u.repo.Create(payment); err != nil {
		return nil, err
	}

	return payment, nil
}

func (u *PaymentUsecase) GetPaymentByOrderID(orderID string) (*domain.Payment, error) {
	payment, err := u.repo.GetByOrderID(orderID)
	if err != nil {
		return nil, err
	}
	if payment == nil {
		return nil, ErrPaymentNotFound
	}
	return payment, nil
}
