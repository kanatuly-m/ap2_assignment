package main

import (
	"context"
	"database/sql"
	"log"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"ap2_assignment/shared/config"
	"notification-service/internal/messaging"
	"notification-service/internal/repository"
	"notification-service/internal/usecase"

	_ "github.com/lib/pq"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	databaseURL := config.GetEnv("DATABASE_URL", "postgres://user:password@localhost:5432/notification_db?sslmode=disable")
	rabbitURL := config.GetEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	exchange := config.GetEnv("PAYMENT_EXCHANGE", "payment.events")
	routingKey := config.GetEnv("PAYMENT_ROUTING_KEY", "payment.completed")
	queueName := config.GetEnv("PAYMENT_QUEUE", "payment.completed")
	dlxName := config.GetEnv("PAYMENT_DLX", "payment.dlx")
	dlqName := config.GetEnv("PAYMENT_DLQ", "payment.completed.dlq")
	maxRetries, err := strconv.Atoi(config.GetEnv("MAX_RETRIES", "3"))
	if err != nil || maxRetries < 1 {
		maxRetries = 3
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal("failed to open notification database: ", err)
	}
	defer db.Close()

	if err := waitForDB(ctx, db); err != nil {
		log.Fatal("notification database is not ready: ", err)
	}

	store := repository.NewPostgresProcessedEventStore(db)
	uc := usecase.NewNotificationUseCase(store)

	consumer, err := messaging.NewRabbitMQConsumer(rabbitURL, exchange, routingKey, queueName, dlxName, dlqName, maxRetries, uc)
	if err != nil {
		log.Fatal("failed to connect to rabbitmq: ", err)
	}
	defer consumer.Close()

	if err := consumer.Start(ctx); err != nil {
		log.Fatal("notification consumer stopped with error: ", err)
	}

	log.Println("Notification Service stopped gracefully")
}

func waitForDB(ctx context.Context, db *sql.DB) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		if err := db.PingContext(ctx); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
