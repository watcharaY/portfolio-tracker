package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
)

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

func queryAll[T any](ctx context.Context, q querier, scan func(pgx.Row) (T, error), sql string, args ...any) ([]T, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, mapErr(err)
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (T, error) { return scan(r) })
	if err != nil {
		return nil, mapErr(err)
	}
	return items, nil
}

func queryOne[T any](ctx context.Context, q querier, scan func(pgx.Row) (T, error), sql string, args ...any) (T, error) {
	item, err := scan(q.QueryRow(ctx, sql, args...))
	return item, mapErr(err)
}

func insertReturningID(ctx context.Context, q querier, sql string, args ...any) (int64, error) {
	var id int64
	err := q.QueryRow(ctx, sql+" RETURNING id", args...).Scan(&id)
	return id, mapErr(err)
}

func execAffectingOne(ctx context.Context, q querier, sql string, args ...any) error {
	tag, err := q.Exec(ctx, sql, args...)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func deleteByID(ctx context.Context, q querier, table string, id int64) error {
	return execAffectingOne(ctx, q, "DELETE FROM "+table+" WHERE id = $1", id)
}

type where struct {
	parts []string
	args  []any
}

func (w *where) add(expr string, arg any) {
	w.args = append(w.args, arg)
	w.parts = append(w.parts, fmt.Sprintf(expr, len(w.args)))
}

func (w *where) String() string {
	if len(w.parts) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(w.parts, " AND ")
}
