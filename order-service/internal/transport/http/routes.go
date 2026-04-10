package http

import "github.com/gin-gonic/gin"

func RegisterOrderRoutes(router *gin.Engine, handler *OrderHandler) {
	router.POST("/orders", handler.CreateOrder)
	router.GET("/orders/:id", handler.GetOrderByID)
	router.PATCH("/orders/:id/cancel", handler.CancelOrder)
	router.GET("/orders", handler.GetOrdersByCustomerID)
}
