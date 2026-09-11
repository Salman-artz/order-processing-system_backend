package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/order-processing-system/inventory-service/internal/repository"
)

type InventoryRESTHandler struct {
	repo repository.Repository
}

func NewInventoryRESTHandler(repo repository.Repository) *InventoryRESTHandler {
	return &InventoryRESTHandler{repo: repo}
}

func (h *InventoryRESTHandler) ListProducts(c *fiber.Ctx) error {
	products, err := h.repo.ListProducts(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if products == nil {
		products = make([]repository.Product, 0)
	}

	// Transform to response struct to handle pgtype.Numeric -> float64
	type ProductResponse struct {
		ID             string  `json:"id"`
		Name           string  `json:"name"`
		StockAvailable int32   `json:"stock_available"`
		StockReserved  int32   `json:"stock_reserved"`
		Price          float64 `json:"price"`
	}

	var res []ProductResponse
	for _, p := range products {
		res = append(res, ProductResponse{
			ID:             p.ID.String(),
			Name:           p.Name,
			StockAvailable: p.StockAvailable,
			StockReserved:  p.StockReserved,
			Price:          repository.NumericToFloat64(p.Price),
		})
	}

	if res == nil {
		res = make([]ProductResponse, 0)
	}

	return c.JSON(fiber.Map{
		"products": res,
		"total":    len(res),
	})
}
