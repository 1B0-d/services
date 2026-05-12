package provider

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"notification-service/internal/domain"
)

type SMTPEmailSender struct {
	host     string
	port     string
	username string
	password string
	from     string
	timeout  time.Duration
}

func NewSMTPEmailSender(host, port, username, password, from string, timeout time.Duration) (*SMTPEmailSender, error) {
	if host == "" {
		return nil, fmt.Errorf("SMTP_HOST is required")
	}
	if port == "" {
		port = "587"
	}
	if from == "" {
		return nil, fmt.Errorf("SMTP_FROM is required")
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &SMTPEmailSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		timeout:  timeout,
	}, nil
}

func (s *SMTPEmailSender) SendPaymentCompleted(ctx context.Context, event domain.PaymentCompletedEvent) error {
	addr := net.JoinHostPort(s.host, s.port)
	message := s.message(event)

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	sendCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, s.from, []string{event.CustomerEmail}, []byte(message))
	}()

	select {
	case <-sendCtx.Done():
		return sendCtx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("send smtp email: %w", err)
		}
		return nil
	}
}

func (s *SMTPEmailSender) message(event domain.PaymentCompletedEvent) string {
	subject := fmt.Sprintf("Payment completed for order %s", event.OrderID)
	body := fmt.Sprintf(
		"Your payment %s for order %s was completed. Amount: $%.2f",
		event.PaymentID,
		event.OrderID,
		float64(event.Amount)/100,
	)

	headers := []string{
		"From: " + s.from,
		"To: " + event.CustomerEmail,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	}

	return strings.Join(headers, "\r\n") + "\r\n\r\n" + body
}
