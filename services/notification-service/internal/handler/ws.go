package handler

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/order-processing-system/notification-service/internal/service"
)

type NotificationWSHandler struct {
	svc service.NotificationService
}

func NewNotificationWSHandler(svc service.NotificationService) *NotificationWSHandler {
	return &NotificationWSHandler{svc: svc}
}

// Websocket middleware to upgrade connection
func UpgradeMiddleware(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

func (h *NotificationWSHandler) HandleWS(c *websocket.Conn) {
	orderID := c.Params("id")
	if orderID == "" {
		c.Close()
		return
	}

	ch := make(chan string, 10)
	h.svc.RegisterClient(orderID, ch)
	defer h.svc.UnregisterClient(orderID, ch)

	// Send initial connection success message
	_ = c.WriteMessage(websocket.TextMessage, []byte(`{"status": "connected", "order_id": "`+orderID+`"}`))

	// Read loop to keep connection alive and detect client disconnects
	go func() {
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				log.Println("WS Read Error:", err)
				c.Close() // this will break the select loop below
				break
			}
		}
	}()

	// Write loop
	for {
		msg, ok := <-ch
		if !ok {
			break
		}
		if err := c.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			log.Println("WS Write Error:", err)
			break
		}
	}
}
