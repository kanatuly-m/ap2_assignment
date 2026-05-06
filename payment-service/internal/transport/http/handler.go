package http

import (
	"net/http"

	"payment-service/internal/domain"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	useCase domain.PaymentUseCase
}

func NewPaymentHandler(uc domain.PaymentUseCase) *PaymentHandler {
	return &PaymentHandler{useCase: uc}
}

type PaymentRequest struct {
	OrderID       string `json:"order_id" binding:"required"`
	CustomerEmail string `json:"customer_email" binding:"required,email"`
	Amount        int64  `json:"amount" binding:"required"`
}

func (h *PaymentHandler) ProcessPayment(c *gin.Context) {
	var req PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := h.useCase.ProcessPayment(c.Request.Context(), req.OrderID, req.CustomerEmail, req.Amount)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         payment.Status,
		"transaction_id": payment.TransactionID,
	})
}

func (h *PaymentHandler) GetPayment(c *gin.Context) {
	payment, err := h.useCase.GetPaymentStatus(c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	c.JSON(http.StatusOK, payment)
}
