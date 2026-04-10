package usecase

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"order-service/internal/domain"
)

var ErrInvalidAmount = errors.New("amount must be greater than 0")
var ErrOrderNotFound = errors.New("order not found")
var ErrOrderCannotBeCancelled = errors.New("only pending orders can be cancelled")
var ErrPaymentServiceUnavailable = errors.New("payment service unavailable")

type OrderUsecase struct {
	repo           domain.OrderRepository
	paymentService domain.PaymentService
}

func NewOrderUsecase(repo domain.OrderRepository, paymentService domain.PaymentService) *OrderUsecase {
	return &OrderUsecase{
		repo:           repo,
		paymentService: paymentService,
	}
}

func (u *OrderUsecase) CreateOrder(customerID, itemName string, amount int64) (*domain.Order, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	order := &domain.Order{
		ID:         uuid.NewString(),
		CustomerID: customerID,
		ItemName:   itemName,
		Amount:     amount,
		Status:     domain.OrderStatusPending,
		CreatedAt:  time.Now().UTC(),
	}

	if err := u.repo.Create(order); err != nil {
		return nil, err
	}

	paymentResult, err := u.paymentService.CreatePayment(order.ID, order.Amount)
	if err != nil {
		_ = u.repo.UpdateStatus(order.ID, domain.OrderStatusPending)
		return order, ErrPaymentServiceUnavailable
	}

	if paymentResult.Status == domain.PaymentStatusAuthorized {
		if err := u.repo.UpdateStatus(order.ID, domain.OrderStatusPaid); err != nil {
			return nil, err
		}
		order.Status = domain.OrderStatusPaid
		return order, nil
	}

	if err := u.repo.UpdateStatus(order.ID, domain.OrderStatusFailed); err != nil {
		return nil, err
	}
	order.Status = domain.OrderStatusFailed

	return order, nil
}

func (u *OrderUsecase) GetOrderByID(id string) (*domain.Order, error) {
	order, err := u.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	return order, nil
}
func (u *OrderUsecase) GetOrdersByCustomerID(customerID string) ([]*domain.Order, error) {
	return u.repo.GetByCustomerID(customerID)
}
func (u *OrderUsecase) CancelOrder(id string) (*domain.Order, error) {
	order, err := u.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}

	if order.Status != domain.OrderStatusPending {
		return nil, ErrOrderCannotBeCancelled
	}

	if err := u.repo.UpdateStatus(id, domain.OrderStatusCancelled); err != nil {
		return nil, err
	}

	order.Status = domain.OrderStatusCancelled
	return order, nil
}
