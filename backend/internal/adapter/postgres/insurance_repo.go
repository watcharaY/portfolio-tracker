package postgres

import (
	"context"

	"github.com/watcharaY/portfolio-tracker/backend/internal/adapter/postgres/internal/gen"
	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type InsurerRepo struct{ db *DB }

func NewInsurerRepo(db *DB) *InsurerRepo { return &InsurerRepo{db: db} }

var _ usecase.InsurerRepository = (*InsurerRepo)(nil)

func (r *InsurerRepo) List(ctx context.Context, _ domain.NoFilter) ([]domain.InsurerSummary, error) {
	rows, err := r.db.q(ctx).ListInsurers(ctx, nil)
	return mapAll(rows, err, toInsurerSummary)
}

func (r *InsurerRepo) Get(ctx context.Context, id int64) (domain.InsurerSummary, error) {
	rows, err := r.db.q(ctx).ListInsurers(ctx, &id)
	return only(rows, err, toInsurerSummary)
}

func (r *InsurerRepo) Create(ctx context.Context, i *domain.Insurer) (int64, error) {
	return r.db.q(ctx).CreateInsurer(ctx, gen.CreateInsurerParams{Company: i.Company, Note: i.Note})
}

func (r *InsurerRepo) Update(ctx context.Context, id int64, i *domain.Insurer) error {
	_, err := r.db.q(ctx).UpdateInsurer(ctx, gen.UpdateInsurerParams{Company: i.Company, Note: i.Note, ID: id})
	return err
}

func (r *InsurerRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.q(ctx).DeleteInsurer(ctx, id)
	return err
}

// TotalPaid is a ::BIGINT cast, not a table column, so no override types it.
func toInsurerSummary(r gen.ListInsurersRow) domain.InsurerSummary {
	return domain.InsurerSummary{
		Insurer:   domain.Insurer(r.Insurer),
		Payments:  r.Payments,
		TotalPaid: domain.Money(r.TotalPaid),
	}
}

type InsurancePaymentRepo struct{ db *DB }

func NewInsurancePaymentRepo(db *DB) *InsurancePaymentRepo { return &InsurancePaymentRepo{db: db} }

var _ usecase.InsurancePaymentRepository = (*InsurancePaymentRepo)(nil)

func (r *InsurancePaymentRepo) List(ctx context.Context, f domain.InsurancePaymentFilter) ([]domain.InsurancePayment, error) {
	rows, err := r.db.q(ctx).ListInsurancePayments(ctx, gen.ListInsurancePaymentsParams{InsurerID: f.InsurerID, Year: f.Year})
	return mapAll(rows, err, toPayment)
}

func (r *InsurancePaymentRepo) Get(ctx context.Context, id int64) (domain.InsurancePayment, error) {
	rows, err := r.db.q(ctx).ListInsurancePayments(ctx, gen.ListInsurancePaymentsParams{ID: &id})
	return only(rows, err, toPayment)
}

func (r *InsurancePaymentRepo) Create(ctx context.Context, p *domain.InsurancePayment) (int64, error) {
	return r.db.q(ctx).CreateInsurancePayment(ctx, gen.CreateInsurancePaymentParams{
		InsurerID: p.InsurerID, Year: p.Year, Premium: p.Premium, Note: p.Note,
	})
}

func (r *InsurancePaymentRepo) Update(ctx context.Context, id int64, p *domain.InsurancePayment) error {
	_, err := r.db.q(ctx).UpdateInsurancePayment(ctx, gen.UpdateInsurancePaymentParams{
		InsurerID: p.InsurerID, Year: p.Year, Premium: p.Premium, Note: p.Note, ID: id,
	})
	return err
}

func (r *InsurancePaymentRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.q(ctx).DeleteInsurancePayment(ctx, id)
	return err
}

func toPayment(p gen.ListInsurancePaymentsRow) domain.InsurancePayment {
	return domain.InsurancePayment(p)
}
