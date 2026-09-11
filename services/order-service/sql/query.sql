-- name: CreateUser :one
INSERT INTO users (email, password_hash, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: CreateOrder :one
INSERT INTO orders (id, customer_id, status, total_amount)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2
WHERE id = $1
RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders
WHERE id = $1 LIMIT 1;

-- name: ListCustomerOrders :many
SELECT * FROM orders
WHERE customer_id = $1
ORDER BY created_at DESC;

-- name: ListAllOrders :many
SELECT * FROM orders
ORDER BY created_at DESC;

-- name: CreateOrderItem :one
INSERT INTO order_items (order_id, product_id, quantity, price)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListOrderItems :many
SELECT * FROM order_items
WHERE order_id = $1;

-- name: CreateSagaLog :one
INSERT INTO saga_logs (order_id, event_type, payload)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListSagaLogsByOrderID :many
SELECT * FROM saga_logs
WHERE order_id = $1
ORDER BY created_at ASC;
