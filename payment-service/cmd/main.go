package main

import (
	"database/sql"
	"google.golang.org/grpc"
	"log"
	"net"
	"payment-service/internal/repository"
	transport "payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
	pb "github.com/kanatuly-m/order-payment-generated/payment"
	_ "github.com/lib/pq"
	grpcTransport "payment-service/internal/transport/grpc"
)

func main() {
	db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/payment_db?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to payment_db:", err)
	}
	defer db.Close()

	repo := repository.NewPostgresPaymentRepository(db)
	uc := usecase.NewPaymentUseCase(repo)
	grpcHandler := grpcTransport.NewPaymentGRPCHandler(uc)
	handler := transport.NewPaymentHandler(uc)

	r := gin.Default()
	r.POST("/payments", handler.ProcessPayment)
	r.GET("/payments/:order_id", handler.GetPayment)

	log.Println("Payment Service is running on port 8081...")
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatal(err)
		}

		grpcServer := grpc.NewServer()

		pb.RegisterPaymentServiceServer(grpcServer, grpcHandler)

		log.Println("gRPC server running on :50051")

		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()
	r.Run(":8081")
}
