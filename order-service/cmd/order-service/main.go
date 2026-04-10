package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-service/internal/repository"
	transporthttp "order-service/internal/transport/http"
	"order-service/internal/usecase"
)

func main() {
	dbURL := getEnv("ORDER_DB_URL", "postgres://postgres:postgres@localhost:5435/orderdb?sslmode=disable")
	port := getEnv("ORDER_SERVICE_PORT", "8080")
	paymentServiceURL := getEnv("PAYMENT_SERVICE_URL", "http://localhost:8081")

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

	httpClient := &http.Client{
		Timeout: 2 * time.Second,
	}

	paymentClient := repository.NewPaymentHTTPClient(paymentServiceURL, httpClient)

	orderUsecase := usecase.NewOrderUsecase(orderRepo, paymentClient)
	orderHandler := transporthttp.NewOrderHandler(orderUsecase)

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
