package http

import (
	"net/http"
	"strconv"

	"order-service/internal/domain"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	useCase domain.OrderUseCase
}

func NewOrderHandler(uc domain.OrderUseCase) *OrderHandler {
	return &OrderHandler{useCase: uc}
}

type CreateOrderRequest struct {
	CustomerID    string `json:"customer_id" binding:"required"`
	CustomerEmail string `json:"customer_email" binding:"required,email"`
	ItemName      string `json:"item_name" binding:"required"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idempKey := c.GetHeader("Idempotency-Key")
	order, err := h.useCase.CreateOrder(c.Request.Context(), req.CustomerID, req.CustomerEmail, req.ItemName, req.Amount, idempKey)
	if err != nil {
		if err.Error() == "payment service is unavailable" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	order, err := h.useCase.GetOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	err := h.useCase.CancelOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "order cancelled"})
}

func (h *OrderHandler) GetOrdersByAmount(c *gin.Context) {
	minAmount, err := strconv.ParseInt(c.Query("min_amount"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or missing min_amount"})
		return
	}
	maxAmount, err := strconv.ParseInt(c.Query("max_amount"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or missing max_amount"})
		return
	}

	orders, err := h.useCase.GetOrdersByAmountRange(c.Request.Context(), minAmount, maxAmount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, orders)
}
