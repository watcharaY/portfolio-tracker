-- Insurers: one query owns the payment count and total; Get reads it with id set.
-- name: ListInsurers :many
SELECT sqlc.embed(i),
       COUNT(p.id)::INT                AS payments,
       COALESCE(SUM(p.premium), 0)::BIGINT AS total_paid
FROM insurers i
LEFT JOIN insurance_payments p ON p.insurer_id = i.id
WHERE sqlc.narg(id)::BIGINT IS NULL OR i.id = sqlc.narg(id)
GROUP BY i.id
ORDER BY i.company;

-- name: CreateInsurer :one
INSERT INTO insurers (company, note) VALUES ($1, $2) RETURNING id;

-- name: UpdateInsurer :one
UPDATE insurers SET company = $1, note = $2, updated_at = now() WHERE id = $3 RETURNING id;

-- name: DeleteInsurer :one
DELETE FROM insurers WHERE id = $1 RETURNING id;

-- The one definition of payment_no. ROW_NUMBER runs over each insurer's whole
-- history before any filter applies, so a payment keeps its number whether the
-- caller filters by year, by insurer, or by id. The repo's Get reads this query
-- with only id set, so List and Get cannot disagree on payment_no.
-- name: ListInsurancePayments :many
SELECT x.id, x.insurer_id, x.company, x.payment_no, x.year, x.premium, x.note, x.created_at, x.updated_at
FROM (
    SELECT p.id, p.insurer_id, i.company,
           ROW_NUMBER() OVER (PARTITION BY p.insurer_id ORDER BY p.year NULLS FIRST, p.id)::INT AS payment_no,
           p.year, p.premium, p.note, p.created_at, p.updated_at
    FROM insurance_payments p
    JOIN insurers i ON i.id = p.insurer_id
) x
WHERE (sqlc.narg(id)::BIGINT IS NULL OR x.id = sqlc.narg(id))
  AND (sqlc.narg(insurer_id)::BIGINT IS NULL OR x.insurer_id = sqlc.narg(insurer_id))
  AND (sqlc.narg(year)::INT IS NULL OR x.year = sqlc.narg(year))
ORDER BY x.company, x.payment_no;

-- name: CreateInsurancePayment :one
INSERT INTO insurance_payments (insurer_id, year, premium, note)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: UpdateInsurancePayment :one
UPDATE insurance_payments
SET insurer_id = $1, year = $2, premium = $3, note = $4, updated_at = now()
WHERE id = $5
RETURNING id;

-- name: DeleteInsurancePayment :one
DELETE FROM insurance_payments WHERE id = $1 RETURNING id;
