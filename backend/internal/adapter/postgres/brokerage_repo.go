package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type BrokerRepo struct{ db *DB }

func NewBrokerRepo(db *DB) *BrokerRepo { return &BrokerRepo{db: db} }

var _ usecase.BrokerRepository = (*BrokerRepo)(nil)

// SUM(bigint) is NUMERIC in Postgres; cast back so it scans into Money.
const brokerSelect = `
SELECT b.id, b.name, b.current_value, b.note, b.created_at, b.updated_at,
       COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'deposit'),  0)::BIGINT,
       COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'withdraw'), 0)::BIGINT
FROM brokers b
LEFT JOIN broker_transactions t ON t.broker_id = b.id`

func scanBroker(r pgx.Row) (b domain.BrokerSummary, err error) {
	err = r.Scan(&b.ID, &b.Name, &b.CurrentValue, &b.Note, &b.CreatedAt, &b.UpdatedAt, &b.Deposits, &b.Withdrawals)
	return
}

func (r *BrokerRepo) List(ctx context.Context, _ domain.NoFilter) ([]domain.BrokerSummary, error) {
	return queryAll(ctx, r.db.q(ctx), scanBroker, brokerSelect+" GROUP BY b.id ORDER BY b.name")
}

func (r *BrokerRepo) Get(ctx context.Context, id int64) (domain.BrokerSummary, error) {
	return queryOne(ctx, r.db.q(ctx), scanBroker, brokerSelect+" WHERE b.id = $1 GROUP BY b.id", id)
}

func (r *BrokerRepo) Create(ctx context.Context, b *domain.Broker) (int64, error) {
	return insertReturningID(ctx, r.db.q(ctx),
		"INSERT INTO brokers (name, current_value, note) VALUES ($1, $2, $3)",
		b.Name, b.CurrentValue, b.Note)
}

func (r *BrokerRepo) Update(ctx context.Context, id int64, b *domain.Broker) error {
	return execAffectingOne(ctx, r.db.q(ctx),
		"UPDATE brokers SET name = $1, current_value = $2, note = $3, updated_at = now() WHERE id = $4",
		b.Name, b.CurrentValue, b.Note, id)
}

func (r *BrokerRepo) Delete(ctx context.Context, id int64) error {
	return deleteByID(ctx, r.db.q(ctx), "brokers", id)
}

type BrokerTransactionRepo struct{ db *DB }

func NewBrokerTransactionRepo(db *DB) *BrokerTransactionRepo { return &BrokerTransactionRepo{db: db} }

var _ usecase.BrokerTransactionRepository = (*BrokerTransactionRepo)(nil)

const brokerTxnSelect = `
SELECT t.id, t.broker_id, b.name, t.txn_date, t.type, t.amount, t.note, t.created_at, t.updated_at
FROM broker_transactions t JOIN brokers b ON b.id = t.broker_id`

func scanBrokerTxn(r pgx.Row) (t domain.BrokerTransaction, err error) {
	err = r.Scan(&t.ID, &t.BrokerID, &t.BrokerName, &t.Date, &t.Type, &t.Amount, &t.Note, &t.CreatedAt, &t.UpdatedAt)
	return
}

func (r *BrokerTransactionRepo) List(ctx context.Context, f domain.BrokerTransactionFilter) ([]domain.BrokerTransaction, error) {
	var w where
	if f.BrokerID != nil {
		w.add("t.broker_id = $%d", *f.BrokerID)
	}
	return queryAll(ctx, r.db.q(ctx), scanBrokerTxn,
		brokerTxnSelect+w.String()+" ORDER BY t.txn_date DESC NULLS LAST, t.id DESC", w.args...)
}

func (r *BrokerTransactionRepo) Get(ctx context.Context, id int64) (domain.BrokerTransaction, error) {
	return queryOne(ctx, r.db.q(ctx), scanBrokerTxn, brokerTxnSelect+" WHERE t.id = $1", id)
}

func (r *BrokerTransactionRepo) Create(ctx context.Context, t *domain.BrokerTransaction) (int64, error) {
	return insertReturningID(ctx, r.db.q(ctx),
		"INSERT INTO broker_transactions (broker_id, txn_date, type, amount, note) VALUES ($1, $2, $3, $4, $5)",
		t.BrokerID, t.Date, t.Type, t.Amount, t.Note)
}

func (r *BrokerTransactionRepo) Update(ctx context.Context, id int64, t *domain.BrokerTransaction) error {
	return execAffectingOne(ctx, r.db.q(ctx),
		`UPDATE broker_transactions
		 SET broker_id = $1, txn_date = $2, type = $3, amount = $4, note = $5, updated_at = now()
		 WHERE id = $6`,
		t.BrokerID, t.Date, t.Type, t.Amount, t.Note, id)
}

func (r *BrokerTransactionRepo) Delete(ctx context.Context, id int64) error {
	return deleteByID(ctx, r.db.q(ctx), "broker_transactions", id)
}
