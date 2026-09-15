package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/order-processing-system/payment-service/internal/repository"
	"github.com/order-processing-system/payment-service/internal/service"
)

type PaymentRESTHandler struct {
	svc service.PaymentService
}

func NewPaymentRESTHandler(svc service.PaymentService) *PaymentRESTHandler {
	return &PaymentRESTHandler{svc: svc}
}

func (h *PaymentRESTHandler) GetPaymentsByOrder(c *fiber.Ctx) error {
	orderID := c.Params("order_id")
	if orderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "order_id is required"})
	}

	payments, err := h.svc.GetPaymentsByOrderID(c.Context(), orderID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if payments == nil {
		payments = make([]repository.Payment, 0)
	}

	type PaymentResponse struct {
		ID                     string  `json:"id"`
		OrderID                string  `json:"order_id"`
		Amount                 float64 `json:"amount"`
		Status                 string  `json:"status"`
		SimulatedFailureReason string  `json:"simulated_failure_reason,omitempty"`
		CreatedAt              string  `json:"created_at"`
	}

	var res []PaymentResponse
	for _, p := range payments {
		res = append(res, PaymentResponse{
			ID:                     p.ID.String(),
			OrderID:                p.OrderID.String(),
			Amount:                 repository.NumericToFloat64(p.Amount),
			Status:                 p.Status,
			SimulatedFailureReason: p.SimulatedFailureReason.String,
			CreatedAt:              p.CreatedAt.Time.String(),
		})
	}

	return c.JSON(fiber.Map{
		"data": res,
	})
}

// ── Midtrans Handlers ────────────────────────────────────────────────────────

type CreateSnapRequest struct {
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
	Email   string  `json:"email"`
}

func (h *PaymentRESTHandler) CreateSnap(c *fiber.Ctx) error {
	var req CreateSnapRequest
	if err := c.BodyParser(&req); err != nil || req.OrderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	snapResp, err := h.svc.CreateSnapTransaction(c.Context(), req.OrderID, req.Amount, req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(snapResp)
}

func (h *PaymentRESTHandler) MidtransNotification(c *fiber.Ctx) error {
	var notif service.MidtransNotification
	if err := c.BodyParser(&notif); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid notification payload"})
	}

	if err := h.svc.HandleMidtransNotification(c.Context(), notif); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": "Notification processed successfully",
	})
}

type SimulatePaymentRequest struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"` // settlement, expire, cancel, deny
}

func (h *PaymentRESTHandler) SimulatePayment(c *fiber.Ctx) error {
	var req SimulatePaymentRequest
	if err := c.BodyParser(&req); err != nil || req.OrderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid simulation payload"})
	}

	if req.Status == "" {
		req.Status = "settlement"
	}

	notif := service.MidtransNotification{
		OrderID:           req.OrderID,
		TransactionStatus: req.Status,
		FraudStatus:       "accept",
		StatusCode:        "200",
		PaymentType:       "qris",
		StatusMessage:     fmt.Sprintf("Simulated %s via Midtrans", req.Status),
	}

	if err := h.svc.HandleMidtransNotification(c.Context(), notif); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": fmt.Sprintf("Midtrans payment simulated as %s", req.Status),
	})
}
