package service

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/order-processing-system/notification-service/internal/repository"
)

type NotificationService interface {
	ProcessEvent(ctx context.Context, orderID, eventType, message string) error
	RegisterClient(orderID string, ch chan string)
	UnregisterClient(orderID string, ch chan string)
	GetLogsByOrderID(ctx context.Context, orderID string) ([]repository.NotificationsLog, error)
}

type notificationService struct {
	repo    repository.Repository
	mu      sync.RWMutex
	clients map[string]map[chan string]bool
}

func NewNotificationService(repo repository.Repository) NotificationService {
	return &notificationService{
		repo:    repo,
		clients: make(map[string]map[chan string]bool),
	}
}

func (s *notificationService) ProcessEvent(ctx context.Context, orderID, eventType, message string) error {
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return fmt.Errorf("invalid order id: %v", err)
	}

	// 1. Log to DB
	_, err = s.repo.CreateNotificationLog(ctx, repository.CreateNotificationLogParams{
		OrderID: orderUUID,
		Type:    eventType,
		Channel: "websocket",
	})
	if err != nil {
		log.Printf("Failed to insert notification log: %v", err)
	}

	// Simulate Email Sending (Mock)
	log.Printf("[EMAIL MOCK] Sending email notification for Order %s | Event: %s", orderID, eventType)

	// 2. Broadcast via WebSocket
	var status string
	switch eventType {
	case "ORDER_CREATED":
		status = "PENDING"
	case "STOCK_RESERVED", "PAYMENT_REQUESTED":
		status = "RESERVED"
	case "PAYMENT_SUCCESS":
		status = "PAID"
	case "ORDER_COMPLETED":
		status = "COMPLETED"
	case "ORDER_CANCELLED", "PAYMENT_FAILED", "STOCK_REJECTED":
		status = "CANCELLED"
	default:
		status = "PENDING"
	}

	s.broadcast(orderID, fmt.Sprintf(`{"event": "%s", "status": "%s", "order_id": "%s"}`, eventType, status, orderID))
	return nil
}

func (s *notificationService) broadcast(orderID, message string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if clients, ok := s.clients[orderID]; ok {
		for ch := range clients {
			// Non-blocking send
			select {
			case ch <- message:
			default:
				// Channel is full or blocked, drop message for this client
			}
		}
	}
}

func (s *notificationService) RegisterClient(orderID string, ch chan string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[orderID]; !ok {
		s.clients[orderID] = make(map[chan string]bool)
	}
	s.clients[orderID][ch] = true
}

func (s *notificationService) UnregisterClient(orderID string, ch chan string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if clients, ok := s.clients[orderID]; ok {
		delete(clients, ch)
		if len(clients) == 0 {
			delete(s.clients, orderID)
		}
	}
}

func (s *notificationService) GetLogsByOrderID(ctx context.Context, orderID string) ([]repository.NotificationsLog, error) {
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order id: %v", err)
	}
	return s.repo.GetNotificationLogsByOrder(ctx, orderUUID)
}
