package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/google/uuid"
	"github.com/order-processing-system/order-service/internal/repository"
	"github.com/order-processing-system/pkg/events"
)

// mockRepo is a simple mock for repository.Repository
type mockRepo struct {
	repository.Querier
	orders map[string]repository.Order
}

func (m *mockRepo) WithTx(ctx context.Context, fn func(q repository.Querier) error) error {
	// For testing, just run the function directly without actual tx overhead
	return fn(m)
}

func (m *mockRepo) CreateOrder(ctx context.Context, arg repository.CreateOrderParams) (repository.Order, error) {
	if m.orders == nil {
		m.orders = make(map[string]repository.Order)
	}
	order := repository.Order{
		ID:          arg.ID,
		CustomerID:  arg.CustomerID,
		Status:      arg.Status,
		TotalAmount: arg.TotalAmount,
		CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	m.orders[arg.ID.String()] = order
	return order, nil
}

func (m *mockRepo) CreateOrderItem(ctx context.Context, arg repository.CreateOrderItemParams) (repository.OrderItem, error) {
	return repository.OrderItem{}, nil
}

func (m *mockRepo) CreateSagaLog(ctx context.Context, arg repository.CreateSagaLogParams) (repository.SagaLog, error) {
	return repository.SagaLog{}, nil
}

func (m *mockRepo) GetOrder(ctx context.Context, id uuid.UUID) (repository.Order, error) {
	if order, exists := m.orders[id.String()]; exists {
		return order, nil
	}
	return repository.Order{}, errors.New("not found")
}

// mockPublisher is a simple mock for event.Publisher
type mockPublisher struct {
	publishedOrders []events.OrderCreatedPayload
}

func (m *mockPublisher) PublishOrderCreated(ctx context.Context, payload events.OrderCreatedPayload) error {
	m.publishedOrders = append(m.publishedOrders, payload)
	return nil
}
func (m *mockPublisher) PublishPaymentRequested(ctx context.Context, payload events.PaymentRequestedPayload) error {
	return nil
}
func (m *mockPublisher) PublishOrderCancelled(ctx context.Context, payload events.OrderCancelledPayload) error {
	return nil
}
func (m *mockPublisher) PublishOrderCompleted(ctx context.Context, payload events.OrderCompletedPayload) error {
	return nil
}

func TestOrderService_CreateOrder(t *testing.T) {
	repo := &mockRepo{}
	pub := &mockPublisher{}
	svc := NewOrderService(repo, pub)

	customerID := uuid.New().String()
	items := []events.OrderItem{
		{ProductID: uuid.New().String(), Quantity: 2, Price: 50.0},
		{ProductID: uuid.New().String(), Quantity: 1, Price: 100.0},
	}

	order, err := svc.CreateOrder(context.Background(), customerID, items)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if order.Status != "PENDING" {
		t.Errorf("expected status PENDING, got %s", order.Status)
	}

	if repository.NumericToFloat64(order.TotalAmount) != 200.0 {
		t.Errorf("expected total 200.0, got %f", repository.NumericToFloat64(order.TotalAmount))
	}

	if len(pub.publishedOrders) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(pub.publishedOrders))
	}

	evt := pub.publishedOrders[0]
	if evt.OrderID != order.ID.String() {
		t.Errorf("expected event OrderID to match order ID")
	}
}
