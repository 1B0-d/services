package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"payment-service/internal/repository"
	transporthttp "payment-service/internal/transport/http"
	"payment-service/internal/usecase"
)

func main() {
	dbURL := getEnv("PAYMENT_DB_URL", "postgres://postgres:postgres@localhost:5434/paymentdb?sslmode=disable")
	port := getEnv("PAYMENT_SERVICE_PORT", "8081")

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

	paymentRepo := repository.NewPaymentRepository(dbpool)
	paymentUsecase := usecase.NewPaymentUsecase(paymentRepo)
	paymentHandler := transporthttp.NewPaymentHandler(paymentUsecase)

	router := gin.Default()
	transporthttp.RegisterPaymentRoutes(router, paymentHandler)

	log.Printf("payment-service running on port %s", port)
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
