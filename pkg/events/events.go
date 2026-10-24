// Package events defines all NATS event subjects and payload structures
// shared across services. This is the single source of truth for the
// WarehouseFlow event schema and Saga Choreography.
package events

import "time"

// ─── NATS Subjects ────────────────────────────────────────────────────────────
// Keep these synchronized across all microservices and JetStream consumers.

const (
	// SubjectOrderCreated is published by Order Service when a new order is placed.
	// Consumer: Inventory Service
	SubjectOrderCreated = "order.created"

	// SubjectStockReserved is published by Inventory Service on successful reservation.
	// Consumer: Order Service
	SubjectStockReserved = "stock.reserved"

	// SubjectStockRejected is published by Inventory Service when stock is insufficient.
	// Consumer: Order Service
	SubjectStockRejected = "stock.rejected"

	// SubjectPaymentRequested is published by Order Service to trigger payment.
	// Consumer: Payment Service
	SubjectPaymentRequested = "payment.requested"

	// SubjectPaymentSuccess is published by Payment Service on successful payment.
	// Consumer: Order Service, Notification Service
	SubjectPaymentSuccess = "payment.success"

	// SubjectPaymentFailed is published by Payment Service on payment failure.
	// Consumer: Order Service, Notification Service
	SubjectPaymentFailed = "payment.failed"

	// SubjectOrderConfirmed is published when payment succeeds and order enters fulfillment.
	// Consumer: Inventory Service (picking queue), Notification Service
	SubjectOrderConfirmed = "order.confirmed"

	// SubjectOrderPicking is published when warehouse staff begins picking items.
	// Consumer: Notification Service
	SubjectOrderPicking = "order.picking"

	// SubjectOrderPacked is published when picked items are verified and packed into shipment.
	// Consumer: Notification Service
	SubjectOrderPacked = "order.packed"

	// SubjectOrderDispatched is published when package leaves warehouse for dispatch.
	// Consumer: Notification Service
	SubjectOrderDispatched = "order.dispatched"

	// SubjectOrderCompleted is published when delivery/fulfillment is finalized.
	// Consumer: Inventory Service (final deduction), Notification Service
	SubjectOrderCompleted = "order.completed"

	// SubjectOrderCancelled is published for compensating transactions.
	// Consumer: Inventory Service (release stock), Notification Service
	SubjectOrderCancelled = "order.cancelled"

	// SubjectStockReleased is published by Inventory Service after compensation completes.
	// Consumer: Order Service, Notification Service
	SubjectStockReleased = "stock.released"

	// SubjectStockAdjusted is published when warehouse staff manually adjusts inventory.
	// Consumer: Notification Service
	SubjectStockAdjusted = "stock.adjusted"

	// SubjectStockReceived is published when inbound shipment/receiving is confirmed.
	// Consumer: Notification Service
	SubjectStockReceived = "stock.received"
)

// ─── JetStream Stream Name ────────────────────────────────────────────────────
const StreamName = "ORDER_EVENTS"

// ─── Event Metadata (Distributed Context Propagation) ─────────────────────────

// EventMetadata encapsulates distributed tracing and causality context.
type EventMetadata struct {
	EventID       string    `json:"event_id,omitempty"`
	SagaID        string    `json:"saga_id,omitempty"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"`
	Timestamp     time.Time `json:"timestamp,omitempty"`
	SourceService string    `json:"source_service,omitempty"`
}

// ─── Payload Structures ───────────────────────────────────────────────────────

// OrderItem represents a single item in an order event payload.
type OrderItem struct {
	ProductID string  `json:"product_id"`
	SKU       string  `json:"sku,omitempty"`
	Name      string  `json:"name,omitempty"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// OrderCreatedPayload is the payload for order.created event.
type OrderCreatedPayload struct {
	EventMetadata
	OrderID     string      `json:"order_id"`
	CustomerID  string      `json:"customer_id"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"total_amount"`
}

// StockReservedPayload is the payload for stock.reserved event.
type StockReservedPayload struct {
	EventMetadata
	OrderID  string `json:"order_id"`
	Reserved bool   `json:"reserved"`
}

// StockRejectedPayload is the payload for stock.rejected event.
type StockRejectedPayload struct {
	EventMetadata
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// PaymentRequestedPayload is the payload for payment.requested event.
type PaymentRequestedPayload struct {
	EventMetadata
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
}

// PaymentSuccessPayload is the payload for payment.success event.
type PaymentSuccessPayload struct {
	EventMetadata
	OrderID       string `json:"order_id"`
	PaymentID     string `json:"payment_id"`
	TransactionID string `json:"transaction_id,omitempty"`
}

// PaymentFailedPayload is the payload for payment.failed event.
type PaymentFailedPayload struct {
	EventMetadata
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// OrderFulfillmentPayload is the payload for order fulfillment stage events
// (order.confirmed, order.picking, order.packed, order.dispatched).
type OrderFulfillmentPayload struct {
	EventMetadata
	OrderID       string `json:"order_id"`
	Status        string `json:"status"`
	WarehouseZone string `json:"warehouse_zone,omitempty"`
	OperatorID    string `json:"operator_id,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

// OrderCancelledPayload is the payload for order.cancelled event.
type OrderCancelledPayload struct {
	EventMetadata
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// OrderCompletedPayload is the payload for order.completed event.
type OrderCompletedPayload struct {
	EventMetadata
	OrderID string `json:"order_id"`
}

// StockMovementPayload is the payload for stock.adjusted and stock.received events.
type StockMovementPayload struct {
	EventMetadata
	MovementID   string `json:"movement_id"`
	ProductID    string `json:"product_id"`
	SKU          string `json:"sku"`
	MovementType string `json:"movement_type"` // INBOUND, OUTBOUND, ADJUSTMENT, TRANSFER
	Quantity     int    `json:"quantity"`
	Reason       string `json:"reason"`
}
