-- name: GetProduct :one
SELECT * FROM products
WHERE id = $1 LIMIT 1;

-- name: ListProducts :many
SELECT * FROM products
ORDER BY name ASC;

-- name: ReserveStock :one
UPDATE products
SET stock_available = stock_available - $2,
    stock_reserved = stock_reserved + $2
WHERE id = $1 AND stock_available >= $2
RETURNING *;

-- name: ReleaseStock :one
UPDATE products
SET stock_available = stock_available + $2,
    stock_reserved = stock_reserved - $2
WHERE id = $1 AND stock_reserved >= $2
RETURNING *;

-- name: ConfirmStock :one
UPDATE products
SET stock_reserved = stock_reserved - $2
WHERE id = $1 AND stock_reserved >= $2
RETURNING *;

-- name: CreateStockReservation :one
INSERT INTO stock_reservations (order_id, product_id, quantity, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateStockReservationStatus :one
UPDATE stock_reservations
SET status = $2
WHERE id = $1
RETURNING *;

-- name: GetStockReservationsByOrder :many
SELECT * FROM stock_reservations
WHERE order_id = $1;
