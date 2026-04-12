package main

import (
	"database/sql"
	"log"
	"payment-service/internal/repository"
	transport "payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/payment_db?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to payment_db:", err)
	}
	defer db.Close()

	repo := repository.NewPostgresPaymentRepository(db)
	uc := usecase.NewPaymentUseCase(repo)
	handler := transport.NewPaymentHandler(uc)

	r := gin.Default()
	r.POST("/payments", handler.ProcessPayment)
	r.GET("/payments/:order_id", handler.GetPayment)

	log.Println("Payment Service is running on port 8081...")
	r.Run(":8081")
}
