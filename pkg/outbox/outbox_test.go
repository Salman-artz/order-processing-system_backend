package outbox_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/order-processing-system/pkg/events"
	"github.com/order-processing-system/pkg/outbox"
	"github.com/stretchr/testify/assert"
)

func TestOutboxRecordSerialization(t *testing.T) {
	recordID := uuid.New()
	payload := events.OrderCreatedPayload{
		EventMetadata: events.EventMetadata{
			EventID:       "evt-test-123",
			CorrelationID: "corr-test-456",
			Timestamp:     time.Now().UTC(),
		},
		OrderID:     "ord-test-789",
		CustomerID:  "cust-test-012",
		TotalAmount: 1500000.0,
	}

	raw, err := json.Marshal(payload)
	assert.NoError(t, err)

	rec := outbox.Record{
		ID:         recordID,
		EventType:  events.SubjectOrderCreated,
		Payload:    raw,
		Status:     "PENDING",
		RetryCount: 0,
		CreatedAt:  time.Now(),
	}

	assert.Equal(t, "PENDING", rec.Status)
	assert.Equal(t, 0, rec.RetryCount)
	assert.Equal(t, events.SubjectOrderCreated, rec.EventType)

	var decoded events.OrderCreatedPayload
	err = json.Unmarshal(rec.Payload, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "ord-test-789", decoded.OrderID)
	assert.Equal(t, "corr-test-456", decoded.CorrelationID)
}
