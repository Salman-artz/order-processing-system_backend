package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/order-processing-system/pkg/events"
)

// Publisher handles publishing events to NATS JetStream.
type Publisher interface {
	PublishOrderCreated(ctx context.Context, payload events.OrderCreatedPayload) error
	PublishPaymentRequested(ctx context.Context, payload events.PaymentRequestedPayload) error
	PublishOrderCancelled(ctx context.Context, payload events.OrderCancelledPayload) error
	PublishOrderCompleted(ctx context.Context, payload events.OrderCompletedPayload) error
}

type natsPublisher struct {
	js jetstream.JetStream
}

// NewPublisher creates a new Publisher using the provided JetStream context.
func NewPublisher(js jetstream.JetStream) Publisher {
	return &natsPublisher{js: js}
}

func (p *natsPublisher) publish(ctx context.Context, subject string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	_, err = p.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}
	return nil
}

func (p *natsPublisher) PublishOrderCreated(ctx context.Context, payload events.OrderCreatedPayload) error {
	return p.publish(ctx, events.SubjectOrderCreated, payload)
}

func (p *natsPublisher) PublishPaymentRequested(ctx context.Context, payload events.PaymentRequestedPayload) error {
	return p.publish(ctx, events.SubjectPaymentRequested, payload)
}

func (p *natsPublisher) PublishOrderCancelled(ctx context.Context, payload events.OrderCancelledPayload) error {
	return p.publish(ctx, events.SubjectOrderCancelled, payload)
}

func (p *natsPublisher) PublishOrderCompleted(ctx context.Context, payload events.OrderCompletedPayload) error {
	return p.publish(ctx, events.SubjectOrderCompleted, payload)
}
