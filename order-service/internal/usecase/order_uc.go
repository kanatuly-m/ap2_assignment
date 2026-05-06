package usecase

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"order-service/internal/domain"

	"github.com/google/uuid"
)

type orderUseCase struct {
	repo       domain.OrderRepository
	paymentURL string
	httpClient *http.Client
}

func NewOrderUseCase(repo domain.OrderRepository, paymentURL string) domain.OrderUseCase {
	return &orderUseCase{
		repo:       repo,
		paymentURL: strings.TrimRight(paymentURL, "/"),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (u *orderUseCase) CreateOrder(customerID, customerEmail, itemName string, amount int64, idempKey string) (*domain.Order, error) {
	if strings.TrimSpace(customerID) == "" {
		return nil, errors.New("customer_id is required")
	}
	if strings.TrimSpace(customerEmail) == "" {
		return nil, errors.New("customer_email is required")
	}
	if strings.TrimSpace(itemName) == "" {
		return nil, errors.New("item_name is required")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be > 0")
	}

	if idempKey != "" {
		existingOrder, err := u.repo.GetByIdempotencyKey(idempKey)
		if err == nil && existingOrder != nil {
			return existingOrder, nil
		}
	}

	order := &domain.Order{
		ID:             uuid.New().String(),
		CustomerID:     customerID,
		CustomerEmail:  customerEmail,
		ItemName:       itemName,
		Amount:         amount,
		Status:         "Pending",
		CreatedAt:      time.Now().UTC(),
		IdempotencyKey: idempKey,
	}

	if err := u.repo.Save(order); err != nil {
		return nil, err
	}

	paymentStatus, err := u.processPayment(order)
	if err != nil {
		_ = u.repo.UpdateStatus(order.ID, "Failed")
		order.Status = "Failed"
		return nil, err
	}

	if paymentStatus == "Authorized" {
		order.Status = "Paid"
	} else {
		order.Status = "Failed"
	}

	if err := u.repo.UpdateStatus(order.ID, order.Status); err != nil {
		return nil, err
	}

	return order, nil
}

func (u *orderUseCase) processPayment(order *domain.Order) (string, error) {
	paymentReq, err := json.Marshal(map[string]interface{}{
		"order_id":       order.ID,
		"amount":         order.Amount,
		"customer_email": order.CustomerEmail,
	})
	if err != nil {
		return "", err
	}

	resp, err := u.httpClient.Post(u.paymentURL+"/payments", "application/json", bytes.NewBuffer(paymentReq))
	if err != nil {
		return "", errors.New("payment service is unavailable")
	}
	defer resp.Body.Close()

	var paymentResp struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&paymentResp); err != nil {
		return "", err
	}

	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("payment service error: %s", paymentResp.Error)
	}
	if paymentResp.Status == "" {
		return "", errors.New("payment service returned empty status")
	}
	return paymentResp.Status, nil
}

func (u *orderUseCase) GetOrder(id string) (*domain.Order, error) {
	return u.repo.GetByID(id)
}

func (u *orderUseCase) CancelOrder(id string) error {
	order, err := u.repo.GetByID(id)
	if err != nil {
		return err
	}
	if order.Status == "Paid" {
		return errors.New("cannot cancel a paid order")
	}
	if order.Status != "Pending" {
		return errors.New("only pending orders can be cancelled")
	}
	return u.repo.UpdateStatus(id, "Cancelled")
}

func (u *orderUseCase) GetOrdersByAmountRange(minAmount, maxAmount int64) ([]*domain.Order, error) {
	if minAmount < 0 {
		return nil, errors.New("min_amount cannot be less than 0")
	}
	if maxAmount > 100000 {
		return nil, errors.New("max_amount cannot be greater than 100000")
	}
	if minAmount > maxAmount {
		return nil, errors.New("min_amount cannot be greater than max_amount")
	}
	return u.repo.GetByAmountRange(minAmount, maxAmount)
}
