package main

import (
	"database/sql"
	"log"
	"order-service/internal/repository"
	transport "order-service/internal/transport/http"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/order_db?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to order_db:", err)
	}
	defer db.Close()

	repo := repository.NewPostgresOrderRepository(db)
	uc := usecase.NewOrderUseCase(repo, "http://localhost:8081")
	handler := transport.NewOrderHandler(uc)

	r := gin.Default()
	r.POST("/orders", handler.CreateOrder)
	r.GET("/orders/:id", handler.GetOrder)
	r.PATCH("/orders/:id/cancel", handler.CancelOrder)
	r.GET("/orders", handler.GetOrdersByAmount)

	log.Println("Order Service is running on port 8080...")
	r.Run(":8080")
}
