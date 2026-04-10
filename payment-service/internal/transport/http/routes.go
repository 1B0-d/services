package http

import "github.com/gin-gonic/gin"

func RegisterPaymentRoutes(router *gin.Engine, handler *PaymentHandler) {
	router.POST("/payments", handler.CreatePayment)
	router.GET("/payments/:order_id", handler.GetPaymentByOrderID)
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "payment-service is running"})
	})
}
