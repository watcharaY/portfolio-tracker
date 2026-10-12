-- One definition of a payment's source name; Get reads this query with only id set.
-- name: ListDividendPayments :many
SELECT d.id, d.broker_id, d.fund_id, d.savings_account_id, d.insurer_id,
       COALESCE(b.name, f.name, s.bank, i.company)::TEXT AS source_name,
       d.paid_on, d.gross, d.withholding_tax, d.note, d.created_at, d.updated_at
FROM dividend_payments d
LEFT JOIN brokers b          ON b.id = d.broker_id
LEFT JOIN funds f            ON f.id = d.fund_id
LEFT JOIN savings_accounts s ON s.id = d.savings_account_id
LEFT JOIN insurers i         ON i.id = d.insurer_id
WHERE (sqlc.narg(id)::BIGINT IS NULL OR d.id = sqlc.narg(id))
  AND (sqlc.narg(year)::INT IS NULL OR EXTRACT(YEAR FROM d.paid_on)::INT = sqlc.narg(year))
ORDER BY d.paid_on DESC, d.id DESC;

-- name: CreateDividendPayment :one
INSERT INTO dividend_payments (broker_id, fund_id, savings_account_id, insurer_id, paid_on, gross, withholding_tax, note)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;

-- name: UpdateDividendPayment :one
UPDATE dividend_payments
SET broker_id = $1, fund_id = $2, savings_account_id = $3, insurer_id = $4,
    paid_on = $5, gross = $6, withholding_tax = $7, note = $8, updated_at = now()
WHERE id = $9
RETURNING id;

-- name: DeleteDividendPayment :one
DELETE FROM dividend_payments WHERE id = $1 RETURNING id;

-- Per source for one year. asset_value is what the yield divides by: a broker's
-- current value, a fund's market value, a savings balance, or the premiums paid
-- to an insurer so far.
-- name: DividendSummary :many
SELECT d.broker_id, d.fund_id, d.savings_account_id, d.insurer_id,
       COALESCE(b.name, f.name, s.bank, i.company)::TEXT AS source_name,
       COALESCE(b.current_value, f.market_value, s.balance,
                (SELECT SUM(p.premium) FROM insurance_payments p WHERE p.insurer_id = d.insurer_id), 0)::BIGINT AS asset_value,
       COUNT(*)::INT AS payments,
       array_agg(DISTINCT EXTRACT(MONTH FROM d.paid_on)::INT)::INT[] AS months,
       SUM(d.gross)::BIGINT AS gross,
       SUM(d.withholding_tax)::BIGINT AS withholding_tax
FROM dividend_payments d
LEFT JOIN brokers b          ON b.id = d.broker_id
LEFT JOIN funds f            ON f.id = d.fund_id
LEFT JOIN savings_accounts s ON s.id = d.savings_account_id
LEFT JOIN insurers i         ON i.id = d.insurer_id
WHERE EXTRACT(YEAR FROM d.paid_on)::INT = sqlc.arg(year)::INT
GROUP BY d.broker_id, d.fund_id, d.savings_account_id, d.insurer_id,
         b.name, f.name, s.bank, i.company, b.current_value, f.market_value, s.balance
ORDER BY (d.broker_id IS NULL), (d.fund_id IS NULL), (d.savings_account_id IS NULL), source_name;
