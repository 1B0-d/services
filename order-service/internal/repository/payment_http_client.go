package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"order-service/internal/domain"
)

type PaymentHTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewPaymentHTTPClient(baseURL string, client *http.Client) *PaymentHTTPClient {
	return &PaymentHTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

type createPaymentRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type createPaymentResponse struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
}

func (c *PaymentHTTPClient) CreatePayment(orderID string, amount int64) (*domain.PaymentResult, error) {
	body := createPaymentRequest{
		OrderID: orderID,
		Amount:  amount,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/payments", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("payment service returned status %d", resp.StatusCode)
	}

	var result createPaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &domain.PaymentResult{
		Status:        result.Status,
		TransactionID: result.TransactionID,
	}, nil
}
