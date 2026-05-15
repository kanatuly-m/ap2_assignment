package usecase

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"order-service/internal/domain"

	"github.com/google/uuid"
)

type orderUseCase struct {
	repo           domain.OrderRepository
	paymentGateway domain.PaymentGateway
	cache          domain.OrderCache
	cacheTTL       time.Duration
}

func NewOrderUseCase(repo domain.OrderRepository, paymentGateway domain.PaymentGateway, cache domain.OrderCache, cacheTTL time.Duration) domain.OrderUseCase {
	return &orderUseCase{
		repo:           repo,
		paymentGateway: paymentGateway,
		cache:          cache,
		cacheTTL:       cacheTTL,
	}
}

func (u *orderUseCase) CreateOrder(ctx context.Context, customerID, customerEmail, itemName string, amount int64, idempKey string) (*domain.Order, error) {
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

	paymentStatus, err := u.paymentGateway.ProcessPayment(ctx, order)
	if err != nil {
		_ = u.repo.UpdateStatus(order.ID, "Failed")
		order.Status = "Failed"
		u.invalidateCache(ctx, order.ID, "payment request failed")
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
	u.invalidateCache(ctx, order.ID, "order status changed after payment")

	return order, nil
}

func (u *orderUseCase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	if u.cache != nil {
		cachedOrder, found, err := u.cache.Get(ctx, id)
		if err != nil {
			log.Printf("[Cache] Redis read failed for order:%s: %v", id, err)
		} else if found {
			log.Printf("[Cache] HIT order:%s", id)
			return cachedOrder, nil
		} else {
			log.Printf("[Cache] MISS order:%s", id)
		}
	}

	order, err := u.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if u.cache != nil {
		if err := u.cache.Set(ctx, order, u.cacheTTL); err != nil {
			log.Printf("[Cache] Redis write failed for order:%s: %v", id, err)
		} else {
			log.Printf("[Cache] SET order:%s ttl=%s", id, u.cacheTTL)
		}
	}
	return order, nil
}

func (u *orderUseCase) CancelOrder(ctx context.Context, id string) error {
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
	if err := u.repo.UpdateStatus(id, "Cancelled"); err != nil {
		return err
	}
	u.invalidateCache(ctx, id, "order cancelled")
	return nil
}

func (u *orderUseCase) GetOrdersByAmountRange(ctx context.Context, minAmount, maxAmount int64) ([]*domain.Order, error) {
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

func (u *orderUseCase) invalidateCache(ctx context.Context, orderID string, reason string) {
	if u.cache == nil {
		return
	}
	if err := u.cache.Delete(ctx, orderID); err != nil {
		log.Printf("[Cache] INVALIDATE failed order:%s reason=%s err=%v", orderID, reason, err)
		return
	}
	log.Printf("[Cache] INVALIDATED order:%s reason=%s", orderID, reason)
}
