package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type FundRepo struct{ db *DB }

func NewFundRepo(db *DB) *FundRepo { return &FundRepo{db: db} }

var _ usecase.FundRepository = (*FundRepo)(nil)

const fundSelect = "SELECT id, name, category, cost, market_value, note, created_at, updated_at FROM funds"

func scanFund(r pgx.Row) (f domain.Fund, err error) {
	err = r.Scan(&f.ID, &f.Name, &f.Category, &f.Cost, &f.MarketValue, &f.Note, &f.CreatedAt, &f.UpdatedAt)
	return
}

func (r *FundRepo) List(ctx context.Context, _ domain.NoFilter) ([]domain.Fund, error) {
	return queryAll(ctx, r.db.q(ctx), scanFund, fundSelect+" ORDER BY name")
}

func (r *FundRepo) Get(ctx context.Context, id int64) (domain.Fund, error) {
	return queryOne(ctx, r.db.q(ctx), scanFund, fundSelect+" WHERE id = $1", id)
}

func (r *FundRepo) Create(ctx context.Context, f *domain.Fund) (int64, error) {
	return insertReturningID(ctx, r.db.q(ctx),
		"INSERT INTO funds (name, category, cost, market_value, note) VALUES ($1, $2, $3, $4, $5)",
		f.Name, f.Category, f.Cost, f.MarketValue, f.Note)
}

func (r *FundRepo) Update(ctx context.Context, id int64, f *domain.Fund) error {
	return execAffectingOne(ctx, r.db.q(ctx),
		`UPDATE funds SET name = $1, category = $2, cost = $3, market_value = $4, note = $5, updated_at = now()
		 WHERE id = $6`,
		f.Name, f.Category, f.Cost, f.MarketValue, f.Note, id)
}

func (r *FundRepo) Delete(ctx context.Context, id int64) error {
	return deleteByID(ctx, r.db.q(ctx), "funds", id)
}

type PVDRepo struct{ db *DB }

func NewPVDRepo(db *DB) *PVDRepo { return &PVDRepo{db: db} }

var _ usecase.PVDRepository = (*PVDRepo)(nil)

const pvdSelect = `SELECT id, year, self_contribution, company_contribution, self_profit, company_profit,
       fund_allocation, created_at, updated_at FROM pvd_years`

func scanPVD(r pgx.Row) (p domain.PVDYear, err error) {
	err = r.Scan(&p.ID, &p.Year, &p.SelfContribution, &p.CompanyContribution, &p.SelfProfit, &p.CompanyProfit,
		&p.FundAllocation, &p.CreatedAt, &p.UpdatedAt)
	return
}

func (r *PVDRepo) List(ctx context.Context, _ domain.NoFilter) ([]domain.PVDYear, error) {
	return queryAll(ctx, r.db.q(ctx), scanPVD, pvdSelect+" ORDER BY year")
}

func (r *PVDRepo) Get(ctx context.Context, id int64) (domain.PVDYear, error) {
	return queryOne(ctx, r.db.q(ctx), scanPVD, pvdSelect+" WHERE id = $1", id)
}

func (r *PVDRepo) Create(ctx context.Context, p *domain.PVDYear) (int64, error) {
	return insertReturningID(ctx, r.db.q(ctx),
		`INSERT INTO pvd_years (year, self_contribution, company_contribution, self_profit, company_profit, fund_allocation)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		p.Year, p.SelfContribution, p.CompanyContribution, p.SelfProfit, p.CompanyProfit, p.FundAllocation)
}

func (r *PVDRepo) Update(ctx context.Context, id int64, p *domain.PVDYear) error {
	return execAffectingOne(ctx, r.db.q(ctx),
		`UPDATE pvd_years SET year = $1, self_contribution = $2, company_contribution = $3,
		        self_profit = $4, company_profit = $5, fund_allocation = $6, updated_at = now()
		 WHERE id = $7`,
		p.Year, p.SelfContribution, p.CompanyContribution, p.SelfProfit, p.CompanyProfit, p.FundAllocation, id)
}

func (r *PVDRepo) Delete(ctx context.Context, id int64) error {
	return deleteByID(ctx, r.db.q(ctx), "pvd_years", id)
}

type SavingsRepo struct{ db *DB }

func NewSavingsRepo(db *DB) *SavingsRepo { return &SavingsRepo{db: db} }

var _ usecase.SavingsRepository = (*SavingsRepo)(nil)

const savingsSelect = "SELECT id, bank, balance, interest_bp, note, created_at, updated_at FROM savings_accounts"

func scanSavings(r pgx.Row) (s domain.SavingsAccount, err error) {
	err = r.Scan(&s.ID, &s.Bank, &s.Balance, &s.InterestRate, &s.Note, &s.CreatedAt, &s.UpdatedAt)
	return
}

func (r *SavingsRepo) List(ctx context.Context, _ domain.NoFilter) ([]domain.SavingsAccount, error) {
	return queryAll(ctx, r.db.q(ctx), scanSavings, savingsSelect+" ORDER BY bank, id")
}

func (r *SavingsRepo) Get(ctx context.Context, id int64) (domain.SavingsAccount, error) {
	return queryOne(ctx, r.db.q(ctx), scanSavings, savingsSelect+" WHERE id = $1", id)
}

func (r *SavingsRepo) Create(ctx context.Context, s *domain.SavingsAccount) (int64, error) {
	return insertReturningID(ctx, r.db.q(ctx),
		"INSERT INTO savings_accounts (bank, balance, interest_bp, note) VALUES ($1, $2, $3, $4)",
		s.Bank, s.Balance, s.InterestRate, s.Note)
}

func (r *SavingsRepo) Update(ctx context.Context, id int64, s *domain.SavingsAccount) error {
	return execAffectingOne(ctx, r.db.q(ctx),
		"UPDATE savings_accounts SET bank = $1, balance = $2, interest_bp = $3, note = $4, updated_at = now() WHERE id = $5",
		s.Bank, s.Balance, s.InterestRate, s.Note, id)
}

func (r *SavingsRepo) Delete(ctx context.Context, id int64) error {
	return deleteByID(ctx, r.db.q(ctx), "savings_accounts", id)
}
