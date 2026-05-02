package repository

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"order-service/internal/domain"

	pb "github.com/1B0-d/ap-pb/payment"
)

type PaymentGRPCClient struct {
	client pb.PaymentServiceClient
	conn   *grpc.ClientConn
}

func NewPaymentGRPCClient(address string) (*PaymentGRPCClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial payment grpc service: %w", err)
	}

	return &PaymentGRPCClient{
		client: pb.NewPaymentServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *PaymentGRPCClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *PaymentGRPCClient) CreatePayment(orderID, customerEmail string, amount int64) (*domain.PaymentResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := c.client.ProcessPayment(ctx, &pb.CreatePaymentRequest{
		OrderId:       orderID,
		Amount:        amount,
		CustomerEmail: customerEmail,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Payment == nil {
		return nil, fmt.Errorf("payment grpc response missing payment")
	}

	return &domain.PaymentResult{
		Status:        grpcPaymentStatusToDomain(resp.Payment.Status),
		TransactionID: resp.Payment.TransactionId,
	}, nil
}

func grpcPaymentStatusToDomain(status pb.PaymentStatus) string {
	switch status {
	case pb.PaymentStatus_PAYMENT_STATUS_AUTHORIZED:
		return domain.PaymentStatusAuthorized
	case pb.PaymentStatus_PAYMENT_STATUS_DECLINED:
		return domain.PaymentStatusDeclined
	default:
		return ""
	}
}
