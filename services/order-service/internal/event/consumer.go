package event

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"log"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/order-processing-system/pkg/events"
	"github.com/order-processing-system/order-service/internal/repository"
)

// Consumer handles incoming NATS JetStream events for the saga orchestration.
type Consumer struct {
	js        jetstream.JetStream
	repo      repository.Repository
	publisher Publisher
}

// NewConsumer creates a new event Consumer.
func NewConsumer(js jetstream.JetStream, repo repository.Repository, pub Publisher) *Consumer {
	return &Consumer{
		js:        js,
		repo:      repo,
		publisher: pub,
	}
}

// Start registers durable consumers for all relevant events.
func (c *Consumer) Start(ctx context.Context) error {
	subs := map[string]func(jetstream.Msg){
		events.SubjectStockReserved: c.handleStockReserved,
		events.SubjectStockRejected: c.handleStockRejected,
		events.SubjectPaymentSuccess: c.handlePaymentSuccess,
		events.SubjectPaymentFailed: c.handlePaymentFailed,
	}

	for subject, handler := range subs {
		cons, err := c.js.CreateOrUpdateConsumer(ctx, events.StreamName, jetstream.ConsumerConfig{
			Durable:       fmt.Sprintf("order-service-%s", strings.ReplaceAll(subject, ".", "_")),
			FilterSubject: subject,
			AckPolicy:     jetstream.AckExplicitPolicy,
			MaxDeliver:    5,
		})
		if err != nil {
			return fmt.Errorf("failed to create consumer for %s: %w", subject, err)
		}

		cc, err := cons.Consume(handler)
		if err != nil {
			return fmt.Errorf("failed to start consuming %s: %w", subject, err)
		}
		
		// Typically we'd keep track of cc to stop them on shutdown
		_ = cc
		log.Printf("Started consuming %s", subject)
	}

	return nil
}

func (c *Consumer) handleStockReserved(msg jetstream.Msg) {
	ctx := context.Background()
	var payload events.StockReservedPayload
	if err := json.Unmarshal(msg.Data(), &payload); err != nil {
		log.Printf("failed to unmarshal stock.reserved: %v", err)
		msg.Term() // Bad format, don't retry
		return
	}

	orderID, err := uuid.Parse(payload.OrderID)
	if err != nil {
		msg.Term()
		return
	}

	err = c.repo.WithTx(ctx, func(q repository.Querier) error {
		order, err := q.GetOrder(ctx, orderID)
		if err != nil {
			return err
		}
		if order.Status != "PENDING" {
			// Idempotency check: only process if status is PENDING
			return nil
		}

		_, err = q.UpdateOrderStatus(ctx, repository.UpdateOrderStatusParams{
			ID:     orderID,
			Status: "AWAITING_PAYMENT",
		})
		if err != nil {
			return err
		}

		// Log receipt of stock.reserved
		_, err = q.CreateSagaLog(ctx, repository.CreateSagaLogParams{
			OrderID:   orderID,
			EventType: events.SubjectStockReserved,
			Payload:   payloadToJSON(payload),
		})
		return err
	})

	if err != nil {
		log.Printf("failed to process stock.reserved: %v", err)
		msg.Nak()
		return
	}

	// Trigger next step: publish payment.requested
	order, _ := c.repo.GetOrder(ctx, orderID)
	
	payPayload := events.PaymentRequestedPayload{
		OrderID: orderID.String(),
		Amount:  repository.NumericToFloat64(order.TotalAmount),
	}
	
	// Assuming eventual consistency on publish failures for this simple project
	_ = c.publisher.PublishPaymentRequested(ctx, payPayload)
	
	// Log publishing payment.requested
	_ = c.repo.WithTx(ctx, func(q repository.Querier) error {
		_, err := q.CreateSagaLog(ctx, repository.CreateSagaLogParams{
			OrderID:   orderID,
			EventType: events.SubjectPaymentRequested,
			Payload:   payloadToJSON(payPayload),
		})
		return err
	})

	msg.Ack()
}

func (c *Consumer) handleStockRejected(msg jetstream.Msg) {
	ctx := context.Background()
	var payload events.StockRejectedPayload
	if err := json.Unmarshal(msg.Data(), &payload); err != nil {
		msg.Term()
		return
	}

	orderID, err := uuid.Parse(payload.OrderID)
	if err != nil {
		msg.Term()
		return
	}

	err = c.repo.WithTx(ctx, func(q repository.Querier) error {
		order, err := q.GetOrder(ctx, orderID)
		if err != nil {
			return err
		}
		if order.Status != "PENDING" {
			return nil
		}

		_, err = q.UpdateOrderStatus(ctx, repository.UpdateOrderStatusParams{
			ID:     orderID,
			Status: "CANCELLED",
		})
		if err != nil {
			return err
		}

		_, err = q.CreateSagaLog(ctx, repository.CreateSagaLogParams{
			OrderID:   orderID,
			EventType: events.SubjectStockRejected,
			Payload:   payloadToJSON(payload),
		})
		return err
	})

	if err != nil {
		msg.Nak()
		return
	}

	// Publish order.cancelled
	cancPayload := events.OrderCancelledPayload{
		OrderID: orderID.String(),
		Reason:  "Stock rejected: " + payload.Reason,
	}
	_ = c.publisher.PublishOrderCancelled(ctx, cancPayload)

	_ = c.repo.WithTx(ctx, func(q repository.Querier) error {
		_, err := q.CreateSagaLog(ctx, repository.CreateSagaLogParams{
			OrderID:   orderID,
			EventType: events.SubjectOrderCancelled,
			Payload:   payloadToJSON(cancPayload),
		})
		return err
	})

	msg.Ack()
}

func (c *Consumer) handlePaymentSuccess(msg jetstream.Msg) {
	ctx := context.Background()
	var payload events.PaymentSuccessPayload
	if err := json.Unmarshal(msg.Data(), &payload); err != nil {
		msg.Term()
		return
	}

	orderID, err := uuid.Parse(payload.OrderID)
	if err != nil {
		msg.Term()
		return
	}

	err = c.repo.WithTx(ctx, func(q repository.Querier) error {
		order, err := q.GetOrder(ctx, orderID)
		if err != nil {
			return err
		}
		if order.Status != "AWAITING_PAYMENT" {
			return nil
		}

		_, err = q.UpdateOrderStatus(ctx, repository.UpdateOrderStatusParams{
			ID:     orderID,
			Status: "COMPLETED",
		})
		if err != nil {
			return err
		}

		_, err = q.CreateSagaLog(ctx, repository.CreateSagaLogParams{
			OrderID:   orderID,
			EventType: events.SubjectPaymentSuccess,
			Payload:   payloadToJSON(payload),
		})
		return err
	})

	if err != nil {
		msg.Nak()
		return
	}

	// Publish order.completed
	compPayload := events.OrderCompletedPayload{
		OrderID: orderID.String(),
	}
	_ = c.publisher.PublishOrderCompleted(ctx, compPayload)

	_ = c.repo.WithTx(ctx, func(q repository.Querier) error {
		_, err := q.CreateSagaLog(ctx, repository.CreateSagaLogParams{
			OrderID:   orderID,
			EventType: events.SubjectOrderCompleted,
			Payload:   payloadToJSON(compPayload),
		})
		return err
	})

	msg.Ack()
}

func (c *Consumer) handlePaymentFailed(msg jetstream.Msg) {
	ctx := context.Background()
	var payload events.PaymentFailedPayload
	if err := json.Unmarshal(msg.Data(), &payload); err != nil {
		msg.Term()
		return
	}

	orderID, err := uuid.Parse(payload.OrderID)
	if err != nil {
		msg.Term()
		return
	}

	err = c.repo.WithTx(ctx, func(q repository.Querier) error {
		order, err := q.GetOrder(ctx, orderID)
		if err != nil {
			return err
		}
		if order.Status != "AWAITING_PAYMENT" {
			return nil
		}

		_, err = q.UpdateOrderStatus(ctx, repository.UpdateOrderStatusParams{
			ID:     orderID,
			Status: "CANCELLED",
		})
		if err != nil {
			return err
		}

		_, err = q.CreateSagaLog(ctx, repository.CreateSagaLogParams{
			OrderID:   orderID,
			EventType: events.SubjectPaymentFailed,
			Payload:   payloadToJSON(payload),
		})
		return err
	})

	if err != nil {
		msg.Nak()
		return
	}

	// Publish order.cancelled (inventory will release stock)
	cancPayload := events.OrderCancelledPayload{
		OrderID: orderID.String(),
		Reason:  "Payment failed: " + payload.Reason,
	}
	_ = c.publisher.PublishOrderCancelled(ctx, cancPayload)

	_ = c.repo.WithTx(ctx, func(q repository.Querier) error {
		_, err := q.CreateSagaLog(ctx, repository.CreateSagaLogParams{
			OrderID:   orderID,
			EventType: events.SubjectOrderCancelled,
			Payload:   payloadToJSON(cancPayload),
		})
		return err
	})

	msg.Ack()
}

func payloadToJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
