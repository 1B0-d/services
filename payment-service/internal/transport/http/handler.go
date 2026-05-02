package http

import (
	stdhttp "net/http"

	"github.com/gin-gonic/gin"

	"payment-service/internal/usecase"
)

type PaymentHandler struct {
	usecase *usecase.PaymentUsecase
}

func NewPaymentHandler(usecase *usecase.PaymentUsecase) *PaymentHandler {
	return &PaymentHandler{usecase: usecase}
}

type createPaymentRequest struct {
	OrderID       string `json:"order_id" binding:"required"`
	CustomerEmail string `json:"customer_email" binding:"required,email"`
	Amount        int64  `json:"amount" binding:"required"`
}

type paymentResponse struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
	CustomerEmail string `json:"customer_email"`
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req createPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(stdhttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := h.usecase.CreatePayment(req.OrderID, req.CustomerEmail, req.Amount)
	if err != nil {
		if err == usecase.ErrInvalidAmount {
			c.JSON(stdhttp.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(stdhttp.StatusInternalServerError, gin.H{"error": "failed to create payment"})
		return
	}

	c.JSON(stdhttp.StatusCreated, paymentResponse{
		ID:            payment.ID,
		OrderID:       payment.OrderID,
		TransactionID: payment.TransactionID,
		Amount:        payment.Amount,
		Status:        payment.Status,
		CustomerEmail: payment.CustomerEmail,
	})
}

func (h *PaymentHandler) GetPaymentByOrderID(c *gin.Context) {
	orderID := c.Param("order_id")

	payment, err := h.usecase.GetPaymentByOrderID(orderID)
	if err != nil {
		if err == usecase.ErrPaymentNotFound {
			c.JSON(stdhttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(stdhttp.StatusInternalServerError, gin.H{"error": "failed to get payment"})
		return
	}

	c.JSON(stdhttp.StatusOK, paymentResponse{
		ID:            payment.ID,
		OrderID:       payment.OrderID,
		TransactionID: payment.TransactionID,
		Amount:        payment.Amount,
		Status:        payment.Status,
		CustomerEmail: payment.CustomerEmail,
	})
}
