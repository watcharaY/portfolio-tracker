-- Brokers: one query owns the deposit/withdraw sums. Get reads it with id set,
-- so List and Get cannot compute them differently. SUM(bigint) is NUMERIC in
-- Postgres; the ::BIGINT cast keeps it an int64.
-- name: ListBrokers :many
SELECT sqlc.embed(b),
       COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'deposit'),  0)::BIGINT AS deposits,
       COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'withdraw'), 0)::BIGINT AS withdrawals
FROM brokers b
LEFT JOIN broker_transactions t ON t.broker_id = b.id
WHERE sqlc.narg(id)::BIGINT IS NULL OR b.id = sqlc.narg(id)
GROUP BY b.id
ORDER BY b.name;

-- name: CreateBroker :one
INSERT INTO brokers (name, current_value, note)
VALUES ($1, $2, $3)
RETURNING id;

-- name: UpdateBroker :one
UPDATE brokers
SET name = $1, current_value = $2, note = $3, updated_at = now()
WHERE id = $4
RETURNING id;

-- name: DeleteBroker :one
DELETE FROM brokers WHERE id = $1 RETURNING id;

-- name: ListBrokerTransactions :many
SELECT sqlc.embed(t), b.name AS broker_name
FROM broker_transactions t
JOIN brokers b ON b.id = t.broker_id
WHERE sqlc.narg(broker_id)::BIGINT IS NULL OR t.broker_id = sqlc.narg(broker_id)
ORDER BY t.txn_date DESC NULLS LAST, t.id DESC;

-- name: GetBrokerTransaction :one
SELECT sqlc.embed(t), b.name AS broker_name
FROM broker_transactions t
JOIN brokers b ON b.id = t.broker_id
WHERE t.id = $1;

-- name: CreateBrokerTransaction :one
INSERT INTO broker_transactions (broker_id, txn_date, type, amount, note)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: UpdateBrokerTransaction :one
UPDATE broker_transactions
SET broker_id = $1, txn_date = $2, type = $3, amount = $4, note = $5, updated_at = now()
WHERE id = $6
RETURNING id;

-- name: DeleteBrokerTransaction :one
DELETE FROM broker_transactions WHERE id = $1 RETURNING id;
