package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/order-processing-system/payment-service/internal/service"
	"github.com/order-processing-system/pkg/events"
)

type Consumer struct {
	js  jetstream.JetStream
	svc service.PaymentService
}

func NewConsumer(js jetstream.JetStream, svc service.PaymentService) *Consumer {
	return &Consumer{
		js:  js,
		svc: svc,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	subject := events.SubjectPaymentRequested
	cons, err := c.js.CreateOrUpdateConsumer(ctx, events.StreamName, jetstream.ConsumerConfig{
		Durable:       "payment-service-requested",
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer for %s: %w", subject, err)
	}

	_, err = cons.Consume(c.handlePaymentRequested)
	if err != nil {
		return fmt.Errorf("failed to start consuming %s: %w", subject, err)
	}
	log.Printf("Started consuming %s", subject)
	return nil
}

func (c *Consumer) handlePaymentRequested(msg jetstream.Msg) {
	ctx := context.Background()
	var payload events.PaymentRequestedPayload
	if err := json.Unmarshal(msg.Data(), &payload); err != nil {
		log.Printf("failed to unmarshal payment.requested: %v", err)
		msg.Term()
		return
	}

	if err := c.svc.ProcessPaymentSaga(ctx, payload); err != nil {
		log.Printf("failed to process payment.requested: %v", err)
		msg.Nak()
		return
	}
	msg.Ack()
}
