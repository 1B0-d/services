package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"notification-service/internal/domain"
)

const (
	paymentExchange       = "payments"
	paymentCompletedQueue = "payment.completed"
	paymentCompletedKey   = "payment.completed"
	deadLetterExchange    = "payments.dlx"
	deadLetterQueue       = "payment.completed.dlq"
	deadLetterKey         = "payment.completed.dead"
)

type PaymentCompletedHandler interface {
	HandlePaymentCompleted(event domain.PaymentCompletedEvent) error
}

type RabbitMQConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	handler PaymentCompletedHandler
}

func NewRabbitMQConsumer(url string, handler PaymentCompletedHandler) (*RabbitMQConsumer, error) {
	conn, err := dialWithRetry(url, 10, 2*time.Second)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	consumer := &RabbitMQConsumer{
		conn:    conn,
		channel: channel,
		handler: handler,
	}
	if err := consumer.setupTopology(); err != nil {
		_ = consumer.Close()
		return nil, err
	}

	if err := channel.Qos(1, 0, false); err != nil {
		_ = consumer.Close()
		return nil, fmt.Errorf("configure rabbitmq qos: %w", err)
	}

	return consumer, nil
}

func (c *RabbitMQConsumer) Run(ctx context.Context) error {
	deliveries, err := c.channel.Consume(paymentCompletedQueue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume payment completed queue: %w", err)
	}

	log.Printf("notification-service listening to queue %s", paymentCompletedQueue)

	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return nil
			}
			c.handleDelivery(delivery)
		}
	}
}

func (c *RabbitMQConsumer) Close() error {
	if c == nil {
		return nil
	}

	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *RabbitMQConsumer) handleDelivery(delivery amqp.Delivery) {
	var event domain.PaymentCompletedEvent
	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		log.Printf("invalid payment.completed message moved to DLQ: %v", err)
		_ = delivery.Nack(false, false)
		return
	}

	if err := c.handler.HandlePaymentCompleted(event); err != nil {
		log.Printf("failed to process payment.completed event %s; moving to DLQ: %v", event.EventID, err)
		_ = delivery.Nack(false, false)
		return
	}

	if err := delivery.Ack(false); err != nil {
		log.Printf("failed to ack payment.completed event %s: %v", event.EventID, err)
	}
}

func (c *RabbitMQConsumer) setupTopology() error {
	if err := c.channel.ExchangeDeclare(paymentExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare payment exchange: %w", err)
	}
	if err := c.channel.ExchangeDeclare(deadLetterExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dead-letter exchange: %w", err)
	}

	queueArgs := amqp.Table{
		"x-dead-letter-exchange":    deadLetterExchange,
		"x-dead-letter-routing-key": deadLetterKey,
	}
	if _, err := c.channel.QueueDeclare(paymentCompletedQueue, true, false, false, false, queueArgs); err != nil {
		return fmt.Errorf("declare payment completed queue: %w", err)
	}
	if err := c.channel.QueueBind(paymentCompletedQueue, paymentCompletedKey, paymentExchange, false, nil); err != nil {
		return fmt.Errorf("bind payment completed queue: %w", err)
	}

	if _, err := c.channel.QueueDeclare(deadLetterQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dead-letter queue: %w", err)
	}
	if err := c.channel.QueueBind(deadLetterQueue, deadLetterKey, deadLetterExchange, false, nil); err != nil {
		return fmt.Errorf("bind dead-letter queue: %w", err)
	}

	return nil
}

func dialWithRetry(url string, attempts int, delay time.Duration) (*amqp.Connection, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		time.Sleep(delay)
	}
	return nil, fmt.Errorf("connect to rabbitmq after %d attempts: %w", attempts, lastErr)
}
