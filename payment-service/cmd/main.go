package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"ap2_assignment/shared/config"
	"payment-service/internal/messaging"
	"payment-service/internal/repository"
	transport "payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	databaseURL := config.GetEnv("DATABASE_URL", "postgres://user:password@localhost:5432/payment_db?sslmode=disable")
	rabbitURL := config.GetEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	exchange := config.GetEnv("PAYMENT_EXCHANGE", "payment.events")
	routingKey := config.GetEnv("PAYMENT_ROUTING_KEY", "payment.completed")
	queueName := config.GetEnv("PAYMENT_QUEUE", "payment.completed")
	dlxName := config.GetEnv("PAYMENT_DLX", "payment.dlx")
	dlqName := config.GetEnv("PAYMENT_DLQ", "payment.completed.dlq")
	port := config.GetEnv("PORT", "8081")

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal("failed to open payment database: ", err)
	}
	defer db.Close()

	if err := waitForDB(ctx, db); err != nil {
		log.Fatal("payment database is not ready: ", err)
	}

	publisher, err := messaging.NewRabbitMQPublisher(rabbitURL, exchange, routingKey, queueName, dlxName, dlqName)
	if err != nil {
		log.Fatal("failed to connect to rabbitmq: ", err)
	}
	defer publisher.Close()

	repo := repository.NewPostgresPaymentRepository(db)
	uc := usecase.NewPaymentUseCase(repo, publisher)
	handler := transport.NewPaymentHandler(uc)

	r := gin.Default()
	r.POST("/payments", handler.ProcessPayment)
	r.GET("/payments/:order_id", handler.GetPayment)

	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Printf("Payment Service is running on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("payment server error: ", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("payment service forced shutdown: ", err)
	}
	log.Println("Payment Service stopped gracefully")
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
