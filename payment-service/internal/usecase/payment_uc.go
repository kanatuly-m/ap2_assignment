package usecase

import (
	"errors"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

type paymentUseCase struct {
	repo domain.PaymentRepository
}

func NewPaymentUseCase(repo domain.PaymentRepository) domain.PaymentUseCase {
	return &paymentUseCase{repo: repo}
}

func (u *paymentUseCase) ProcessPayment(orderID string, amount int64) (*domain.Payment, error) {
	if amount <= 0 {
		return &domain.Payment{Status: "Declined"}, nil
	}

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		TransactionID: uuid.New().String(),
		Amount:        amount,
		Status:        "Authorized",
	}

	if amount > 100000 {
		payment.Status = "Declined"
	}

	err := u.repo.Save(payment)
	return payment, err
}

func (u *paymentUseCase) GetPaymentStatus(orderID string) (*domain.Payment, error) {
	return u.repo.GetByOrderID(orderID)
}

func (u *paymentUseCase) ListPayments(status string) ([]*domain.Payment, error) {
	if status != "Authorized" && status != "Declined" {
		return nil, errors.New("status must be Authorized or Declined")
	}

	return u.repo.ListByStatus(status)
}
