package main

import (
	"context"
	"log"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"ap2_assignment/shared/config"
	"notification-service/internal/messaging"
	"notification-service/internal/provider"
	"notification-service/internal/repository"
	"notification-service/internal/usecase"

	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rabbitURL := config.GetEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")
	exchange := config.GetEnv("PAYMENT_EXCHANGE", "payment.events")
	routingKey := config.GetEnv("PAYMENT_ROUTING_KEY", "payment.completed")
	queueName := config.GetEnv("PAYMENT_QUEUE", "payment.completed")
	dlxName := config.GetEnv("PAYMENT_DLX", "payment.dlx")
	dlqName := config.GetEnv("PAYMENT_DLQ", "payment.completed.dlq")
	maxRetries := intFromEnv("MAX_RETRIES", 3)
	baseRetryDelay := secondsFromEnv("RETRY_BASE_DELAY_SECONDS", 2)
	jobStatusTTL := hoursFromEnv("NOTIFICATION_STATUS_TTL_HOURS", 24)

	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer redisClient.Close()
	if err := waitForRedis(ctx, redisClient); err != nil {
		log.Fatal("redis is not ready for notification service: ", err)
	}

	sender, providerMode, err := provider.NewEmailSenderFromEnv()
	if err != nil {
		log.Fatal("failed to initialize notification provider: ", err)
	}

	store := repository.NewRedisNotificationJobStore(redisClient)
	uc := usecase.NewNotificationUseCase(store, sender, jobStatusTTL)

	consumer, err := messaging.NewRabbitMQConsumer(rabbitURL, exchange, routingKey, queueName, dlxName, dlqName, maxRetries, baseRetryDelay, uc)
	if err != nil {
		log.Fatal("failed to connect to rabbitmq: ", err)
	}
	defer consumer.Close()

	log.Printf("Notification provider mode: %s | Retry backoff base: %s | Max retries: %d | Redis job TTL: %s", providerMode, baseRetryDelay, maxRetries, jobStatusTTL)
	if err := consumer.Start(ctx); err != nil {
		log.Fatal("notification consumer stopped with error: ", err)
	}

	log.Println("Notification Service stopped gracefully")
}

func waitForRedis(ctx context.Context, client *redis.Client) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		if err := client.Ping(ctx).Err(); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func intFromEnv(key string, fallback int) int {
	value, err := strconv.Atoi(config.GetEnv(key, strconv.Itoa(fallback)))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func secondsFromEnv(key string, fallback int) time.Duration {
	value := intFromEnv(key, fallback)
	if value <= 0 {
		value = fallback
	}
	return time.Duration(value) * time.Second
}

func hoursFromEnv(key string, fallback int) time.Duration {
	value := intFromEnv(key, fallback)
	if value <= 0 {
		value = fallback
	}
	return time.Duration(value) * time.Hour
}
