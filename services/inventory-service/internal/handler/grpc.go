package handler

import (
	"context"

	"github.com/google/uuid"
	"github.com/order-processing-system/pkg/events"
	inventory "github.com/order-processing-system/proto/inventory"
	"github.com/order-processing-system/inventory-service/internal/repository"
	"github.com/order-processing-system/inventory-service/internal/service"
)

type InventoryGRPCServer struct {
	inventory.UnimplementedInventoryServiceServer
	svc  service.InventoryService
	repo repository.Repository
}

func NewInventoryGRPCServer(svc service.InventoryService, repo repository.Repository) *InventoryGRPCServer {
	return &InventoryGRPCServer{
		svc:  svc,
		repo: repo,
	}
}

func (s *InventoryGRPCServer) CheckStock(ctx context.Context, req *inventory.CheckStockRequest) (*inventory.CheckStockResponse, error) {
	var items []events.OrderItem
	for _, it := range req.Items {
		items = append(items, events.OrderItem{
			ProductID: it.ProductId,
			Quantity:  int(it.Quantity),
		})
	}

	available, reason := s.svc.CheckStock(ctx, items)
	return &inventory.CheckStockResponse{
		Available:    available,
		RejectReason: reason,
	}, nil
}

func (s *InventoryGRPCServer) GetProducts(ctx context.Context, req *inventory.GetProductsRequest) (*inventory.GetProductsResponse, error) {
	products, err := s.repo.ListProducts(ctx)
	if err != nil {
		return nil, err
	}

	var pbProducts []*inventory.ProductResponse
	for _, p := range products {
		pbProducts = append(pbProducts, &inventory.ProductResponse{
			ProductId:      p.ID.String(),
			Name:           p.Name,
			StockAvailable: p.StockAvailable,
			StockReserved:  p.StockReserved,
			Price:          repository.NumericToFloat64(p.Price),
		})
	}

	return &inventory.GetProductsResponse{
		Products: pbProducts,
		Total:    int32(len(products)),
	}, nil
}

func (s *InventoryGRPCServer) GetProductByID(ctx context.Context, req *inventory.GetProductByIDRequest) (*inventory.ProductResponse, error) {
	prodUUID, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, err
	}

	p, err := s.repo.GetProduct(ctx, prodUUID)
	if err != nil {
		return nil, err
	}

	return &inventory.ProductResponse{
		ProductId:      p.ID.String(),
		Name:           p.Name,
		StockAvailable: p.StockAvailable,
		StockReserved:  p.StockReserved,
		Price:          repository.NumericToFloat64(p.Price),
	}, nil
}
