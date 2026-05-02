package main

import (
	"context"
	"errors"
	"log"
	"net"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"order-service/internal/pubsub"
	"order-service/internal/repository"
	transportgrpc "order-service/internal/transport/grpc"
	transporthttp "order-service/internal/transport/http"
	"order-service/internal/usecase"

	pb "github.com/1B0-d/ap-pb/order"
)

func main() {
	dbURL := getEnv("ORDER_DB_URL", "postgres://postgres:postgres@localhost:5435/orderdb?sslmode=disable")
	port := getEnv("ORDER_SERVICE_PORT", "8080")
	migrationPath := getEnv("ORDER_MIGRATION_PATH", "migrations/001_create_payments_table.sql")
	paymentServiceGRPCAddress := os.Getenv("PAYMENT_SERVICE_GRPC_ADDRESS")
	if paymentServiceGRPCAddress == "" {
		paymentServiceGRPCHost := getEnv("PAYMENT_GRPC_ADDR", "localhost")
		paymentServiceGRPCPort := getEnv("PAYMENT_GRPC_PORT", "50051")
		paymentServiceGRPCAddress = net.JoinHostPort(paymentServiceGRPCHost, paymentServiceGRPCPort)
	}

	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

	if err := repository.RunMigrations(ctx, dbpool, migrationPath); err != nil {
		log.Fatalf("failed to run order migrations: %v", err)
	}

	orderRepo := repository.NewOrderRepository(dbpool)

	paymentClient, err := repository.NewPaymentGRPCClient(paymentServiceGRPCAddress)
	if err != nil {
		log.Fatalf("failed to connect to payment grpc service: %v", err)
	}
	defer func() {
		_ = paymentClient.Close()
	}()

	notifier := pubsub.NewOrderStatusBroadcaster()
	orderUsecase := usecase.NewOrderUsecase(orderRepo, paymentClient, notifier)
	orderHandler := transporthttp.NewOrderHandler(orderUsecase)

	orderGRPCPort := getEnv("ORDER_GRPC_PORT", "50052")
	orderGRPCLis, err := net.Listen("tcp", ":"+orderGRPCPort)
	if err != nil {
		log.Fatalf("failed to listen order grpc: %v", err)
	}

	orderGRPCServer := grpc.NewServer()
	pb.RegisterOrderServiceServer(orderGRPCServer, transportgrpc.NewOrderGRPCServer(orderUsecase))

	errCh := make(chan error, 2)
	go func() {
		log.Printf("order-service grpc running on port %s", orderGRPCPort)
		if err := orderGRPCServer.Serve(orderGRPCLis); err != nil {
			if !errors.Is(err, grpc.ErrServerStopped) {
				errCh <- err
			}
		}
	}()

	router := gin.Default()
	transporthttp.RegisterOrderRoutes(router, orderHandler)

	httpServer := &stdhttp.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	log.Printf("order-service running on port %s", port)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			if !errors.Is(err, stdhttp.ErrServerClosed) {
				errCh <- err
			}
		}
	}()

	select {
	case <-appCtx.Done():
		log.Println("order-service shutdown requested")
	case err := <-errCh:
		log.Printf("order-service server error: %v", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("failed to shutdown order http server: %v", err)
	}

	stopped := make(chan struct{})
	go func() {
		orderGRPCServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		orderGRPCServer.Stop()
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
