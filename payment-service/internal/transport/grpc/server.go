package grpcserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"payment-service/internal/domain"
	"payment-service/internal/usecase"

	pb "github.com/1B0-d/ap-pb/payment"
)

type PaymentGRPCServer struct {
	pb.UnimplementedPaymentServiceServer
	usecase *usecase.PaymentUsecase
}

func NewPaymentGRPCServer(usecase *usecase.PaymentUsecase) *PaymentGRPCServer {
	return &PaymentGRPCServer{usecase: usecase}
}

func (s *PaymentGRPCServer) ProcessPayment(ctx context.Context, req *pb.CreatePaymentRequest) (*pb.CreatePaymentResponse, error) {
	payment, err := s.usecase.CreatePayment(req.OrderId, req.CustomerEmail, req.Amount)
	if err != nil {
		if err == usecase.ErrInvalidAmount {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, "failed to process payment")
	}

	return &pb.CreatePaymentResponse{Payment: toPaymentProto(payment)}, nil
}

func (s *PaymentGRPCServer) GetPaymentByOrderID(ctx context.Context, req *pb.GetPaymentRequest) (*pb.GetPaymentResponse, error) {
	payment, err := s.usecase.GetPaymentByOrderID(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get payment")
	}
	if payment == nil {
		return nil, status.Error(codes.NotFound, usecase.ErrPaymentNotFound.Error())
	}

	return &pb.GetPaymentResponse{Payment: toPaymentProto(payment)}, nil
}

func (s *PaymentGRPCServer) ListPayments(ctx context.Context, req *pb.ListPaymentsRequest) (*pb.ListPaymentsResponse, error) {
	payments, err := s.usecase.ListPayments(req.MinAmount, req.MaxAmount)
	if err != nil {
		if err == usecase.ErrInvalidAmountRange {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, "failed to list payments")
	}

	return &pb.ListPaymentsResponse{Payments: toPaymentProtoList(payments)}, nil
}

func toPaymentProto(payment *domain.Payment) *pb.Payment {
	if payment == nil {
		return nil
	}

	return &pb.Payment{
		Id:            payment.ID,
		OrderId:       payment.OrderID,
		TransactionId: payment.TransactionID,
		Amount:        payment.Amount,
		Status:        toPaymentStatusProto(payment.Status),
		CustomerEmail: payment.CustomerEmail,
	}
}

func toPaymentProtoList(payments []*domain.Payment) []*pb.Payment {
	if len(payments) == 0 {
		return []*pb.Payment{}
	}

	result := make([]*pb.Payment, 0, len(payments))
	for _, payment := range payments {
		result = append(result, toPaymentProto(payment))
	}

	return result
}

func toPaymentStatusProto(status string) pb.PaymentStatus {
	switch status {
	case domain.PaymentStatusAuthorized:
		return pb.PaymentStatus_PAYMENT_STATUS_AUTHORIZED
	case domain.PaymentStatusDeclined:
		return pb.PaymentStatus_PAYMENT_STATUS_DECLINED
	default:
		return pb.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}
