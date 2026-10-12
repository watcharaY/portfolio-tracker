package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type InsurerRepo struct{ db *DB }

func NewInsurerRepo(db *DB) *InsurerRepo { return &InsurerRepo{db: db} }

var _ usecase.InsurerRepository = (*InsurerRepo)(nil)

const insurerSelect = `
SELECT i.id, i.company, i.note, i.created_at, i.updated_at,
       COUNT(p.id)::INT, COALESCE(SUM(p.premium), 0)::BIGINT
FROM insurers i
LEFT JOIN insurance_payments p ON p.insurer_id = i.id`

func scanInsurer(r pgx.Row) (i domain.InsurerSummary, err error) {
	err = r.Scan(&i.ID, &i.Company, &i.Note, &i.CreatedAt, &i.UpdatedAt, &i.Payments, &i.TotalPaid)
	return
}

func (r *InsurerRepo) List(ctx context.Context, _ domain.NoFilter) ([]domain.InsurerSummary, error) {
	return queryAll(ctx, r.db.q(ctx), scanInsurer, insurerSelect+" GROUP BY i.id ORDER BY i.company")
}

func (r *InsurerRepo) Get(ctx context.Context, id int64) (domain.InsurerSummary, error) {
	return queryOne(ctx, r.db.q(ctx), scanInsurer, insurerSelect+" WHERE i.id = $1 GROUP BY i.id", id)
}

func (r *InsurerRepo) Create(ctx context.Context, i *domain.Insurer) (int64, error) {
	return insertReturningID(ctx, r.db.q(ctx), "INSERT INTO insurers (company, note) VALUES ($1, $2)", i.Company, i.Note)
}

func (r *InsurerRepo) Update(ctx context.Context, id int64, i *domain.Insurer) error {
	return execAffectingOne(ctx, r.db.q(ctx),
		"UPDATE insurers SET company = $1, note = $2, updated_at = now() WHERE id = $3", i.Company, i.Note, id)
}

func (r *InsurerRepo) Delete(ctx context.Context, id int64) error {
	return deleteByID(ctx, r.db.q(ctx), "insurers", id)
}

type InsurancePaymentRepo struct{ db *DB }

func NewInsurancePaymentRepo(db *DB) *InsurancePaymentRepo { return &InsurancePaymentRepo{db: db} }

var _ usecase.InsurancePaymentRepository = (*InsurancePaymentRepo)(nil)

const paymentSelect = `
SELECT * FROM (
    SELECT p.id, p.insurer_id, i.company,
           ROW_NUMBER() OVER (PARTITION BY p.insurer_id ORDER BY p.year NULLS FIRST, p.id)::INT AS payment_no,
           p.year, p.premium, p.note, p.created_at, p.updated_at
    FROM insurance_payments p JOIN insurers i ON i.id = p.insurer_id
) x`

func scanPayment(r pgx.Row) (p domain.InsurancePayment, err error) {
	err = r.Scan(&p.ID, &p.InsurerID, &p.Company, &p.PaymentNo, &p.Year, &p.Premium, &p.Note, &p.CreatedAt, &p.UpdatedAt)
	return
}

func (r *InsurancePaymentRepo) List(ctx context.Context, f domain.InsurancePaymentFilter) ([]domain.InsurancePayment, error) {
	var w where
	if f.InsurerID != nil {
		w.add("x.insurer_id = $%d", *f.InsurerID)
	}
	if f.Year != nil {
		w.add("x.year = $%d", *f.Year)
	}
	return queryAll(ctx, r.db.q(ctx), scanPayment, paymentSelect+w.String()+" ORDER BY x.company, x.payment_no", w.args...)
}

func (r *InsurancePaymentRepo) Get(ctx context.Context, id int64) (domain.InsurancePayment, error) {
	return queryOne(ctx, r.db.q(ctx), scanPayment, paymentSelect+" WHERE x.id = $1", id)
}

func (r *InsurancePaymentRepo) Create(ctx context.Context, p *domain.InsurancePayment) (int64, error) {
	return insertReturningID(ctx, r.db.q(ctx),
		"INSERT INTO insurance_payments (insurer_id, year, premium, note) VALUES ($1, $2, $3, $4)",
		p.InsurerID, p.Year, p.Premium, p.Note)
}

func (r *InsurancePaymentRepo) Update(ctx context.Context, id int64, p *domain.InsurancePayment) error {
	return execAffectingOne(ctx, r.db.q(ctx),
		"UPDATE insurance_payments SET insurer_id = $1, year = $2, premium = $3, note = $4, updated_at = now() WHERE id = $5",
		p.InsurerID, p.Year, p.Premium, p.Note, id)
}

func (r *InsurancePaymentRepo) Delete(ctx context.Context, id int64) error {
	return deleteByID(ctx, r.db.q(ctx), "insurance_payments", id)
}
