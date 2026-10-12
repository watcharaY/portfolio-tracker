package postgres

import (
	"context"

	"github.com/watcharaY/portfolio-tracker/backend/internal/adapter/postgres/internal/gen"
	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type BrokerRepo struct{ db *DB }

func NewBrokerRepo(db *DB) *BrokerRepo { return &BrokerRepo{db: db} }

var _ usecase.BrokerRepository = (*BrokerRepo)(nil)

func (r *BrokerRepo) List(ctx context.Context, _ domain.NoFilter) ([]domain.BrokerSummary, error) {
	rows, err := r.db.q(ctx).ListBrokers(ctx, nil)
	return mapAll(rows, err, toBrokerSummary)
}

func (r *BrokerRepo) Get(ctx context.Context, id int64) (domain.BrokerSummary, error) {
	rows, err := r.db.q(ctx).ListBrokers(ctx, &id)
	return only(rows, err, toBrokerSummary)
}

func (r *BrokerRepo) Create(ctx context.Context, b *domain.Broker) (int64, error) {
	return r.db.q(ctx).CreateBroker(ctx, gen.CreateBrokerParams{Name: b.Name, CurrentValue: b.CurrentValue, Note: b.Note})
}

func (r *BrokerRepo) Update(ctx context.Context, id int64, b *domain.Broker) error {
	_, err := r.db.q(ctx).UpdateBroker(ctx, gen.UpdateBrokerParams{Name: b.Name, CurrentValue: b.CurrentValue, Note: b.Note, ID: id})
	return err
}

func (r *BrokerRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.q(ctx).DeleteBroker(ctx, id)
	return err
}

// The sums are ::BIGINT casts, not table columns, so no override types them.
func toBrokerSummary(r gen.ListBrokersRow) domain.BrokerSummary {
	return domain.BrokerSummary{
		Broker:      domain.Broker(r.Broker),
		Deposits:    domain.Money(r.Deposits),
		Withdrawals: domain.Money(r.Withdrawals),
	}
}

type BrokerTransactionRepo struct{ db *DB }

func NewBrokerTransactionRepo(db *DB) *BrokerTransactionRepo { return &BrokerTransactionRepo{db: db} }

var _ usecase.BrokerTransactionRepository = (*BrokerTransactionRepo)(nil)

func (r *BrokerTransactionRepo) List(ctx context.Context, f domain.BrokerTransactionFilter) ([]domain.BrokerTransaction, error) {
	rows, err := r.db.q(ctx).ListBrokerTransactions(ctx, f.BrokerID)
	return mapAll(rows, err, func(r gen.ListBrokerTransactionsRow) domain.BrokerTransaction {
		return toBrokerTxn(r.BrokerTransaction, r.BrokerName)
	})
}

func (r *BrokerTransactionRepo) Get(ctx context.Context, id int64) (domain.BrokerTransaction, error) {
	row, err := r.db.q(ctx).GetBrokerTransaction(ctx, id)
	return toBrokerTxn(row.BrokerTransaction, row.BrokerName), err
}

func (r *BrokerTransactionRepo) Create(ctx context.Context, t *domain.BrokerTransaction) (int64, error) {
	return r.db.q(ctx).CreateBrokerTransaction(ctx, gen.CreateBrokerTransactionParams{
		BrokerID: t.BrokerID, Date: t.Date, Type: t.Type, Amount: t.Amount, Note: t.Note,
	})
}

func (r *BrokerTransactionRepo) Update(ctx context.Context, id int64, t *domain.BrokerTransaction) error {
	_, err := r.db.q(ctx).UpdateBrokerTransaction(ctx, gen.UpdateBrokerTransactionParams{
		BrokerID: t.BrokerID, Date: t.Date, Type: t.Type, Amount: t.Amount, Note: t.Note, ID: id,
	})
	return err
}

func (r *BrokerTransactionRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.q(ctx).DeleteBrokerTransaction(ctx, id)
	return err
}

// BrokerName sits between BrokerID and Date in the domain struct, so the
// embedded table row cannot convert directly.
func toBrokerTxn(t gen.BrokerTransaction, brokerName string) domain.BrokerTransaction {
	return domain.BrokerTransaction{
		ID: t.ID, BrokerID: t.BrokerID, BrokerName: brokerName, Date: t.Date,
		Type: t.Type, Amount: t.Amount, Note: t.Note, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}
