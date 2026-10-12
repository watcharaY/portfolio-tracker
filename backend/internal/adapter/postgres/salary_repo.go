package postgres

import (
	"context"

	"github.com/watcharaY/portfolio-tracker/backend/internal/adapter/postgres/internal/gen"
	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type SalaryRepo struct{ db *DB }

func NewSalaryRepo(db *DB) *SalaryRepo { return &SalaryRepo{db: db} }

var _ usecase.SalaryRepository = (*SalaryRepo)(nil)

func (r *SalaryRepo) List(ctx context.Context, f domain.YearFilter) ([]domain.SalaryMonth, error) {
	rows, err := r.db.q(ctx).ListSalaryMonths(ctx, f.Year)
	return mapAll(rows, err, func(s gen.SalaryMonth) domain.SalaryMonth { return domain.SalaryMonth(s) })
}

func (r *SalaryRepo) Get(ctx context.Context, id int64) (domain.SalaryMonth, error) {
	s, err := r.db.q(ctx).GetSalaryMonth(ctx, id)
	return domain.SalaryMonth(s), err
}

func (r *SalaryRepo) Create(ctx context.Context, s *domain.SalaryMonth) (int64, error) {
	return r.db.q(ctx).CreateSalaryMonth(ctx, gen.CreateSalaryMonthParams{
		Year: s.Year, Month: s.Month, Salary: s.Salary, Bonus: s.Bonus,
		WithholdingTax: s.WithholdingTax, PVD: s.PVD, SSO: s.SSO,
	})
}

func (r *SalaryRepo) Update(ctx context.Context, id int64, s *domain.SalaryMonth) error {
	_, err := r.db.q(ctx).UpdateSalaryMonth(ctx, gen.UpdateSalaryMonthParams{
		Year: s.Year, Month: s.Month, Salary: s.Salary, Bonus: s.Bonus,
		WithholdingTax: s.WithholdingTax, PVD: s.PVD, SSO: s.SSO, ID: id,
	})
	return err
}

func (r *SalaryRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.q(ctx).DeleteSalaryMonth(ctx, id)
	return err
}

type TaxDeductionRepo struct{ db *DB }

func NewTaxDeductionRepo(db *DB) *TaxDeductionRepo { return &TaxDeductionRepo{db: db} }

var _ usecase.TaxDeductionRepository = (*TaxDeductionRepo)(nil)

func (r *TaxDeductionRepo) List(ctx context.Context, f domain.YearFilter) ([]domain.TaxDeduction, error) {
	rows, err := r.db.q(ctx).ListTaxDeductions(ctx, f.Year)
	return mapAll(rows, err, func(d gen.TaxDeduction) domain.TaxDeduction { return domain.TaxDeduction(d) })
}

func (r *TaxDeductionRepo) Get(ctx context.Context, id int64) (domain.TaxDeduction, error) {
	d, err := r.db.q(ctx).GetTaxDeduction(ctx, id)
	return domain.TaxDeduction(d), err
}

func (r *TaxDeductionRepo) Create(ctx context.Context, d *domain.TaxDeduction) (int64, error) {
	return r.db.q(ctx).CreateTaxDeduction(ctx, gen.CreateTaxDeductionParams{Year: d.Year, Item: d.Item, Amount: d.Amount})
}

func (r *TaxDeductionRepo) Update(ctx context.Context, id int64, d *domain.TaxDeduction) error {
	_, err := r.db.q(ctx).UpdateTaxDeduction(ctx, gen.UpdateTaxDeductionParams{Year: d.Year, Item: d.Item, Amount: d.Amount, ID: id})
	return err
}

func (r *TaxDeductionRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.q(ctx).DeleteTaxDeduction(ctx, id)
	return err
}
