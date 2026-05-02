package usecase

import (
	"context"
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
	publisher      domain.OrderStatusPublisher
}

func NewOrderUsecase(repo domain.OrderRepository, paymentService domain.PaymentService, publisher domain.OrderStatusPublisher) *OrderUsecase {
	return &OrderUsecase{
		repo:           repo,
		paymentService: paymentService,
		publisher:      publisher,
	}
}

func (u *OrderUsecase) CreateOrder(customerID, customerEmail, itemName string, amount int64) (*domain.Order, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	order := &domain.Order{
		ID:            uuid.NewString(),
		CustomerID:    customerID,
		CustomerEmail: customerEmail,
		ItemName:      itemName,
		Amount:        amount,
		Status:        domain.OrderStatusPending,
		CreatedAt:     time.Now().UTC(),
	}

	if err := u.repo.Create(order); err != nil {
		return nil, err
	}
	u.notifyOrderUpdate(order)

	paymentResult, err := u.paymentService.CreatePayment(order.ID, order.CustomerEmail, order.Amount)
	if err != nil {
		_ = u.repo.UpdateStatus(order.ID, domain.OrderStatusPending)
		return order, ErrPaymentServiceUnavailable
	}

	if paymentResult.Status == domain.PaymentStatusAuthorized {
		if err := u.repo.UpdateStatus(order.ID, domain.OrderStatusPaid); err != nil {
			return nil, err
		}
		order.Status = domain.OrderStatusPaid
		u.notifyOrderUpdate(order)
		return order, nil
	}

	if err := u.repo.UpdateStatus(order.ID, domain.OrderStatusFailed); err != nil {
		return nil, err
	}
	order.Status = domain.OrderStatusFailed
	u.notifyOrderUpdate(order)

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
	u.notifyOrderUpdate(order)
	return order, nil
}

func (u *OrderUsecase) SubscribeToOrderUpdates(orderID string, ctx context.Context) (<-chan *domain.Order, error) {
	if u.publisher == nil {
		return nil, errors.New("order updates publisher not configured")
	}
	return u.publisher.Subscribe(orderID, ctx)
}

func (u *OrderUsecase) notifyOrderUpdate(order *domain.Order) {
	if u.publisher == nil {
		return
	}
	_ = u.publisher.Publish(order)
}
