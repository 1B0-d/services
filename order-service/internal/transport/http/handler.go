package http

import (
	stdhttp "net/http"
	"time"

	"github.com/gin-gonic/gin"

	"order-service/internal/domain"
	"order-service/internal/usecase"
)

type OrderHandler struct {
	usecase *usecase.OrderUsecase
}

func NewOrderHandler(usecase *usecase.OrderUsecase) *OrderHandler {
	return &OrderHandler{usecase: usecase}
}

type createOrderRequest struct {
	CustomerID    string `json:"customer_id" binding:"required"`
	CustomerEmail string `json:"customer_email" binding:"required,email"`
	ItemName      string `json:"item_name" binding:"required"`
	Amount        int64  `json:"amount" binding:"required"`
}

type orderResponse struct {
	ID            string    `json:"id"`
	CustomerID    string    `json:"customer_id"`
	CustomerEmail string    `json:"customer_email"`
	ItemName      string    `json:"item_name"`
	Amount        int64     `json:"amount"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(stdhttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.usecase.CreateOrder(req.CustomerID, req.CustomerEmail, req.ItemName, req.Amount)
	if err != nil {
		switch err {
		case usecase.ErrInvalidAmount:
			c.JSON(stdhttp.StatusBadRequest, gin.H{"error": err.Error()})
			return
		case usecase.ErrPaymentServiceUnavailable:
			c.JSON(stdhttp.StatusServiceUnavailable, gin.H{
				"error": "payment service unavailable",
				"order": toOrderResponse(order),
			})
			return
		default:
			c.JSON(stdhttp.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
	}

	c.JSON(stdhttp.StatusCreated, toOrderResponse(order))
}

func (h *OrderHandler) GetOrderByID(c *gin.Context) {
	id := c.Param("id")

	order, err := h.usecase.GetOrderByID(id)
	if err != nil {
		if err == usecase.ErrOrderNotFound {
			c.JSON(stdhttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(stdhttp.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(stdhttp.StatusOK, toOrderResponse(order))
}

func (h *OrderHandler) GetOrdersByCustomerID(c *gin.Context) {
	customerID := c.Query("customer_id")
	if customerID == "" {
		c.JSON(stdhttp.StatusBadRequest, gin.H{"error": "customer_id is required"})
		return
	}

	orders, err := h.usecase.GetOrdersByCustomerID(customerID)
	if err != nil {
		c.JSON(stdhttp.StatusInternalServerError, gin.H{"error": "failed to get orders"})
		return
	}

	var response []orderResponse
	for _, order := range orders {
		response = append(response, toOrderResponse(order))
	}

	c.JSON(stdhttp.StatusOK, response)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.usecase.CancelOrder(id)
	if err != nil {
		switch err {
		case usecase.ErrOrderNotFound:
			c.JSON(stdhttp.StatusNotFound, gin.H{"error": err.Error()})
			return
		case usecase.ErrOrderCannotBeCancelled:
			c.JSON(stdhttp.StatusConflict, gin.H{"error": err.Error()})
			return
		default:
			c.JSON(stdhttp.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
	}

	c.JSON(stdhttp.StatusOK, toOrderResponse(order))
}

func toOrderResponse(order *domain.Order) orderResponse {
	return orderResponse{
		ID:            order.ID,
		CustomerID:    order.CustomerID,
		CustomerEmail: order.CustomerEmail,
		ItemName:      order.ItemName,
		Amount:        order.Amount,
		Status:        order.Status,
		CreatedAt:     order.CreatedAt,
	}
}
