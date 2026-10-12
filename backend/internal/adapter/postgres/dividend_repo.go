package postgres

import (
	"context"

	"github.com/watcharaY/portfolio-tracker/backend/internal/adapter/postgres/internal/gen"
	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type DividendRepo struct{ db *DB }

func NewDividendRepo(db *DB) *DividendRepo { return &DividendRepo{db: db} }

var (
	_ usecase.DividendRepository        = (*DividendRepo)(nil)
	_ usecase.DividendSummaryRepository = (*DividendRepo)(nil)
)

func (r *DividendRepo) List(ctx context.Context, f domain.YearFilter) ([]domain.DividendPayment, error) {
	rows, err := r.db.q(ctx).ListDividendPayments(ctx, gen.ListDividendPaymentsParams{Year: f.Year})
	return mapAll(rows, err, toDividend)
}

func (r *DividendRepo) Get(ctx context.Context, id int64) (domain.DividendPayment, error) {
	rows, err := r.db.q(ctx).ListDividendPayments(ctx, gen.ListDividendPaymentsParams{ID: &id})
	return only(rows, err, toDividend)
}

func (r *DividendRepo) Create(ctx context.Context, d *domain.DividendPayment) (int64, error) {
	b, f, s, i := sourceColumns(d.Source)
	return r.db.q(ctx).CreateDividendPayment(ctx, gen.CreateDividendPaymentParams{
		BrokerID: b, FundID: f, SavingsAccountID: s, InsurerID: i,
		PaidOn: d.PaidOn, Gross: d.Gross, WithholdingTax: d.WithholdingTax, Note: d.Note,
	})
}

func (r *DividendRepo) Update(ctx context.Context, id int64, d *domain.DividendPayment) error {
	b, f, s, i := sourceColumns(d.Source)
	_, err := r.db.q(ctx).UpdateDividendPayment(ctx, gen.UpdateDividendPaymentParams{
		BrokerID: b, FundID: f, SavingsAccountID: s, InsurerID: i,
		PaidOn: d.PaidOn, Gross: d.Gross, WithholdingTax: d.WithholdingTax, Note: d.Note, ID: id,
	})
	return err
}

func (r *DividendRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.q(ctx).DeleteDividendPayment(ctx, id)
	return err
}

func (r *DividendRepo) Summary(ctx context.Context, year int) ([]domain.DividendSourceYear, error) {
	rows, err := r.db.q(ctx).DividendSummary(ctx, year)
	return mapAll(rows, err, func(s gen.DividendSummaryRow) domain.DividendSourceYear {
		return domain.DividendSourceYear{
			Source:     sourceOf(s.BrokerID, s.FundID, s.SavingsAccountID, s.InsurerID),
			SourceName: s.SourceName, AssetValue: domain.Money(s.AssetValue),
			Payments: s.Payments, Months: s.Months,
			Gross: domain.Money(s.Gross), WithholdingTax: domain.Money(s.WithholdingTax),
		}
	})
}

func toDividend(d gen.ListDividendPaymentsRow) domain.DividendPayment {
	return domain.DividendPayment{
		ID: d.ID, Source: sourceOf(d.BrokerID, d.FundID, d.SavingsAccountID, d.InsurerID), SourceName: d.SourceName,
		PaidOn: d.PaidOn, Gross: d.Gross, WithholdingTax: d.WithholdingTax, Note: d.Note,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

// The table's CHECK keeps exactly one of the four columns set.
func sourceColumns(s domain.DividendSource) (broker, fund, savings, insurer *int64) {
	id := s.ID
	switch s.Type {
	case domain.SourceBroker:
		broker = &id
	case domain.SourceFund:
		fund = &id
	case domain.SourceSavings:
		savings = &id
	case domain.SourceInsurer:
		insurer = &id
	}
	return
}

func sourceOf(broker, fund, savings, insurer *int64) domain.DividendSource {
	switch {
	case broker != nil:
		return domain.DividendSource{Type: domain.SourceBroker, ID: *broker}
	case fund != nil:
		return domain.DividendSource{Type: domain.SourceFund, ID: *fund}
	case savings != nil:
		return domain.DividendSource{Type: domain.SourceSavings, ID: *savings}
	default:
		return domain.DividendSource{Type: domain.SourceInsurer, ID: *insurer}
	}
}
