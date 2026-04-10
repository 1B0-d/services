package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-service/internal/domain"
)

type OrderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(order *domain.Order) error {
	query := `
		INSERT INTO orders (id, customer_id, item_name, amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(
		context.Background(),
		query,
		order.ID,
		order.CustomerID,
		order.ItemName,
		order.Amount,
		order.Status,
		order.CreatedAt,
	)
	return err
}

func (r *OrderRepository) GetByID(id string) (*domain.Order, error) {
	query := `
		SELECT id, customer_id, item_name, amount, status, created_at
		FROM orders
		WHERE id = $1
		LIMIT 1
	`

	var order domain.Order
	err := r.db.QueryRow(context.Background(), query, id).Scan(
		&order.ID,
		&order.CustomerID,
		&order.ItemName,
		&order.Amount,
		&order.Status,
		&order.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &order, nil
}

func (r *OrderRepository) GetByCustomerID(customerID string) ([]*domain.Order, error) {
	query := `
		SELECT id, customer_id, item_name, amount, status, created_at
		FROM orders
		WHERE customer_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(context.Background(), query, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.Order

	for rows.Next() {
		var order domain.Order
		if err := rows.Scan(
			&order.ID,
			&order.CustomerID,
			&order.ItemName,
			&order.Amount,
			&order.Status,
			&order.CreatedAt,
		); err != nil {
			return nil, err
		}

		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *OrderRepository) UpdateStatus(id string, status string) error {
	query := `
		UPDATE orders
		SET status = $1
		WHERE id = $2
	`
	_, err := r.db.Exec(context.Background(), query, status, id)
	return err
}
