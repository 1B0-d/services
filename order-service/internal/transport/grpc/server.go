package grpcserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"order-service/internal/domain"
	"order-service/internal/usecase"

	pb "github.com/1B0-d/ap-pb/order"
)

type OrderGRPCServer struct {
	pb.UnimplementedOrderServiceServer
	usecase *usecase.OrderUsecase
}

func NewOrderGRPCServer(usecase *usecase.OrderUsecase) *OrderGRPCServer {
	return &OrderGRPCServer{usecase: usecase}
}

func (s *OrderGRPCServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	order, err := s.usecase.CreateOrder(req.CustomerId, req.ItemName, req.Amount)
	if err != nil {
		switch err {
		case usecase.ErrInvalidAmount:
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case usecase.ErrPaymentServiceUnavailable:
			return nil, status.Error(codes.Unavailable, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &pb.CreateOrderResponse{Order: toOrderProto(order)}, nil
}

func (s *OrderGRPCServer) GetOrderByID(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	order, err := s.usecase.GetOrderByID(req.Id)
	if err != nil {
		switch err {
		case usecase.ErrOrderNotFound:
			return nil, status.Error(codes.NotFound, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &pb.GetOrderResponse{Order: toOrderProto(order)}, nil
}

func (s *OrderGRPCServer) GetOrdersByCustomerID(ctx context.Context, req *pb.GetOrdersByCustomerRequest) (*pb.GetOrdersByCustomerResponse, error) {
	orders, err := s.usecase.GetOrdersByCustomerID(req.CustomerId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := make([]*pb.Order, 0, len(orders))
	for _, order := range orders {
		response = append(response, toOrderProto(order))
	}

	return &pb.GetOrdersByCustomerResponse{Orders: response}, nil
}

func toOrderProto(order *domain.Order) *pb.Order {
	if order == nil {
		return nil
	}

	return &pb.Order{
		Id:         order.ID,
		CustomerId: order.CustomerID,
		ItemName:   order.ItemName,
		Amount:     order.Amount,
		Status:     toOrderStatusProto(order.Status),
		CreatedAt:  timestamppb.New(order.CreatedAt),
	}
}

func toOrderStatusProto(status string) pb.OrderStatus {
	switch status {
	case domain.OrderStatusPending:
		return pb.OrderStatus_ORDER_STATUS_PENDING
	case domain.OrderStatusPaid:
		return pb.OrderStatus_ORDER_STATUS_PAID
	case domain.OrderStatusFailed:
		return pb.OrderStatus_ORDER_STATUS_FAILED
	case domain.OrderStatusCancelled:
		return pb.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return pb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}
