package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"order-service/internal/repository"
	transportgrpc "order-service/internal/transport/grpc"
	transporthttp "order-service/internal/transport/http"
	"order-service/internal/usecase"

	pb "github.com/1B0-d/ap-pb/order"
)

func main() {
	dbURL := getEnv("ORDER_DB_URL", "postgres://postgres:postgres@localhost:5435/orderdb?sslmode=disable")
	port := getEnv("ORDER_SERVICE_PORT", "8080")
	paymentServiceGRPCAddress := getEnv("PAYMENT_SERVICE_GRPC_ADDRESS", "localhost:50051")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbpool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to create db pool: %v", err)
	}
	defer dbpool.Close()

	if err := dbpool.Ping(ctx); err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	orderRepo := repository.NewOrderRepository(dbpool)

	paymentClient, err := repository.NewPaymentGRPCClient(paymentServiceGRPCAddress)
	if err != nil {
		log.Fatalf("failed to connect to payment grpc service: %v", err)
	}
	defer func() {
		_ = paymentClient.Close()
	}()

	orderUsecase := usecase.NewOrderUsecase(orderRepo, paymentClient)
	orderHandler := transporthttp.NewOrderHandler(orderUsecase)

	orderGRPCPort := getEnv("ORDER_GRPC_PORT", "50052")
	orderGRPCLis, err := net.Listen("tcp", ":"+orderGRPCPort)
	if err != nil {
		log.Fatalf("failed to listen order grpc: %v", err)
	}

	orderGRPCServer := grpc.NewServer()
	pb.RegisterOrderServiceServer(orderGRPCServer, transportgrpc.NewOrderGRPCServer(orderUsecase))
	go func() {
		log.Printf("order-service grpc running on port %s", orderGRPCPort)
		if err := orderGRPCServer.Serve(orderGRPCLis); err != nil {
			log.Fatalf("failed to run order grpc server: %v", err)
		}
	}()

	router := gin.Default()
	transporthttp.RegisterOrderRoutes(router, orderHandler)

	log.Printf("order-service running on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
