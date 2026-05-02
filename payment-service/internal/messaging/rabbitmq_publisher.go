package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"payment-service/internal/domain"
)

const (
	paymentExchange       = "payments"
	paymentCompletedQueue = "payment.completed"
	paymentCompletedKey   = "payment.completed"
	deadLetterExchange    = "payments.dlx"
	deadLetterQueue       = "payment.completed.dlq"
	deadLetterKey         = "payment.completed.dead"
)

type RabbitMQPublisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	confirms <-chan amqp.Confirmation
	mu       sync.Mutex
}

func NewRabbitMQPublisher(url string) (*RabbitMQPublisher, error) {
	conn, err := dialWithRetry(url, 10, 2*time.Second)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	if err := channel.Confirm(false); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	publisher := &RabbitMQPublisher{
		conn:     conn,
		channel:  channel,
		confirms: channel.NotifyPublish(make(chan amqp.Confirmation, 1)),
	}
	if err := publisher.setupTopology(); err != nil {
		_ = publisher.Close()
		return nil, err
	}

	return publisher, nil
}

func (p *RabbitMQPublisher) PublishPaymentCompleted(event domain.PaymentCompletedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal payment completed event: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p.mu.Lock()
	defer p.mu.Unlock()

	err = p.channel.PublishWithContext(
		ctx,
		paymentExchange,
		paymentCompletedKey,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    event.EventID,
			Type:         paymentCompletedKey,
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish payment completed event: %w", err)
	}

	select {
	case confirmation := <-p.confirms:
		if !confirmation.Ack {
			return fmt.Errorf("rabbitmq did not acknowledge published event %s", event.EventID)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for rabbitmq publish confirm: %w", ctx.Err())
	}
}

func (p *RabbitMQPublisher) Close() error {
	if p == nil {
		return nil
	}

	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

func (p *RabbitMQPublisher) setupTopology() error {
	if err := p.channel.ExchangeDeclare(paymentExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare payment exchange: %w", err)
	}
	if err := p.channel.ExchangeDeclare(deadLetterExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dead-letter exchange: %w", err)
	}

	queueArgs := amqp.Table{
		"x-dead-letter-exchange":    deadLetterExchange,
		"x-dead-letter-routing-key": deadLetterKey,
	}
	if _, err := p.channel.QueueDeclare(paymentCompletedQueue, true, false, false, false, queueArgs); err != nil {
		return fmt.Errorf("declare payment completed queue: %w", err)
	}
	if err := p.channel.QueueBind(paymentCompletedQueue, paymentCompletedKey, paymentExchange, false, nil); err != nil {
		return fmt.Errorf("bind payment completed queue: %w", err)
	}

	if _, err := p.channel.QueueDeclare(deadLetterQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dead-letter queue: %w", err)
	}
	if err := p.channel.QueueBind(deadLetterQueue, deadLetterKey, deadLetterExchange, false, nil); err != nil {
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
