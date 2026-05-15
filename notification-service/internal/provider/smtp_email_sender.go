package provider

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"ap2_assignment/shared/events"
	"notification-service/internal/domain"
)

type SMTPEmailSender struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPEmailSender(host, port, username, password, from string) (domain.EmailSender, error) {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	from = strings.TrimSpace(from)
	if host == "" || port == "" || from == "" {
		return nil, errors.New("SMTP_HOST, SMTP_PORT and SMTP_FROM are required in REAL provider mode")
	}
	return &SMTPEmailSender{
		host:     host,
		port:     port,
		username: strings.TrimSpace(username),
		password: password,
		from:     from,
	}, nil
}

func (s *SMTPEmailSender) Send(ctx context.Context, event events.PaymentCompletedEvent) error {
	addr := net.JoinHostPort(s.host, s.port)
	subject := fmt.Sprintf("Payment completed for Order #%s", event.OrderID)
	body := fmt.Sprintf("Your payment of $%s has been completed successfully for Order #%s.", event.AmountAsDollars(), event.OrderID)
	message := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", event.CustomerEmail, subject, body))

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, s.from, []string{event.CustomerEmail}, message)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
