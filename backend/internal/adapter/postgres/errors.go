package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/watcharaY/portfolio-tracker/backend/internal/adapter/postgres/internal/gen"
	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
)

// mapErr is the one translation from driver errors to domain errors. A Get,
// Update or Delete that hits no row is a :one query, so its miss arrives here
// as pgx.ErrNoRows.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return domain.ErrConflict
		case "23503":
			return domain.ErrReferenceNotFound
		case "23514":
			return &domain.ValidationError{Field: pgErr.ConstraintName, Message: "is out of range"}
		case "22003":
			return &domain.ValidationError{Field: pgErr.ColumnName, Message: "is too large"}
		}
	}
	return err
}

// domainErrors wraps the connection handed to gen.New. Generated code reports
// failures only through Exec, Query, QueryRow, Row.Scan, Rows.Scan and
// Rows.Err, so mapping those covers every generated method.
type domainErrors struct{ conn gen.DBTX }

func (d domainErrors) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tag, err := d.conn.Exec(ctx, sql, args...)
	return tag, mapErr(err)
}

func (d domainErrors) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	rows, err := d.conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, mapErr(err)
	}
	return mappedRows{rows}, nil
}

func (d domainErrors) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return mappedRow{d.conn.QueryRow(ctx, sql, args...)}
}

type mappedRow struct{ pgx.Row }

func (r mappedRow) Scan(dest ...any) error { return mapErr(r.Row.Scan(dest...)) }

type mappedRows struct{ pgx.Rows }

func (r mappedRows) Scan(dest ...any) error { return mapErr(r.Rows.Scan(dest...)) }
func (r mappedRows) Err() error             { return mapErr(r.Rows.Err()) }

// mapAll converts generated rows to domain values.
func mapAll[R, T any](rows []R, err error, to func(R) T) ([]T, error) {
	if err != nil {
		return nil, err
	}
	out := make([]T, len(rows))
	for i, r := range rows {
		out[i] = to(r)
	}
	return out, nil
}

// only is Get for the read models whose List query is the one definition of
// their computed columns (broker sums, insurer totals, payment_no). The query
// runs with only the id filter set, so it yields zero rows or one.
func only[R, T any](rows []R, err error, to func(R) T) (T, error) {
	var zero T
	if err != nil {
		return zero, err
	}
	if len(rows) == 0 {
		return zero, domain.ErrNotFound
	}
	return to(rows[0]), nil
}
