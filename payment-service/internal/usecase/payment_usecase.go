package usecase

import (
	"errors"

	"github.com/google/uuid"

	"payment-service/internal/domain"
)

var ErrInvalidAmount = errors.New("amount must be greater than 0")
var ErrPaymentNotFound = errors.New("payment not found")
var ErrInvalidAmountRange = errors.New("min_amount cannot be greater than max_amount")

type PaymentUsecase struct {
	repo      domain.PaymentRepository
	publisher domain.PaymentEventPublisher
}

func NewPaymentUsecase(repo domain.PaymentRepository, publisher ...domain.PaymentEventPublisher) *PaymentUsecase {
	var eventPublisher domain.PaymentEventPublisher
	if len(publisher) > 0 {
		eventPublisher = publisher[0]
	}
	return &PaymentUsecase{repo: repo, publisher: eventPublisher}
}

func (u *PaymentUsecase) CreatePayment(orderID, customerEmail string, amount int64) (*domain.Payment, error) {
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
		CustomerEmail: customerEmail,
	}

	if err := u.repo.Create(payment); err != nil {
		return nil, err
	}

	if status == domain.PaymentStatusAuthorized && u.publisher != nil {
		err := u.publisher.PublishPaymentCompleted(domain.PaymentCompletedEvent{
			EventID:       uuid.NewString(),
			PaymentID:     payment.ID,
			OrderID:       payment.OrderID,
			Amount:        payment.Amount,
			CustomerEmail: payment.CustomerEmail,
			Status:        payment.Status,
		})
		if err != nil {
			return nil, err
		}
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

func (u *PaymentUsecase) ListPayments(minAmount, maxAmount int64) ([]*domain.Payment, error) {
	if minAmount > 0 && maxAmount > 0 && minAmount > maxAmount {
		return nil, ErrInvalidAmountRange
	}

	return u.repo.FindByAmountRange(minAmount, maxAmount)
}
