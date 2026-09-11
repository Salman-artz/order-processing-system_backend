package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/order-processing-system/pkg/events"
	"github.com/order-processing-system/payment-service/internal/repository"
)

type PaymentService interface {
	ProcessPaymentSaga(ctx context.Context, payload events.PaymentRequestedPayload) error
	GetPaymentsByOrderID(ctx context.Context, orderID string) ([]repository.Payment, error)
}

type Publisher interface {
	PublishPaymentSuccess(ctx context.Context, payload events.PaymentSuccessPayload) error
	PublishPaymentFailed(ctx context.Context, payload events.PaymentFailedPayload) error
}

type paymentService struct {
	repo      repository.Repository
	publisher Publisher
}

func NewPaymentService(repo repository.Repository, publisher Publisher) PaymentService {
	return &paymentService{
		repo:      repo,
		publisher: publisher,
	}
}

func (s *paymentService) ProcessPaymentSaga(ctx context.Context, payload events.PaymentRequestedPayload) error {
	orderUUID, err := uuid.Parse(payload.OrderID)
	if err != nil {
		return fmt.Errorf("invalid order id: %v", err)
	}

	// Idempotency check: Have we processed this payment already?
	payments, err := s.repo.GetPaymentsByOrder(ctx, orderUUID)
	if err == nil && len(payments) > 0 {
		// Already processed. Just return
		return nil
	}

	amountNumeric := repository.Float64ToNumeric(payload.Amount)

	// Create payment record as PENDING
	payment, err := s.repo.CreatePayment(ctx, repository.CreatePaymentParams{
		OrderID: orderUUID,
		Amount:  amountNumeric,
		Status:  "PENDING",
	})
	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	// Simulate payment processing delay
	time.Sleep(500 * time.Millisecond)

	// Mock rules for predictable failure during testing
	var isSuccess bool = true
	var failReason string

	if payload.Amount >= 10000000 { // 10 million
		isSuccess = false
		failReason = "Transaction limit exceeded"
	} else if payload.Amount == 8888 {
		isSuccess = false
		failReason = "Insufficient mock balance"
	}

	// Update payment status
	newStatus := "SUCCESS"
	simulatedReason := pgtype.Text{Valid: false}

	if !isSuccess {
		newStatus = "FAILED"
		simulatedReason = pgtype.Text{String: failReason, Valid: true}
	}

	_, err = s.repo.UpdatePaymentStatus(ctx, repository.UpdatePaymentStatusParams{
		ID:                     payment.ID,
		Status:                 newStatus,
		SimulatedFailureReason: simulatedReason,
	})
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	// Publish appropriate NATS event
	if isSuccess {
		return s.publisher.PublishPaymentSuccess(ctx, events.PaymentSuccessPayload{
			OrderID:   payload.OrderID,
			PaymentID: payment.ID.String(),
		})
	} else {
		return s.publisher.PublishPaymentFailed(ctx, events.PaymentFailedPayload{
			OrderID: payload.OrderID,
			Reason:  failReason,
		})
	}
}

func (s *paymentService) GetPaymentsByOrderID(ctx context.Context, orderID string) ([]repository.Payment, error) {
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order id: %v", err)
	}
	return s.repo.GetPaymentsByOrder(ctx, orderUUID)
}
