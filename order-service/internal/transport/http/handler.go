package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"order-service/internal/domain"
	"strconv"
)

type OrderHandler struct {
	useCase domain.OrderUseCase
}

func NewOrderHandler(uc domain.OrderUseCase) *OrderHandler {
	return &OrderHandler{useCase: uc}
}

type CreateOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name" binding:"required"`
	Amount     int64  `json:"amount" binding:"required"`
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idempKey := c.GetHeader("Idempotency-Key")

	order, err := h.useCase.CreateOrder(req.CustomerID, req.ItemName, req.Amount, idempKey)
	if err != nil {
		if err.Error() == "503 Service Unavailable" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment Service is unavailable"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	order, err := h.useCase.GetOrder(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	err := h.useCase.CancelOrder(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order cancelled"})
}

func (h *OrderHandler) GetOrdersByAmount(c *gin.Context) {
	minStr := c.Query("min_amount")
	maxStr := c.Query("max_amount")

	minAmount, err := strconv.ParseInt(minStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or missing min_amount"})
		return
	}

	maxAmount, err := strconv.ParseInt(maxStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or missing max_amount"})
		return
	}

	orders, err := h.useCase.GetOrdersByAmountRange(minAmount, maxAmount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}
