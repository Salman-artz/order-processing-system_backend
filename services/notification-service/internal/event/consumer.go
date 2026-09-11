package event

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/order-processing-system/pkg/events"
	"github.com/order-processing-system/notification-service/internal/service"
)

type Consumer struct {
	js  jetstream.JetStream
	svc service.NotificationService
}

func NewConsumer(js jetstream.JetStream, svc service.NotificationService) *Consumer {
	return &Consumer{
		js:  js,
		svc: svc,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	// Consume everything on the stream
	cons, err := c.js.CreateOrUpdateConsumer(ctx, events.StreamName, jetstream.ConsumerConfig{
		Durable:       "notification-service-all",
		FilterSubject: ">", // Matches everything in the stream
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
	})
	if err != nil {
		return err
	}

	_, err = cons.Consume(c.handleEvent)
	if err != nil {
		return err
	}
	log.Printf("Started consuming all events for notification")
	return nil
}

func (c *Consumer) handleEvent(msg jetstream.Msg) {
	subject := msg.Subject()
	ctx := context.Background()

	// Extract orderID from payload since every event has it
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Data(), &payload); err != nil {
		log.Printf("failed to unmarshal for subject %s: %v", subject, err)
		msg.Ack() // Ack to skip bad format
		return
	}

	var orderID string
	if val, ok := payload["order_id"].(string); ok {
		orderID = val
	} else if val, ok := payload["OrderID"].(string); ok { // Some structs might use camelCase/PascalCase if no json tag
		orderID = val
	}

	if orderID == "" {
		// Can't notify without order ID
		msg.Ack()
		return
	}

	// Create a generic message
	message := string(msg.Data())
	eventType := strings.ToUpper(strings.ReplaceAll(subject, ".", "_"))

	_ = c.svc.ProcessEvent(ctx, orderID, eventType, message)

	msg.Ack()
}
