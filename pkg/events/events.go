// Package events defines all NATS event subjects and payload structures
// shared across services. This is the single source of truth for the
// event schema defined in PRD Section 8.
package events

// ── NATS Subjects ────────────────────────────────────────────────────────────
// Keep these in sync with PRD Section 8 (Event Schema / NATS Subjects).

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

	// SubjectPaymentSuccess is published by Payment Service on successful mock payment.
	// Consumer: Order Service, Notification Service
	SubjectPaymentSuccess = "payment.success"

	// SubjectPaymentFailed is published by Payment Service on mock payment failure.
	// Consumer: Order Service, Notification Service
	SubjectPaymentFailed = "payment.failed"

	// SubjectOrderCancelled is published by Order Service for compensating transaction.
	// Consumer: Inventory Service (release stock), Notification Service
	SubjectOrderCancelled = "order.cancelled"

	// SubjectOrderCompleted is published by Order Service on successful completion.
	// Consumer: Notification Service
	SubjectOrderCompleted = "order.completed"
)

// ── JetStream Stream Name ─────────────────────────────────────────────────────
// All order-processing events are under one stream.
const StreamName = "ORDER_EVENTS"

// ── Payload Structures ────────────────────────────────────────────────────────

// OrderItem represents a single item in an order event payload.
type OrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// OrderCreatedPayload is the payload for order.created event.
type OrderCreatedPayload struct {
	OrderID     string      `json:"order_id"`
	CustomerID  string      `json:"customer_id"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"total_amount"`
}

// StockReservedPayload is the payload for stock.reserved event.
type StockReservedPayload struct {
	OrderID  string `json:"order_id"`
	Reserved bool   `json:"reserved"`
}

// StockRejectedPayload is the payload for stock.rejected event.
type StockRejectedPayload struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// PaymentRequestedPayload is the payload for payment.requested event.
type PaymentRequestedPayload struct {
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
}

// PaymentSuccessPayload is the payload for payment.success event.
type PaymentSuccessPayload struct {
	OrderID   string `json:"order_id"`
	PaymentID string `json:"payment_id"`
}

// PaymentFailedPayload is the payload for payment.failed event.
type PaymentFailedPayload struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// OrderCancelledPayload is the payload for order.cancelled event.
type OrderCancelledPayload struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// OrderCompletedPayload is the payload for order.completed event.
type OrderCompletedPayload struct {
	OrderID string `json:"order_id"`
}
