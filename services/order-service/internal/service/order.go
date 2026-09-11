package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/order-processing-system/order-service/internal/event"
	"github.com/order-processing-system/order-service/internal/repository"
	"github.com/order-processing-system/pkg/events"
	"github.com/rs/zerolog/log"
)

// OrderService handles core business logic for orders.
type OrderService interface {
	CreateOrder(ctx context.Context, customerID string, items []events.OrderItem) (repository.Order, error)
	GetOrder(ctx context.Context, orderID string) (repository.Order, error)
	ListCustomerOrders(ctx context.Context, customerID string) ([]repository.Order, error)
	ListAllOrders(ctx context.Context) ([]repository.Order, error)
	GetOrderTimeline(ctx context.Context, orderID string) ([]repository.SagaLog, error)
}

type orderService struct {
	repo      repository.Repository
	publisher event.Publisher
}

// NewOrderService creates a new OrderService.
func NewOrderService(repo repository.Repository, publisher event.Publisher) OrderService {
	return &orderService{
		repo:      repo,
		publisher: publisher,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, customerID string, items []events.OrderItem) (repository.Order, error) {
	var createdOrder repository.Order
	orderID := uuid.New()

	// Calculate total amount
	var total float64
	for _, item := range items {
		total += float64(item.Quantity) * item.Price
	}

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return createdOrder, fmt.Errorf("invalid customer ID format")
	}

	err = s.repo.WithTx(ctx, func(q repository.Querier) error {
		// 1. Insert Order
		order, err := q.CreateOrder(ctx, repository.CreateOrderParams{
			ID:          orderID,
			CustomerID:  custUUID,
			Status:      "PENDING",
			TotalAmount: repository.Float64ToNumeric(total),
		})
		if err != nil {
			return err
		}
		createdOrder = order

		// 2. Insert Order Items
		for _, item := range items {
			prodUUID, err := uuid.Parse(item.ProductID)
			if err != nil {
				return fmt.Errorf("invalid product ID: %s", item.ProductID)
			}
			_, err = q.CreateOrderItem(ctx, repository.CreateOrderItemParams{
				OrderID:   orderID,
				ProductID: prodUUID,
				Quantity:  int32(item.Quantity),
				Price:     repository.Float64ToNumeric(item.Price),
			})
			if err != nil {
				return err
			}
		}

		// 3. Saga Log
		payloadRaw, _ := json.Marshal(map[string]interface{}{
			"order_id": orderID.String(),
			"customer": customerID,
			"items":    items,
			"total":    total,
		})
		if _, err := q.CreateSagaLog(ctx, repository.CreateSagaLogParams{
			OrderID:   orderID,
			EventType: events.SubjectOrderCreated,
			Payload:   payloadRaw,
		}); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return createdOrder, err
	}

	// 4. Publish NATS Event
	evtPayload := events.OrderCreatedPayload{
		OrderID:     orderID.String(),
		CustomerID:  customerID,
		Items:       items,
		TotalAmount: total,
	}
	if err := s.publisher.PublishOrderCreated(ctx, evtPayload); err != nil {
		log.Error().Err(err).Str("order_id", orderID.String()).Msg("Failed to publish order.created event")
		// Eventual consistency: we could have an outbox pattern here, but for this portfolio project,
		// we log and assume the event might need manual retry if broker is totally down.
	}

	return createdOrder, nil
}

func (s *orderService) GetOrder(ctx context.Context, orderID string) (repository.Order, error) {
	uid, err := uuid.Parse(orderID)
	if err != nil {
		return repository.Order{}, err
	}
	return s.repo.GetOrder(ctx, uid)
}

func (s *orderService) ListCustomerOrders(ctx context.Context, customerID string) ([]repository.Order, error) {
	uid, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListCustomerOrders(ctx, uid)
}

func (s *orderService) ListAllOrders(ctx context.Context) ([]repository.Order, error) {
	return s.repo.ListAllOrders(ctx)
}

func (s *orderService) GetOrderTimeline(ctx context.Context, orderID string) ([]repository.SagaLog, error) {
	uid, err := uuid.Parse(orderID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListSagaLogsByOrderID(ctx, uid)
}
