package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"ap2_assignment/shared/config"
	cachepkg "order-service/internal/cache"
	"order-service/internal/client"
	"order-service/internal/repository"
	transport "order-service/internal/transport/http"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	databaseURL := config.GetEnv("DATABASE_URL", "postgres://user:password@localhost:5432/order_db?sslmode=disable")
	paymentURL := config.GetEnv("PAYMENT_SERVICE_URL", "http://localhost:8081")
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")
	port := config.GetEnv("PORT", "8080")
	cacheTTL := secondsFromEnv("CACHE_TTL_SECONDS", 300)
	rateLimit := intFromEnv("RATE_LIMIT", 10)
	rateWindow := secondsFromEnv("RATE_WINDOW_SECONDS", 60)

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal("failed to open order database: ", err)
	}
	defer db.Close()

	if err := waitForDB(ctx, db); err != nil {
		log.Fatal("order database is not ready: ", err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer redisClient.Close()
	if err := waitForRedis(ctx, redisClient); err != nil {
		log.Fatal("redis is not ready for order service: ", err)
	}

	repo := repository.NewPostgresOrderRepository(db)
	orderCache := cachepkg.NewRedisOrderCache(redisClient)
	paymentGateway := client.NewHTTPPaymentClient(paymentURL, 5*time.Second)
	uc := usecase.NewOrderUseCase(repo, paymentGateway, orderCache, cacheTTL)
	handler := transport.NewOrderHandler(uc)

	r := gin.Default()
	r.Use(transport.RateLimiter(redisClient, rateLimit, rateWindow))
	r.POST("/orders", handler.CreateOrder)
	r.GET("/orders/:id", handler.GetOrder)
	r.PATCH("/orders/:id/cancel", handler.CancelOrder)
	r.GET("/orders", handler.GetOrdersByAmount)

	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Printf("Order Service is running on port %s", port)
		log.Printf("Order cache TTL: %s | Rate limit: %d requests per %s", cacheTTL, rateLimit, rateWindow)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("order server error: ", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("order service forced shutdown: ", err)
	}
	log.Println("Order Service stopped gracefully")
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
