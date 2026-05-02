package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"notification-service/internal/idempotency"
	"notification-service/internal/messaging"
	"notification-service/internal/usecase"
)

func main() {
	rabbitMQURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	failEmail := os.Getenv("NOTIFICATION_FAIL_EMAIL")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := idempotency.NewMemoryStore()
	notificationUsecase := usecase.NewNotificationUsecase(store, failEmail)

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

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
