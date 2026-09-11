package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/order-processing-system/notification-service/internal/repository"
)

type mockRepo struct {
	repository.Querier
	logs []repository.NotificationsLog
}

func (m *mockRepo) WithTx(ctx context.Context, fn func(q repository.Querier) error) error {
	return fn(m)
}

func (m *mockRepo) CreateNotificationLog(ctx context.Context, arg repository.CreateNotificationLogParams) (repository.NotificationsLog, error) {
	log := repository.NotificationsLog{
		ID:        uuid.New(),
		OrderID:   arg.OrderID,
		Type:      arg.Type,
		Channel:   arg.Channel,
		SentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	m.logs = append(m.logs, log)
	return log, nil
}

func (m *mockRepo) GetNotificationLogsByOrder(ctx context.Context, orderID uuid.UUID) ([]repository.NotificationsLog, error) {
	var res []repository.NotificationsLog
	for _, l := range m.logs {
		if l.OrderID == orderID {
			res = append(res, l)
		}
	}
	return res, nil
}

func TestNotificationService_ProcessEvent(t *testing.T) {
	repo := &mockRepo{}
	svc := NewNotificationService(repo)

	orderID := uuid.New().String()
	
	// Test DB logging
	err := svc.ProcessEvent(context.Background(), orderID, "ORDER_CREATED", `{"mock":"data"}`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repo.logs) != 1 {
		t.Errorf("expected 1 log entry")
	}

	// Test broadcast functionality with clients
	ch := make(chan string, 1)
	svc.RegisterClient(orderID, ch)
	
	err = svc.ProcessEvent(context.Background(), orderID, "STOCK_RESERVED", `{"mock":"data2"}`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	select {
	case msg := <-ch:
		if msg == "" {
			t.Errorf("expected message, got empty")
		}
	case <-time.After(time.Second):
		t.Errorf("timeout waiting for message on channel")
	}

	svc.UnregisterClient(orderID, ch)
}

func TestNotificationService_GetLogsByOrderID(t *testing.T) {
	repo := &mockRepo{}
	svc := NewNotificationService(repo)

	orderID := uuid.New()
	
	repo.CreateNotificationLog(context.Background(), repository.CreateNotificationLogParams{
		OrderID: orderID,
		Type:    "TEST",
		Channel: "ws",
	})

	logs, err := svc.GetLogsByOrderID(context.Background(), orderID.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log entry")
	}
}
