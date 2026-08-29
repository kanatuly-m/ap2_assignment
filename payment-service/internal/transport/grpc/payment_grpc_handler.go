package grpc

import (
	"context"
	"payment-service/internal/domain"
	"time"

	pb "github.com/kanatuly-m/order-payment-generated/payment"
	gogrpc "google.golang.org/grpc"
)

type PaymentGRPCHandler struct {
	useCase domain.PaymentUseCase
	pb.UnimplementedPaymentServiceServer
}

func NewPaymentGRPCHandler(uc domain.PaymentUseCase) *PaymentGRPCHandler {
	return &PaymentGRPCHandler{useCase: uc}
}

func (h *PaymentGRPCHandler) ProcessPayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {
	payment, err := h.useCase.ProcessPayment(req.OrderId, int64(req.Amount))
	if err != nil {
		return nil, err
	}

	return &pb.PaymentResponse{
		Success: payment.Status == "Authorized",
		Message: payment.Status,
	}, nil
}

func (h *PaymentGRPCHandler) SubscribeToOrderUpdates(req *pb.PaymentRequest, stream gogrpc.ServerStreamingServer[pb.PaymentResponse]) error {
	statuses := []string{"Pending", "Processing", "Completed"}

	for _, status := range statuses {
		time.Sleep(1 * time.Second)

		err := stream.Send(&pb.PaymentResponse{
			Success: status == "Completed",
			Message: status,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *PaymentGRPCHandler) ListPayments(ctx context.Context, req *pb.ListPaymentsRequest) (*pb.ListPaymentsResponse, error) {
	payments, err := h.useCase.ListPayments(req.Status)
	if err != nil {
		return nil, err
	}

	items := make([]*pb.PaymentItem, 0, len(payments))
	for _, p := range payments {
		items = append(items, &pb.PaymentItem{
			Id:            p.ID,
			OrderId:       p.OrderID,
			TransactionId: p.TransactionID,
			Amount:        p.Amount,
			Status:        p.Status,
		})
	}

	return &pb.ListPaymentsResponse{
		Payments: items,
	}, nil
}
