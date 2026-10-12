package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type SalaryRepo struct{ db *DB }

func NewSalaryRepo(db *DB) *SalaryRepo { return &SalaryRepo{db: db} }

var _ usecase.SalaryRepository = (*SalaryRepo)(nil)

const salarySelect = `SELECT id, year, month, salary, bonus, withholding_tax, pvd, sso, created_at, updated_at
FROM salary_months`

func scanSalary(r pgx.Row) (s domain.SalaryMonth, err error) {
	err = r.Scan(&s.ID, &s.Year, &s.Month, &s.Salary, &s.Bonus, &s.WithholdingTax, &s.PVD, &s.SSO, &s.CreatedAt, &s.UpdatedAt)
	return
}

func (r *SalaryRepo) List(ctx context.Context, f domain.YearFilter) ([]domain.SalaryMonth, error) {
	var w where
	if f.Year != nil {
		w.add("year = $%d", *f.Year)
	}
	return queryAll(ctx, r.db.q(ctx), scanSalary, salarySelect+w.String()+" ORDER BY year, month", w.args...)
}

func (r *SalaryRepo) Get(ctx context.Context, id int64) (domain.SalaryMonth, error) {
	return queryOne(ctx, r.db.q(ctx), scanSalary, salarySelect+" WHERE id = $1", id)
}

func (r *SalaryRepo) Create(ctx context.Context, s *domain.SalaryMonth) (int64, error) {
	return insertReturningID(ctx, r.db.q(ctx),
		`INSERT INTO salary_months (year, month, salary, bonus, withholding_tax, pvd, sso)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.Year, s.Month, s.Salary, s.Bonus, s.WithholdingTax, s.PVD, s.SSO)
}

func (r *SalaryRepo) Update(ctx context.Context, id int64, s *domain.SalaryMonth) error {
	return execAffectingOne(ctx, r.db.q(ctx),
		`UPDATE salary_months SET year = $1, month = $2, salary = $3, bonus = $4, withholding_tax = $5,
		        pvd = $6, sso = $7, updated_at = now()
		 WHERE id = $8`,
		s.Year, s.Month, s.Salary, s.Bonus, s.WithholdingTax, s.PVD, s.SSO, id)
}

func (r *SalaryRepo) Delete(ctx context.Context, id int64) error {
	return deleteByID(ctx, r.db.q(ctx), "salary_months", id)
}

type TaxDeductionRepo struct{ db *DB }

func NewTaxDeductionRepo(db *DB) *TaxDeductionRepo { return &TaxDeductionRepo{db: db} }

var _ usecase.TaxDeductionRepository = (*TaxDeductionRepo)(nil)

const deductionSelect = "SELECT id, year, item, amount, created_at, updated_at FROM tax_deductions"

func scanDeduction(r pgx.Row) (d domain.TaxDeduction, err error) {
	err = r.Scan(&d.ID, &d.Year, &d.Item, &d.Amount, &d.CreatedAt, &d.UpdatedAt)
	return
}

func (r *TaxDeductionRepo) List(ctx context.Context, f domain.YearFilter) ([]domain.TaxDeduction, error) {
	var w where
	if f.Year != nil {
		w.add("year = $%d", *f.Year)
	}
	return queryAll(ctx, r.db.q(ctx), scanDeduction, deductionSelect+w.String()+" ORDER BY year, id", w.args...)
}

func (r *TaxDeductionRepo) Get(ctx context.Context, id int64) (domain.TaxDeduction, error) {
	return queryOne(ctx, r.db.q(ctx), scanDeduction, deductionSelect+" WHERE id = $1", id)
}

func (r *TaxDeductionRepo) Create(ctx context.Context, d *domain.TaxDeduction) (int64, error) {
	return insertReturningID(ctx, r.db.q(ctx),
		"INSERT INTO tax_deductions (year, item, amount) VALUES ($1, $2, $3)", d.Year, d.Item, d.Amount)
}

func (r *TaxDeductionRepo) Update(ctx context.Context, id int64, d *domain.TaxDeduction) error {
	return execAffectingOne(ctx, r.db.q(ctx),
		"UPDATE tax_deductions SET year = $1, item = $2, amount = $3, updated_at = now() WHERE id = $4",
		d.Year, d.Item, d.Amount, id)
}

func (r *TaxDeductionRepo) Delete(ctx context.Context, id int64) error {
	return deleteByID(ctx, r.db.q(ctx), "tax_deductions", id)
}
