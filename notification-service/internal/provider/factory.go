package provider

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"ap2_assignment/shared/config"
	"notification-service/internal/domain"
)

func NewEmailSenderFromEnv() (domain.EmailSender, string, error) {
	mode := strings.ToUpper(strings.TrimSpace(config.GetEnv("PROVIDER_MODE", "SIMULATED")))
	switch mode {
	case "REAL":
		sender, err := NewSMTPEmailSender(
			config.GetEnv("SMTP_HOST", ""),
			config.GetEnv("SMTP_PORT", "587"),
			config.GetEnv("SMTP_USERNAME", ""),
			config.GetEnv("SMTP_PASSWORD", ""),
			config.GetEnv("SMTP_FROM", ""),
		)
		return sender, mode, err
	case "SIMULATED":
		latencyMS := intFromEnv("SIMULATED_LATENCY_MS", 500)
		failureRate := floatFromEnv("SIMULATED_FAILURE_RATE", 0.20)
		sender := NewMockEmailSender(
			time.Duration(latencyMS)*time.Millisecond,
			failureRate,
			config.GetEnv("SIMULATED_ALWAYS_FAIL_EMAIL", "fail@example.com"),
		)
		return sender, mode, nil
	default:
		return nil, mode, fmt.Errorf("unsupported PROVIDER_MODE: %s", mode)
	}
}

func intFromEnv(key string, fallback int) int {
	value, err := strconv.Atoi(config.GetEnv(key, strconv.Itoa(fallback)))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func floatFromEnv(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(config.GetEnv(key, strconv.FormatFloat(fallback, 'f', -1, 64)), 64)
	if err != nil || value < 0 || value > 1 {
		return fallback
	}
	return value
}
