package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/order-processing-system/inventory-service/internal/repository"
	"github.com/order-processing-system/pkg/events"
)

type InventoryService interface {
	CheckStock(ctx context.Context, items []events.OrderItem) (bool, string)
	ReserveStockSaga(ctx context.Context, payload events.OrderCreatedPayload) error
	ReleaseStockSaga(ctx context.Context, payload events.OrderCancelledPayload) error
	ConfirmStockSaga(ctx context.Context, payload events.OrderCompletedPayload) error
}

type Publisher interface {
	PublishStockReserved(ctx context.Context, payload events.StockReservedPayload) error
	PublishStockRejected(ctx context.Context, payload events.StockRejectedPayload) error
}

type inventoryService struct {
	repo      repository.Repository
	publisher Publisher
}

func NewInventoryService(repo repository.Repository, publisher Publisher) InventoryService {
	return &inventoryService{
		repo:      repo,
		publisher: publisher,
	}
}

// CheckStock returns true if stock is available, false + reason if not. This is for sync gRPC check.
func (s *inventoryService) CheckStock(ctx context.Context, items []events.OrderItem) (bool, string) {
	for _, item := range items {
		prodUUID, err := uuid.Parse(item.ProductID)
		if err != nil {
			return false, "invalid product ID format"
		}

		prod, err := s.repo.GetProduct(ctx, prodUUID)
		if err != nil {
			return false, fmt.Sprintf("product %s not found", item.ProductID)
		}

		if prod.StockAvailable < int32(item.Quantity) {
			return false, fmt.Sprintf("insufficient stock for product %s", prod.Name)
		}
	}
	return true, ""
}

// ReserveStockSaga handles order.created by reserving stock and publishing result.
func (s *inventoryService) ReserveStockSaga(ctx context.Context, payload events.OrderCreatedPayload) error {
	orderID, err := uuid.Parse(payload.OrderID)
	if err != nil {
		return err
	}

	// Idempotency check: did we already reserve for this order?
	reservations, err := s.repo.GetStockReservationsByOrder(ctx, orderID)
	if err == nil && len(reservations) > 0 {
		return nil // Already processed
	}

	err = s.repo.WithTx(ctx, func(q repository.Querier) error {
		for _, item := range payload.Items {
			prodUUID, err := uuid.Parse(item.ProductID)
			if err != nil {
				return err
			}

			// Atomic reserve
			_, err = q.ReserveStock(ctx, repository.ReserveStockParams{
				ID:             prodUUID,
				StockAvailable: int32(item.Quantity), // the param is passed to SET stock_available = stock_available - $2
			})
			if err != nil {
				// E.g. constraint violation or not found
				return fmt.Errorf("failed to reserve product %s: %w", item.ProductID, err)
			}

			// Create reservation record
			_, err = q.CreateStockReservation(ctx, repository.CreateStockReservationParams{
				OrderID:   orderID,
				ProductID: prodUUID,
				Quantity:  int32(item.Quantity),
				Status:    "RESERVED",
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		// Stock reservation failed (likely insufficient stock at exact moment of processing)
		rejPayload := events.StockRejectedPayload{
			OrderID: payload.OrderID,
			Reason:  err.Error(),
		}
		return s.publisher.PublishStockRejected(ctx, rejPayload)
	}

	// Stock reserved successfully
	resPayload := events.StockReservedPayload{
		OrderID:  payload.OrderID,
		Reserved: true,
	}
	return s.publisher.PublishStockReserved(ctx, resPayload)
}

// ReleaseStockSaga handles order.cancelled by releasing reserved stock back.
func (s *inventoryService) ReleaseStockSaga(ctx context.Context, payload events.OrderCancelledPayload) error {
	orderID, err := uuid.Parse(payload.OrderID)
	if err != nil {
		return err
	}

	return s.repo.WithTx(ctx, func(q repository.Querier) error {
		reservations, err := q.GetStockReservationsByOrder(ctx, orderID)
		if err != nil {
			return err
		}

		for _, r := range reservations {
			if r.Status == "RESERVED" {
				// Update product stock (release)
				_, err = q.ReleaseStock(ctx, repository.ReleaseStockParams{
					ID:             r.ProductID,
					StockAvailable: r.Quantity, // SET stock_available = stock_available + $2, stock_reserved = stock_reserved - $2
				})
				if err != nil {
					return err
				}

				// Update reservation status
				_, err = q.UpdateStockReservationStatus(ctx, repository.UpdateStockReservationStatusParams{
					ID:     r.ID,
					Status: "RELEASED",
				})
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// ConfirmStockSaga handles order.completed by finalizing the stock deductions.
func (s *inventoryService) ConfirmStockSaga(ctx context.Context, payload events.OrderCompletedPayload) error {
	orderID, err := uuid.Parse(payload.OrderID)
	if err != nil {
		return err
	}

	return s.repo.WithTx(ctx, func(q repository.Querier) error {
		reservations, err := q.GetStockReservationsByOrder(ctx, orderID)
		if err != nil {
			return err
		}

		for _, r := range reservations {
			if r.Status == "RESERVED" {
				// Update product stock (confirm)
				_, err = q.ConfirmStock(ctx, repository.ConfirmStockParams{
					ID:            r.ProductID,
					StockReserved: r.Quantity, // SET stock_reserved = stock_reserved - $2
				})
				if err != nil {
					return err
				}

				// Update reservation status
				_, err = q.UpdateStockReservationStatus(ctx, repository.UpdateStockReservationStatusParams{
					ID:     r.ID,
					Status: "CONFIRMED",
				})
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}
