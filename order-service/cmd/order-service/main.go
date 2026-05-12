package main

import (
	"context"
	"errors"
	"log"
	"net"
	stdhttp "net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	ordercache "order-service/internal/cache"
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
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := getEnvAsInt("REDIS_DB", 0)
	cacheTTL := getEnvAsDuration("ORDER_CACHE_TTL", 5*time.Minute)
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

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})
	defer func() {
		_ = redisClient.Close()
	}()

	redisCtx, redisCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisClient.Ping(redisCtx).Err(); err != nil {
		redisCancel()
		log.Fatalf("failed to connect to redis: %v", err)
	}
	redisCancel()

	orderRepo := repository.NewOrderRepository(dbpool)
	orderCache := ordercache.NewRedisOrderCache(redisClient, cacheTTL)

	paymentClient, err := repository.NewPaymentGRPCClient(paymentServiceGRPCAddress)
	if err != nil {
		log.Fatalf("failed to connect to payment grpc service: %v", err)
	}
	defer func() {
		_ = paymentClient.Close()
	}()

	notifier := pubsub.NewOrderStatusBroadcaster()
	orderUsecase := usecase.NewOrderUsecase(orderRepo, paymentClient, notifier, orderCache)
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
	if getEnvAsBool("ORDER_RATE_LIMIT_ENABLED", false) {
		router.Use(transporthttp.NewRedisRateLimiter(
			redisClient,
			int64(getEnvAsInt("ORDER_RATE_LIMIT_REQUESTS", 10)),
			getEnvAsDuration("ORDER_RATE_LIMIT_WINDOW", time.Minute),
		))
	}
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

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("invalid %s=%q, using %d", key, value, fallback)
		return fallback
	}
	return parsed
}

func getEnvAsBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("invalid %s=%q, using %t", key, value, fallback)
		return fallback
	}
	return parsed
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("invalid %s=%q, using %s", key, value, fallback)
		return fallback
	}
	return parsed
}
