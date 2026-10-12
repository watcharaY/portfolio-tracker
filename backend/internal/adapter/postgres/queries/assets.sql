-- name: ListFunds :many
SELECT * FROM funds ORDER BY name;

-- name: GetFund :one
SELECT * FROM funds WHERE id = $1;

-- name: CreateFund :one
INSERT INTO funds (name, category, cost, market_value, note)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: UpdateFund :one
UPDATE funds
SET name = $1, category = $2, cost = $3, market_value = $4, note = $5, updated_at = now()
WHERE id = $6
RETURNING id;

-- name: DeleteFund :one
DELETE FROM funds WHERE id = $1 RETURNING id;

-- name: ListPVDYears :many
SELECT * FROM pvd_years ORDER BY year;

-- name: GetPVDYear :one
SELECT * FROM pvd_years WHERE id = $1;

-- name: CreatePVDYear :one
INSERT INTO pvd_years (year, self_contribution, company_contribution, self_profit, company_profit, fund_allocation)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: UpdatePVDYear :one
UPDATE pvd_years
SET year = $1, self_contribution = $2, company_contribution = $3,
    self_profit = $4, company_profit = $5, fund_allocation = $6, updated_at = now()
WHERE id = $7
RETURNING id;

-- name: DeletePVDYear :one
DELETE FROM pvd_years WHERE id = $1 RETURNING id;

-- name: ListSavingsAccounts :many
SELECT * FROM savings_accounts ORDER BY bank, id;

-- name: GetSavingsAccount :one
SELECT * FROM savings_accounts WHERE id = $1;

-- name: CreateSavingsAccount :one
INSERT INTO savings_accounts (bank, balance, interest_bp, note)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: UpdateSavingsAccount :one
UPDATE savings_accounts
SET bank = $1, balance = $2, interest_bp = $3, note = $4, updated_at = now()
WHERE id = $5
RETURNING id;

-- name: DeleteSavingsAccount :one
DELETE FROM savings_accounts WHERE id = $1 RETURNING id;
