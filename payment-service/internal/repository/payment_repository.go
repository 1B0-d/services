package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"payment-service/internal/domain"
)

type PaymentRepository struct {
	db *pgxpool.Pool
}

func NewPaymentRepository(db *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(payment *domain.Payment) error {
	query := `
		INSERT INTO payments (id, order_id, transaction_id, amount, status, customer_email)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(
		context.Background(),
		query,
		payment.ID,
		payment.OrderID,
		payment.TransactionID,
		payment.Amount,
		payment.Status,
		payment.CustomerEmail,
	)
	return err
}

func (r *PaymentRepository) GetByOrderID(orderID string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, transaction_id, amount, status, customer_email
		FROM payments
		WHERE order_id = $1
		LIMIT 1
	`

	var payment domain.Payment
	err := r.db.QueryRow(context.Background(), query, orderID).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.TransactionID,
		&payment.Amount,
		&payment.Status,
		&payment.CustomerEmail,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepository) FindByAmountRange(minAmount, maxAmount int64) ([]*domain.Payment, error) {
	query := `
		SELECT id, order_id, transaction_id, amount, status, customer_email
		FROM payments
		WHERE ($1 = 0 OR amount >= $1)
		  AND ($2 = 0 OR amount <= $2)
		ORDER BY amount ASC, id ASC
	`

	rows, err := r.db.Query(context.Background(), query, minAmount, maxAmount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	payments := make([]*domain.Payment, 0)
	for rows.Next() {
		var payment domain.Payment
		if err := rows.Scan(
			&payment.ID,
			&payment.OrderID,
			&payment.TransactionID,
			&payment.Amount,
			&payment.Status,
			&payment.CustomerEmail,
		); err != nil {
			return nil, err
		}
		payments = append(payments, &payment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return payments, nil
}
