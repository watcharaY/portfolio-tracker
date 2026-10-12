package postgres

import (
	"context"

	"github.com/watcharaY/portfolio-tracker/backend/internal/adapter/postgres/internal/gen"
	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type FundRepo struct{ db *DB }

func NewFundRepo(db *DB) *FundRepo { return &FundRepo{db: db} }

var _ usecase.FundRepository = (*FundRepo)(nil)

func (r *FundRepo) List(ctx context.Context, _ domain.NoFilter) ([]domain.Fund, error) {
	rows, err := r.db.q(ctx).ListFunds(ctx)
	return mapAll(rows, err, func(f gen.Fund) domain.Fund { return domain.Fund(f) })
}

func (r *FundRepo) Get(ctx context.Context, id int64) (domain.Fund, error) {
	f, err := r.db.q(ctx).GetFund(ctx, id)
	return domain.Fund(f), err
}

func (r *FundRepo) Create(ctx context.Context, f *domain.Fund) (int64, error) {
	return r.db.q(ctx).CreateFund(ctx, gen.CreateFundParams{
		Name: f.Name, Category: f.Category, Cost: f.Cost, MarketValue: f.MarketValue, Note: f.Note,
	})
}

func (r *FundRepo) Update(ctx context.Context, id int64, f *domain.Fund) error {
	_, err := r.db.q(ctx).UpdateFund(ctx, gen.UpdateFundParams{
		Name: f.Name, Category: f.Category, Cost: f.Cost, MarketValue: f.MarketValue, Note: f.Note, ID: id,
	})
	return err
}

func (r *FundRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.q(ctx).DeleteFund(ctx, id)
	return err
}

type PVDRepo struct{ db *DB }

func NewPVDRepo(db *DB) *PVDRepo { return &PVDRepo{db: db} }

var _ usecase.PVDRepository = (*PVDRepo)(nil)

func (r *PVDRepo) List(ctx context.Context, _ domain.NoFilter) ([]domain.PVDYear, error) {
	rows, err := r.db.q(ctx).ListPVDYears(ctx)
	return mapAll(rows, err, func(p gen.PVDYear) domain.PVDYear { return domain.PVDYear(p) })
}

func (r *PVDRepo) Get(ctx context.Context, id int64) (domain.PVDYear, error) {
	p, err := r.db.q(ctx).GetPVDYear(ctx, id)
	return domain.PVDYear(p), err
}

func (r *PVDRepo) Create(ctx context.Context, p *domain.PVDYear) (int64, error) {
	return r.db.q(ctx).CreatePVDYear(ctx, gen.CreatePVDYearParams{
		Year: p.Year, SelfContribution: p.SelfContribution, CompanyContribution: p.CompanyContribution,
		SelfProfit: p.SelfProfit, CompanyProfit: p.CompanyProfit, FundAllocation: p.FundAllocation,
	})
}

func (r *PVDRepo) Update(ctx context.Context, id int64, p *domain.PVDYear) error {
	_, err := r.db.q(ctx).UpdatePVDYear(ctx, gen.UpdatePVDYearParams{
		Year: p.Year, SelfContribution: p.SelfContribution, CompanyContribution: p.CompanyContribution,
		SelfProfit: p.SelfProfit, CompanyProfit: p.CompanyProfit, FundAllocation: p.FundAllocation, ID: id,
	})
	return err
}

func (r *PVDRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.q(ctx).DeletePVDYear(ctx, id)
	return err
}

type SavingsRepo struct{ db *DB }

func NewSavingsRepo(db *DB) *SavingsRepo { return &SavingsRepo{db: db} }

var _ usecase.SavingsRepository = (*SavingsRepo)(nil)

func (r *SavingsRepo) List(ctx context.Context, _ domain.NoFilter) ([]domain.SavingsAccount, error) {
	rows, err := r.db.q(ctx).ListSavingsAccounts(ctx)
	return mapAll(rows, err, func(s gen.SavingsAccount) domain.SavingsAccount { return domain.SavingsAccount(s) })
}

func (r *SavingsRepo) Get(ctx context.Context, id int64) (domain.SavingsAccount, error) {
	s, err := r.db.q(ctx).GetSavingsAccount(ctx, id)
	return domain.SavingsAccount(s), err
}

func (r *SavingsRepo) Create(ctx context.Context, s *domain.SavingsAccount) (int64, error) {
	return r.db.q(ctx).CreateSavingsAccount(ctx, gen.CreateSavingsAccountParams{
		Bank: s.Bank, Balance: s.Balance, InterestRate: s.InterestRate, Note: s.Note,
	})
}

func (r *SavingsRepo) Update(ctx context.Context, id int64, s *domain.SavingsAccount) error {
	_, err := r.db.q(ctx).UpdateSavingsAccount(ctx, gen.UpdateSavingsAccountParams{
		Bank: s.Bank, Balance: s.Balance, InterestRate: s.InterestRate, Note: s.Note, ID: id,
	})
	return err
}

func (r *SavingsRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.q(ctx).DeleteSavingsAccount(ctx, id)
	return err
}
