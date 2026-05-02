package usecase

import (
	"testing"

	"payment-service/internal/domain"
)

type stubPaymentRepository struct {
	findMin  int64
	findMax  int64
	findResp []*domain.Payment
	findErr  error
}

func (s *stubPaymentRepository) Create(payment *domain.Payment) error {
	return nil
}

func (s *stubPaymentRepository) GetByOrderID(orderID string) (*domain.Payment, error) {
	return nil, nil
}

func (s *stubPaymentRepository) FindByAmountRange(minAmount, maxAmount int64) ([]*domain.Payment, error) {
	s.findMin = minAmount
	s.findMax = maxAmount
	return s.findResp, s.findErr
}

func TestListPaymentsRejectsInvalidRange(t *testing.T) {
	repo := &stubPaymentRepository{}
	usecase := NewPaymentUsecase(repo)

	_, err := usecase.ListPayments(5000, 1000)
	if err != ErrInvalidAmountRange {
		t.Fatalf("expected ErrInvalidAmountRange, got %v", err)
	}
}

func TestListPaymentsUsesRepositoryRange(t *testing.T) {
	repo := &stubPaymentRepository{
		findResp: []*domain.Payment{
			{ID: "p-1", OrderID: "o-1", Amount: 1500, Status: domain.PaymentStatusAuthorized},
		},
	}
	usecase := NewPaymentUsecase(repo)

	payments, err := usecase.ListPayments(1000, 2000)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.findMin != 1000 || repo.findMax != 2000 {
		t.Fatalf("expected repository call with 1000..2000, got %d..%d", repo.findMin, repo.findMax)
	}
	if len(payments) != 1 || payments[0].ID != "p-1" {
		t.Fatalf("unexpected payments result: %#v", payments)
	}
}
