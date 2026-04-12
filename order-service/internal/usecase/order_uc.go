package usecase

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"order-service/internal/domain"
	"time"

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
		paymentURL: paymentURL,
		httpClient: &http.Client{Timeout: 2 * time.Second}, // Required Failure Scenario
	}
}

func (u *orderUseCase) CreateOrder(customerID, itemName string, amount int64, idempKey string) (*domain.Order, error) {
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
		ItemName:       itemName,
		Amount:         amount,
		Status:         "Pending",
		CreatedAt:      time.Now(),
		IdempotencyKey: idempKey,
	}

	if err := u.repo.Save(order); err != nil {
		return nil, err
	}

	paymentReq, _ := json.Marshal(map[string]interface{}{
		"order_id": order.ID,
		"amount":   order.Amount,
	})

	resp, err := u.httpClient.Post(u.paymentURL+"/payments", "application/json", bytes.NewBuffer(paymentReq))
	if err != nil {
		u.repo.UpdateStatus(order.ID, "Failed")
		return nil, errors.New("503 Service Unavailable")
	}
	defer resp.Body.Close()

	var paymentResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&paymentResp)

	if paymentResp["status"] == "Authorized" {
		order.Status = "Paid"
	} else {
		order.Status = "Failed"
	}

	u.repo.UpdateStatus(order.ID, order.Status)
	return order, nil
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
