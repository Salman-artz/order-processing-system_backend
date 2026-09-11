package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/order-processing-system/pkg/events"
	"github.com/order-processing-system/payment-service/internal/repository"
)

type mockRepo struct {
	repository.Querier
	payments map[string]repository.Payment
}

func (m *mockRepo) WithTx(ctx context.Context, fn func(q repository.Querier) error) error {
	return fn(m)
}

func (m *mockRepo) GetPaymentsByOrder(ctx context.Context, orderID uuid.UUID) ([]repository.Payment, error) {
	var res []repository.Payment
	for _, p := range m.payments {
		if p.OrderID == orderID {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *mockRepo) CreatePayment(ctx context.Context, arg repository.CreatePaymentParams) (repository.Payment, error) {
	if m.payments == nil {
		m.payments = make(map[string]repository.Payment)
	}
	p := repository.Payment{
		ID:        uuid.New(),
		OrderID:   arg.OrderID,
		Amount:    arg.Amount,
		Status:    arg.Status,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	m.payments[p.ID.String()] = p
	return p, nil
}

func (m *mockRepo) UpdatePaymentStatus(ctx context.Context, arg repository.UpdatePaymentStatusParams) (repository.Payment, error) {
	p, ok := m.payments[arg.ID.String()]
	if !ok {
		return repository.Payment{}, errors.New("not found")
	}
	p.Status = arg.Status
	p.SimulatedFailureReason = arg.SimulatedFailureReason
	m.payments[arg.ID.String()] = p
	return p, nil
}

type mockPublisher struct {
	success []events.PaymentSuccessPayload
	failed  []events.PaymentFailedPayload
}

func (m *mockPublisher) PublishPaymentSuccess(ctx context.Context, payload events.PaymentSuccessPayload) error {
	m.success = append(m.success, payload)
	return nil
}

func (m *mockPublisher) PublishPaymentFailed(ctx context.Context, payload events.PaymentFailedPayload) error {
	m.failed = append(m.failed, payload)
	return nil
}

func TestPaymentService_ProcessPaymentSaga_Success(t *testing.T) {
	repo := &mockRepo{}
	pub := &mockPublisher{}
	svc := NewPaymentService(repo, pub)

	orderID := uuid.New().String()
	payload := events.PaymentRequestedPayload{
		OrderID: orderID,
		Amount:  150000.00,
	}

	err := svc.ProcessPaymentSaga(context.Background(), payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(pub.success) != 1 {
		t.Errorf("expected 1 success event")
	}

	payments, _ := repo.GetPaymentsByOrder(context.Background(), uuid.MustParse(orderID))
	if len(payments) != 1 || payments[0].Status != "SUCCESS" {
		t.Errorf("expected 1 SUCCESS payment")
	}
}

func TestPaymentService_ProcessPaymentSaga_Fail_Limit(t *testing.T) {
	repo := &mockRepo{}
	pub := &mockPublisher{}
	svc := NewPaymentService(repo, pub)

	orderID := uuid.New().String()
	payload := events.PaymentRequestedPayload{
		OrderID: orderID,
		Amount:  15000000.00, // > 10 million fails
	}

	err := svc.ProcessPaymentSaga(context.Background(), payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(pub.failed) != 1 {
		t.Errorf("expected 1 failed event")
	}

	payments, _ := repo.GetPaymentsByOrder(context.Background(), uuid.MustParse(orderID))
	if len(payments) != 1 || payments[0].Status != "FAILED" {
		t.Errorf("expected 1 FAILED payment")
	}
	if !payments[0].SimulatedFailureReason.Valid {
		t.Errorf("expected simulated failure reason to be populated")
	}
}

func TestPaymentService_ProcessPaymentSaga_Fail_SpecificBalance(t *testing.T) {
	repo := &mockRepo{}
	pub := &mockPublisher{}
	svc := NewPaymentService(repo, pub)

	orderID := uuid.New().String()
	payload := events.PaymentRequestedPayload{
		OrderID: orderID,
		Amount:  8888.0, // specific fail triggers
	}

	err := svc.ProcessPaymentSaga(context.Background(), payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(pub.failed) != 1 {
		t.Errorf("expected 1 failed event")
	}
}

func TestPaymentService_GetPayments(t *testing.T) {
	repo := &mockRepo{}
	pub := &mockPublisher{}
	svc := NewPaymentService(repo, pub)

	orderID := uuid.New()
	repo.CreatePayment(context.Background(), repository.CreatePaymentParams{
		OrderID: orderID,
		Amount:  repository.Float64ToNumeric(100.0),
		Status:  "SUCCESS",
	})

	payments, err := svc.GetPaymentsByOrderID(context.Background(), orderID.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(payments) != 1 {
		t.Errorf("expected 1 payment")
	}
}
