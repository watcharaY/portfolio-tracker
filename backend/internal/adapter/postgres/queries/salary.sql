-- name: ListSalaryMonths :many
SELECT * FROM salary_months
WHERE sqlc.narg(year)::INT IS NULL OR year = sqlc.narg(year)
ORDER BY year, month;

-- name: GetSalaryMonth :one
SELECT * FROM salary_months WHERE id = $1;

-- name: CreateSalaryMonth :one
INSERT INTO salary_months (year, month, salary, bonus, withholding_tax, pvd, sso)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: UpdateSalaryMonth :one
UPDATE salary_months
SET year = $1, month = $2, salary = $3, bonus = $4, withholding_tax = $5,
    pvd = $6, sso = $7, updated_at = now()
WHERE id = $8
RETURNING id;

-- name: DeleteSalaryMonth :one
DELETE FROM salary_months WHERE id = $1 RETURNING id;

-- name: ListTaxDeductions :many
SELECT * FROM tax_deductions
WHERE sqlc.narg(year)::INT IS NULL OR year = sqlc.narg(year)
ORDER BY year, id;

-- name: GetTaxDeduction :one
SELECT * FROM tax_deductions WHERE id = $1;

-- name: CreateTaxDeduction :one
INSERT INTO tax_deductions (year, item, amount) VALUES ($1, $2, $3) RETURNING id;

-- name: UpdateTaxDeduction :one
UPDATE tax_deductions SET year = $1, item = $2, amount = $3, updated_at = now() WHERE id = $4 RETURNING id;

-- name: DeleteTaxDeduction :one
DELETE FROM tax_deductions WHERE id = $1 RETURNING id;
