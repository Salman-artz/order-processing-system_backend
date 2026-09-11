package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/order-processing-system/pkg/events"
	"github.com/order-processing-system/inventory-service/internal/repository"
)

type mockRepo struct {
	repository.Querier
	products     map[string]repository.Product
	reservations map[string]repository.StockReservation
}

func (m *mockRepo) WithTx(ctx context.Context, fn func(q repository.Querier) error) error {
	return fn(m)
}

func (m *mockRepo) GetProduct(ctx context.Context, id uuid.UUID) (repository.Product, error) {
	if p, ok := m.products[id.String()]; ok {
		return p, nil
	}
	return repository.Product{}, errors.New("not found")
}

func (m *mockRepo) GetStockReservationsByOrder(ctx context.Context, orderID uuid.UUID) ([]repository.StockReservation, error) {
	var res []repository.StockReservation
	for _, r := range m.reservations {
		if r.OrderID == orderID {
			res = append(res, r)
		}
	}
	return res, nil
}

func (m *mockRepo) ReserveStock(ctx context.Context, arg repository.ReserveStockParams) (repository.Product, error) {
	p, ok := m.products[arg.ID.String()]
	if !ok {
		return repository.Product{}, errors.New("not found")
	}
	if p.StockAvailable < arg.StockAvailable {
		return repository.Product{}, errors.New("insufficient stock")
	}
	p.StockAvailable -= arg.StockAvailable
	p.StockReserved += arg.StockAvailable
	m.products[arg.ID.String()] = p
	return p, nil
}

func (m *mockRepo) ReleaseStock(ctx context.Context, arg repository.ReleaseStockParams) (repository.Product, error) {
	p, ok := m.products[arg.ID.String()]
	if !ok {
		return repository.Product{}, errors.New("not found")
	}
	p.StockAvailable += arg.StockAvailable
	p.StockReserved -= arg.StockAvailable
	m.products[arg.ID.String()] = p
	return p, nil
}

func (m *mockRepo) ConfirmStock(ctx context.Context, arg repository.ConfirmStockParams) (repository.Product, error) {
	p, ok := m.products[arg.ID.String()]
	if !ok {
		return repository.Product{}, errors.New("not found")
	}
	p.StockReserved -= arg.StockReserved
	m.products[arg.ID.String()] = p
	return p, nil
}

func (m *mockRepo) CreateStockReservation(ctx context.Context, arg repository.CreateStockReservationParams) (repository.StockReservation, error) {
	r := repository.StockReservation{
		ID:         uuid.New(),
		OrderID:    arg.OrderID,
		ProductID:  arg.ProductID,
		Quantity:   arg.Quantity,
		Status:     arg.Status,
		CreatedAt:  pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:  pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	if m.reservations == nil {
		m.reservations = make(map[string]repository.StockReservation)
	}
	m.reservations[r.ID.String()] = r
	return r, nil
}

func (m *mockRepo) UpdateStockReservationStatus(ctx context.Context, arg repository.UpdateStockReservationStatusParams) (repository.StockReservation, error) {
	r, ok := m.reservations[arg.ID.String()]
	if !ok {
		return repository.StockReservation{}, errors.New("not found")
	}
	r.Status = arg.Status
	m.reservations[arg.ID.String()] = r
	return r, nil
}

type mockPublisher struct {
	reserved []events.StockReservedPayload
	rejected []events.StockRejectedPayload
}

func (m *mockPublisher) PublishStockReserved(ctx context.Context, payload events.StockReservedPayload) error {
	m.reserved = append(m.reserved, payload)
	return nil
}

func (m *mockPublisher) PublishStockRejected(ctx context.Context, payload events.StockRejectedPayload) error {
	m.rejected = append(m.rejected, payload)
	return nil
}

func TestInventoryService_CheckStock(t *testing.T) {
	repo := &mockRepo{
		products: map[string]repository.Product{
			"00000000-0000-0000-0000-000000000001": {
				ID:             uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				StockAvailable: 10,
			},
		},
	}
	svc := NewInventoryService(repo, &mockPublisher{})

	ok, _ := svc.CheckStock(context.Background(), []events.OrderItem{
		{ProductID: "00000000-0000-0000-0000-000000000001", Quantity: 5},
	})
	if !ok {
		t.Errorf("expected true, got false")
	}
}

func TestInventoryService_ReserveStockSaga(t *testing.T) {
	repo := &mockRepo{
		products: map[string]repository.Product{
			"00000000-0000-0000-0000-000000000001": {
				ID:             uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				StockAvailable: 10,
			},
		},
	}
	pub := &mockPublisher{}
	svc := NewInventoryService(repo, pub)

	orderID := uuid.New().String()
	payload := events.OrderCreatedPayload{
		OrderID: orderID,
		Items: []events.OrderItem{
			{ProductID: "00000000-0000-0000-0000-000000000001", Quantity: 5},
		},
	}

	err := svc.ReserveStockSaga(context.Background(), payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestInventoryService_ReleaseStockSaga(t *testing.T) {
	orderID := uuid.New()
	prodID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	resID := uuid.New()

	repo := &mockRepo{
		products: map[string]repository.Product{
			prodID.String(): {
				ID:             prodID,
				StockAvailable: 5,
				StockReserved:  5,
			},
		},
		reservations: map[string]repository.StockReservation{
			resID.String(): {
				ID:        resID,
				OrderID:   orderID,
				ProductID: prodID,
				Quantity:  5,
				Status:    "RESERVED",
			},
		},
	}
	svc := NewInventoryService(repo, &mockPublisher{})

	err := svc.ReleaseStockSaga(context.Background(), events.OrderCancelledPayload{
		OrderID: orderID.String(),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.products[prodID.String()].StockAvailable != 10 {
		t.Errorf("expected 10 available, got %d", repo.products[prodID.String()].StockAvailable)
	}
	if repo.reservations[resID.String()].Status != "RELEASED" {
		t.Errorf("expected RELEASED, got %s", repo.reservations[resID.String()].Status)
	}
}

func TestInventoryService_ConfirmStockSaga(t *testing.T) {
	orderID := uuid.New()
	prodID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	resID := uuid.New()

	repo := &mockRepo{
		products: map[string]repository.Product{
			prodID.String(): {
				ID:             prodID,
				StockAvailable: 5,
				StockReserved:  5,
			},
		},
		reservations: map[string]repository.StockReservation{
			resID.String(): {
				ID:        resID,
				OrderID:   orderID,
				ProductID: prodID,
				Quantity:  5,
				Status:    "RESERVED",
			},
		},
	}
	svc := NewInventoryService(repo, &mockPublisher{})

	err := svc.ConfirmStockSaga(context.Background(), events.OrderCompletedPayload{
		OrderID: orderID.String(),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.products[prodID.String()].StockReserved != 0 {
		t.Errorf("expected 0 reserved, got %d", repo.products[prodID.String()].StockReserved)
	}
	if repo.reservations[resID.String()].Status != "CONFIRMED" {
		t.Errorf("expected CONFIRMED, got %s", repo.reservations[resID.String()].Status)
	}
}
