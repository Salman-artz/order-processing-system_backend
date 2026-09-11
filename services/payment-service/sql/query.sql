-- name: CreatePayment :one
INSERT INTO payments (order_id, amount, status)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdatePaymentStatus :one
UPDATE payments
SET status = $2, simulated_failure_reason = $3
WHERE id = $1
RETURNING *;

-- name: GetPaymentsByOrder :many
SELECT * FROM payments
WHERE order_id = $1
ORDER BY created_at ASC;

-- name: GetPaymentByID :one
SELECT * FROM payments
WHERE id = $1;
