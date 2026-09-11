package handler

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/order-processing-system/pkg/events"
	"github.com/order-processing-system/order-service/internal/repository"
	"github.com/order-processing-system/order-service/internal/service"
)

type OrderHandler struct {
	svc service.OrderService
}

func NewOrderHandler(svc service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

type createOrderRequest struct {
	Items []events.OrderItem `json:"items"`
}

func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
	var req createOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	if len(req.Items) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Order must contain at least one item"})
	}

	customerID := c.Locals("user_id").(string)

	order, err := h.svc.CreateOrder(c.Context(), customerID, req.Items)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Order created successfully",
		"order":   order,
	})
}

func (h *OrderHandler) GetOrder(c *fiber.Ctx) error {
	orderID := c.Params("id")
	customerID := c.Locals("user_id").(string)
	roleStr := fmt.Sprintf("%v", c.Locals("role")); role := roleStr // from jwtutil.Role, but stored as string in locals sometimes

	order, err := h.svc.GetOrder(c.Context(), orderID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Order not found"})
	}

	// Simple authorization: only admin or the order owner can view it
	if string(role) != "admin" && order.CustomerID.String() != customerID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}

	return c.JSON(order)
}

func (h *OrderHandler) ListCustomerOrders(c *fiber.Ctx) error {
	customerID := c.Locals("user_id").(string)

	orders, err := h.svc.ListCustomerOrders(c.Context(), customerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if orders == nil {
		orders = make([]repository.Order, 0) // Return empty array instead of null
	}

	return c.JSON(fiber.Map{
		"orders": orders,
	})
}

// Admin endpoints

func (h *OrderHandler) ListAllOrders(c *fiber.Ctx) error {
	orders, err := h.svc.ListAllOrders(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if orders == nil {
		orders = make([]repository.Order, 0)
	}

	return c.JSON(fiber.Map{
		"orders": orders,
	})
}

func (h *OrderHandler) GetOrderTimeline(c *fiber.Ctx) error {
	orderID := c.Params("id")

	logs, err := h.svc.GetOrderTimeline(c.Context(), orderID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if logs == nil {
		logs = make([]repository.SagaLog, 0)
	}

	return c.JSON(fiber.Map{
		"timeline": logs,
	})
}
