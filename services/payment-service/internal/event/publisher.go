package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/order-processing-system/pkg/events"
)

type natsPublisher struct {
	js jetstream.JetStream
}

func NewPublisher(js jetstream.JetStream) *natsPublisher {
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

func (p *natsPublisher) PublishPaymentSuccess(ctx context.Context, payload events.PaymentSuccessPayload) error {
	return p.publish(ctx, events.SubjectPaymentSuccess, payload)
}

func (p *natsPublisher) PublishPaymentFailed(ctx context.Context, payload events.PaymentFailedPayload) error {
	return p.publish(ctx, events.SubjectPaymentFailed, payload)
}
