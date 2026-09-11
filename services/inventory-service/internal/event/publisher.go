package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/order-processing-system/pkg/events"
)

type Publisher interface {
	PublishStockReserved(ctx context.Context, payload events.StockReservedPayload) error
	PublishStockRejected(ctx context.Context, payload events.StockRejectedPayload) error
}

type natsPublisher struct {
	js jetstream.JetStream
}

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

func (p *natsPublisher) PublishStockReserved(ctx context.Context, payload events.StockReservedPayload) error {
	return p.publish(ctx, events.SubjectStockReserved, payload)
}

func (p *natsPublisher) PublishStockRejected(ctx context.Context, payload events.StockRejectedPayload) error {
	return p.publish(ctx, events.SubjectStockRejected, payload)
}
