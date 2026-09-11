package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/order-processing-system/notification-service/internal/service"
)

type NotificationRESTHandler struct {
	svc service.NotificationService
}

func NewNotificationRESTHandler(svc service.NotificationService) *NotificationRESTHandler {
	return &NotificationRESTHandler{svc: svc}
}

func (h *NotificationRESTHandler) GetLogs(c *fiber.Ctx) error {
	orderID := c.Params("id")
	if orderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "order_id is required"})
	}

	logs, err := h.svc.GetLogsByOrderID(c.Context(), orderID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data": logs,
	})
}
