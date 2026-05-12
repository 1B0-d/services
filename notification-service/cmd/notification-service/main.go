package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"notification-service/internal/domain"
	"notification-service/internal/idempotency"
	"notification-service/internal/messaging"
	"notification-service/internal/provider"
	"notification-service/internal/usecase"
)

func main() {
	rabbitMQURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := getEnvAsInt("REDIS_DB", 0)
	idempotencyTTL := getEnvAsDuration("NOTIFICATION_IDEMPOTENCY_TTL", 24*time.Hour)
	retryCount := getEnvAsInt("NOTIFICATION_RETRY_COUNT", 4)
	retryBaseDelay := getEnvAsDuration("NOTIFICATION_RETRY_BASE_DELAY", 2*time.Second)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})
	defer func() {
		_ = redisClient.Close()
	}()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		pingCancel()
		log.Fatalf("failed to connect to redis: %v", err)
	}
	pingCancel()

	notificationProvider, err := buildNotificationProvider()
	if err != nil {
		log.Fatalf("failed to configure notification provider: %v", err)
	}

	store := idempotency.NewRedisStore(redisClient, idempotencyTTL)
	notificationUsecase := usecase.NewNotificationUsecase(store, notificationProvider, retryCount, retryBaseDelay)

	consumer, err := messaging.NewRabbitMQConsumer(rabbitMQURL, notificationUsecase)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}
	defer func() {
		_ = consumer.Close()
	}()

	if err := consumer.Run(ctx); err != nil {
		log.Fatalf("notification-service stopped with error: %v", err)
	}

	log.Println("notification-service shutdown complete")
}

func buildNotificationProvider() (domain.NotificationProvider, error) {
	mode := strings.ToUpper(getEnv("PROVIDER_MODE", "SIMULATED"))
	switch mode {
	case "SIMULATED", "MOCK":
		return provider.NewSimulatedEmailSender(
			getEnvAsDuration("SIMULATED_PROVIDER_LATENCY", 750*time.Millisecond),
			getEnvAsFloat("SIMULATED_PROVIDER_FAILURE_RATE", 0.25),
		), nil
	case "SMTP", "REAL":
		return provider.NewSMTPEmailSender(
			os.Getenv("SMTP_HOST"),
			getEnv("SMTP_PORT", "587"),
			os.Getenv("SMTP_USERNAME"),
			os.Getenv("SMTP_PASSWORD"),
			os.Getenv("SMTP_FROM"),
			getEnvAsDuration("SMTP_TIMEOUT", 10*time.Second),
		)
	default:
		return nil, fmt.Errorf("unsupported PROVIDER_MODE %q", mode)
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

func getEnvAsFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Printf("invalid %s=%q, using %.2f", key, value, fallback)
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
