package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type UserRepo struct{ db *DB }

func NewUserRepo(db *DB) *UserRepo { return &UserRepo{db: db} }

var _ usecase.UserRepository = (*UserRepo)(nil)

const userColumns = "id, username, password_hash, must_change_password, created_at, updated_at"

func scanUser(r pgx.Row) (u domain.User, err error) {
	err = r.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.MustChangePassword, &u.CreatedAt, &u.UpdatedAt)
	return
}

func (r *UserRepo) Count(ctx context.Context) (n int, err error) {
	err = r.db.q(ctx).QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&n)
	return n, mapErr(err)
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (domain.User, error) {
	return queryOne(ctx, r.db.q(ctx), scanUser, "SELECT "+userColumns+" FROM users WHERE id = $1", id)
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (domain.User, error) {
	return queryOne(ctx, r.db.q(ctx), scanUser, "SELECT "+userColumns+" FROM users WHERE username = $1", username)
}

func (r *UserRepo) Create(ctx context.Context, u domain.User) (int64, error) {
	return insertReturningID(ctx, r.db.q(ctx),
		"INSERT INTO users (username, password_hash, must_change_password) VALUES ($1, $2, $3)",
		u.Username, u.PasswordHash, u.MustChangePassword)
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id int64, hash string, mustChange bool) error {
	return execAffectingOne(ctx, r.db.q(ctx),
		"UPDATE users SET password_hash = $1, must_change_password = $2, updated_at = now() WHERE id = $3",
		hash, mustChange, id)
}

type SessionRepo struct{ db *DB }

func NewSessionRepo(db *DB) *SessionRepo { return &SessionRepo{db: db} }

var _ usecase.SessionRepository = (*SessionRepo)(nil)

func (r *SessionRepo) Create(ctx context.Context, s domain.Session) error {
	_, err := r.db.q(ctx).Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, ip, user_agent, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		s.TokenHash, s.UserID, s.IP, s.UserAgent, s.CreatedAt, s.ExpiresAt)
	return mapErr(err)
}

func (r *SessionRepo) GetValid(ctx context.Context, tokenHash string, now time.Time) (domain.Session, error) {
	return queryOne(ctx, r.db.q(ctx), func(row pgx.Row) (s domain.Session, err error) {
		err = row.Scan(&s.TokenHash, &s.UserID, &s.IP, &s.UserAgent, &s.CreatedAt, &s.ExpiresAt)
		return
	}, `SELECT token_hash, user_id, ip, user_agent, created_at, expires_at
	    FROM sessions WHERE token_hash = $1 AND expires_at > $2`, tokenHash, now)
}

func (r *SessionRepo) Delete(ctx context.Context, tokenHash string) error {
	_, err := r.db.q(ctx).Exec(ctx, "DELETE FROM sessions WHERE token_hash = $1", tokenHash)
	return mapErr(err)
}

func (r *SessionRepo) DeleteOthers(ctx context.Context, userID int64, keepTokenHash string) error {
	_, err := r.db.q(ctx).Exec(ctx,
		"DELETE FROM sessions WHERE user_id = $1 AND token_hash <> $2", userID, keepTokenHash)
	return mapErr(err)
}

func (r *SessionRepo) DeleteExpired(ctx context.Context, now time.Time) error {
	_, err := r.db.q(ctx).Exec(ctx, "DELETE FROM sessions WHERE expires_at <= $1", now)
	return mapErr(err)
}
