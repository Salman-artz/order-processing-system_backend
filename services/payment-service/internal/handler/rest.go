package handler

import (
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
