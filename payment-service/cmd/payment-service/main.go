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

	"payment-service/internal/messaging"
	"payment-service/internal/repository"
	transportgrpc "payment-service/internal/transport/grpc"
	transporthttp "payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	pb "github.com/1B0-d/ap-pb/payment"
)

func main() {
	dbURL := getEnv("PAYMENT_DB_URL", "postgres://postgres:postgres@localhost:5434/paymentdb?sslmode=disable")
	port := getEnv("PAYMENT_SERVICE_PORT", "8081")
	rabbitMQURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	migrationPath := getEnv("PAYMENT_MIGRATION_PATH", "migrations/001_create_payments_table.sql")

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
		log.Fatalf("failed to run payment migrations: %v", err)
	}

	paymentRepo := repository.NewPaymentRepository(dbpool)
	paymentPublisher, err := messaging.NewRabbitMQPublisher(rabbitMQURL)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}
	defer func() {
		_ = paymentPublisher.Close()
	}()

	paymentUsecase := usecase.NewPaymentUsecase(paymentRepo, paymentPublisher)
	paymentHandler := transporthttp.NewPaymentHandler(paymentUsecase)

	paymentGRPCHost := getEnv("PAYMENT_GRPC_ADDR", "0.0.0.0")
	grpcPort := getEnv("PAYMENT_GRPC_PORT", "50051")
	grpcLis, err := net.Listen("tcp", net.JoinHostPort(paymentGRPCHost, grpcPort))
	if err != nil {
		log.Fatalf("failed to listen grpc: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPaymentServiceServer(grpcServer, transportgrpc.NewPaymentGRPCServer(paymentUsecase))

	errCh := make(chan error, 2)
	go func() {
		log.Printf("payment-service grpc running on port %s", grpcPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			if !errors.Is(err, grpc.ErrServerStopped) {
				errCh <- err
			}
		}
	}()

	router := gin.Default()
	transporthttp.RegisterPaymentRoutes(router, paymentHandler)

	httpServer := &stdhttp.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	log.Printf("payment-service running on port %s", port)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			if !errors.Is(err, stdhttp.ErrServerClosed) {
				errCh <- err
			}
		}
	}()

	select {
	case <-appCtx.Done():
		log.Println("payment-service shutdown requested")
	case err := <-errCh:
		log.Printf("payment-service server error: %v", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("failed to shutdown payment http server: %v", err)
	}

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
