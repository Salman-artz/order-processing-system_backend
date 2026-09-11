package event

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"log"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/order-processing-system/pkg/events"
	"github.com/order-processing-system/inventory-service/internal/service"
)

type Consumer struct {
	js  jetstream.JetStream
	svc service.InventoryService
}

func NewConsumer(js jetstream.JetStream, svc service.InventoryService) *Consumer {
	return &Consumer{
		js:  js,
		svc: svc,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	subs := map[string]func(jetstream.Msg){
		events.SubjectOrderCreated:   c.handleOrderCreated,
		events.SubjectOrderCancelled: c.handleOrderCancelled,
		events.SubjectOrderCompleted: c.handleOrderCompleted,
	}

	for subject, handler := range subs {
		cons, err := c.js.CreateOrUpdateConsumer(ctx, events.StreamName, jetstream.ConsumerConfig{
			Durable:       fmt.Sprintf("inventory-service-%s", strings.ReplaceAll(subject, ".", "_")),
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
		_ = cc
		log.Printf("Started consuming %s", subject)
	}

	return nil
}

func (c *Consumer) handleOrderCreated(msg jetstream.Msg) {
	ctx := context.Background()
	var payload events.OrderCreatedPayload
	if err := json.Unmarshal(msg.Data(), &payload); err != nil {
		msg.Term()
		return
	}

	if err := c.svc.ReserveStockSaga(ctx, payload); err != nil {
		log.Printf("failed to handle order.created: %v", err)
		msg.Nak()
		return
	}
	msg.Ack()
}

func (c *Consumer) handleOrderCancelled(msg jetstream.Msg) {
	ctx := context.Background()
	var payload events.OrderCancelledPayload
	if err := json.Unmarshal(msg.Data(), &payload); err != nil {
		msg.Term()
		return
	}

	if err := c.svc.ReleaseStockSaga(ctx, payload); err != nil {
		log.Printf("failed to handle order.cancelled: %v", err)
		msg.Nak()
		return
	}
	msg.Ack()
}

func (c *Consumer) handleOrderCompleted(msg jetstream.Msg) {
	ctx := context.Background()
	var payload events.OrderCompletedPayload
	if err := json.Unmarshal(msg.Data(), &payload); err != nil {
		msg.Term()
		return
	}

	if err := c.svc.ConfirmStockSaga(ctx, payload); err != nil {
		log.Printf("failed to handle order.completed: %v", err)
		msg.Nak()
		return
	}
	msg.Ack()
}
