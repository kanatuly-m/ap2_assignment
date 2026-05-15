package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"ap2_assignment/shared/events"
	"notification-service/internal/domain"
)

type MockEmailSender struct {
	latency         time.Duration
	failureRate     float64
	alwaysFailEmail string
}

func NewMockEmailSender(latency time.Duration, failureRate float64, alwaysFailEmail string) domain.EmailSender {
	if failureRate < 0 {
		failureRate = 0
	}
	if failureRate > 1 {
		failureRate = 1
	}
	return &MockEmailSender{
		latency:         latency,
		failureRate:     failureRate,
		alwaysFailEmail: strings.TrimSpace(alwaysFailEmail),
	}
}

func (s *MockEmailSender) Send(ctx context.Context, event events.PaymentCompletedEvent) error {
	if err := waitWithContext(ctx, s.latency); err != nil {
		return err
	}

	if s.alwaysFailEmail != "" && strings.EqualFold(strings.TrimSpace(event.CustomerEmail), s.alwaysFailEmail) {
		return errors.New("simulated permanent provider failure")
	}

	if s.failureRate > 0 && rand.Float64() < s.failureRate {
		return errors.New("simulated transient provider failure")
	}

	log.Printf("[Provider:SIMULATED] Accepted notification for %s", event.CustomerEmail)
	return nil
}

func waitWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("provider wait cancelled: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}
