package usecase

import (
	"context"
	"errors"
	"log"
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
	cache          domain.OrderCache
	paymentService domain.PaymentService
	publisher      domain.OrderStatusPublisher
}

func NewOrderUsecase(repo domain.OrderRepository, paymentService domain.PaymentService, publisher domain.OrderStatusPublisher, orderCache ...domain.OrderCache) *OrderUsecase {
	usecase := &OrderUsecase{
		repo:           repo,
		paymentService: paymentService,
		publisher:      publisher,
	}
	if len(orderCache) > 0 {
		usecase.cache = orderCache[0]
	}
	return usecase
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
		if updateErr := u.repo.UpdateStatus(order.ID, domain.OrderStatusPending); updateErr == nil {
			u.invalidateOrderCache(order.ID)
		}
		return order, ErrPaymentServiceUnavailable
	}

	if paymentResult.Status == domain.PaymentStatusAuthorized {
		if err := u.repo.UpdateStatus(order.ID, domain.OrderStatusPaid); err != nil {
			return nil, err
		}
		u.invalidateOrderCache(order.ID)
		order.Status = domain.OrderStatusPaid
		u.notifyOrderUpdate(order)
		return order, nil
	}

	if err := u.repo.UpdateStatus(order.ID, domain.OrderStatusFailed); err != nil {
		return nil, err
	}
	u.invalidateOrderCache(order.ID)
	order.Status = domain.OrderStatusFailed
	u.notifyOrderUpdate(order)

	return order, nil
}

func (u *OrderUsecase) GetOrderByID(id string) (*domain.Order, error) {
	if u.cache != nil {
		order, ok, err := u.cache.GetByID(id)
		if err != nil {
			log.Printf("order cache read failed for %s: %v", id, err)
		}
		if ok {
			return order, nil
		}
	}

	order, err := u.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}

	u.cacheOrder(order)
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

	u.invalidateOrderCache(id)
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

func (u *OrderUsecase) cacheOrder(order *domain.Order) {
	if u.cache == nil {
		return
	}
	if err := u.cache.Set(order); err != nil {
		log.Printf("order cache write failed for %s: %v", order.ID, err)
	}
}

func (u *OrderUsecase) invalidateOrderCache(id string) {
	if u.cache == nil {
		return
	}
	if err := u.cache.DeleteByID(id); err != nil {
		log.Printf("order cache invalidation failed for %s: %v", id, err)
	}
}
