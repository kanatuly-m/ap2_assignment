package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"order-service/internal/domain"
)

type HTTPPaymentClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPPaymentClient(baseURL string, timeout time.Duration) domain.PaymentGateway {
	return &HTTPPaymentClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *HTTPPaymentClient) ProcessPayment(ctx context.Context, order *domain.Order) (string, error) {
	paymentReq, err := json.Marshal(map[string]interface{}{
		"order_id":       order.ID,
		"amount":         order.Amount,
		"customer_email": order.CustomerEmail,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/payments", bytes.NewBuffer(paymentReq))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
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
