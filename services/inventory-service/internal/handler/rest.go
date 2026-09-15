package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/order-processing-system/inventory-service/internal/repository"
)

type InventoryRESTHandler struct {
	repo repository.Repository
}

func NewInventoryRESTHandler(repo repository.Repository) *InventoryRESTHandler {
	return &InventoryRESTHandler{repo: repo}
}

type ProductResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	StockAvailable int32   `json:"stock_available"`
	StockReserved  int32   `json:"stock_reserved"`
	Stock          int32   `json:"stock"`
	Price          float64 `json:"price"`
}

func formatProduct(p repository.Product) ProductResponse {
	return ProductResponse{
		ID:             p.ID.String(),
		Name:           p.Name,
		StockAvailable: p.StockAvailable,
		StockReserved:  p.StockReserved,
		Stock:          p.StockAvailable,
		Price:          repository.NumericToFloat64(p.Price),
	}
}

func (h *InventoryRESTHandler) ListProducts(c *fiber.Ctx) error {
	products, err := h.repo.ListProducts(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	res := make([]ProductResponse, 0, len(products))
	for _, p := range products {
		res = append(res, formatProduct(p))
	}

	return c.JSON(fiber.Map{
		"products": res,
		"total":    len(res),
	})
}

func (h *InventoryRESTHandler) CreateProduct(c *fiber.Ctx) error {
	type Request struct {
		Name  string  `json:"name"`
		Price float64 `json:"price"`
		Stock int32   `json:"stock"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Product name is required"})
	}
	if req.Price < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Price must be non-negative"})
	}
	if req.Stock < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Stock must be non-negative"})
	}

	p, err := h.repo.CreateProduct(c.Context(), repository.CreateProductParams{
		Name:           req.Name,
		StockAvailable: req.Stock,
		Price:          repository.Float64ToNumeric(req.Price),
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Product created successfully",
		"product": formatProduct(p),
	})
}

func (h *InventoryRESTHandler) UpdateProduct(c *fiber.Ctx) error {
	idStr := c.Params("id")
	uid, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid product ID"})
	}

	type Request struct {
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Product name is required"})
	}
	if req.Price < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Price must be non-negative"})
	}

	p, err := h.repo.UpdateProduct(c.Context(), repository.UpdateProductParams{
		ID:    uid,
		Name:  req.Name,
		Price: repository.Float64ToNumeric(req.Price),
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Product updated successfully",
		"product": formatProduct(p),
	})
}

func (h *InventoryRESTHandler) RestockProduct(c *fiber.Ctx) error {
	idStr := c.Params("id")
	uid, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid product ID"})
	}

	type Request struct {
		Quantity int32 `json:"quantity"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	if req.Quantity <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Quantity must be greater than zero"})
	}

	p, err := h.repo.RestockProduct(c.Context(), repository.RestockProductParams{
		ID:       uid,
		Quantity: req.Quantity,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Product restocked successfully",
		"product": formatProduct(p),
	})
}

func (h *InventoryRESTHandler) DeleteProduct(c *fiber.Ctx) error {
	idStr := c.Params("id")
	uid, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid product ID"})
	}

	if err := h.repo.DeleteProduct(c.Context(), uid); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Product deleted successfully",
	})
}
