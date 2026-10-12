package postgres

import (
	"context"
	"time"

	"github.com/watcharaY/portfolio-tracker/backend/internal/adapter/postgres/internal/gen"
	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type UserRepo struct{ db *DB }

func NewUserRepo(db *DB) *UserRepo { return &UserRepo{db: db} }

var _ usecase.UserRepository = (*UserRepo)(nil)

func (r *UserRepo) Count(ctx context.Context) (int, error) {
	n, err := r.db.q(ctx).CountUsers(ctx)
	return int(n), err
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (domain.User, error) {
	u, err := r.db.q(ctx).GetUserByID(ctx, id)
	return domain.User(u), err
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (domain.User, error) {
	u, err := r.db.q(ctx).GetUserByUsername(ctx, username)
	return domain.User(u), err
}

func (r *UserRepo) Create(ctx context.Context, u domain.User) (int64, error) {
	return r.db.q(ctx).CreateUser(ctx, gen.CreateUserParams{
		Username: u.Username, PasswordHash: u.PasswordHash, MustChangePassword: u.MustChangePassword,
	})
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id int64, hash string, mustChange bool) error {
	_, err := r.db.q(ctx).UpdateUserPassword(ctx, gen.UpdateUserPasswordParams{
		PasswordHash: hash, MustChangePassword: mustChange, ID: id,
	})
	return err
}

type SessionRepo struct{ db *DB }

func NewSessionRepo(db *DB) *SessionRepo { return &SessionRepo{db: db} }

var _ usecase.SessionRepository = (*SessionRepo)(nil)

func (r *SessionRepo) Create(ctx context.Context, s domain.Session) error {
	return r.db.q(ctx).CreateSession(ctx, gen.CreateSessionParams(s))
}

func (r *SessionRepo) GetValid(ctx context.Context, tokenHash string, now time.Time) (domain.Session, error) {
	s, err := r.db.q(ctx).GetValidSession(ctx, gen.GetValidSessionParams{TokenHash: tokenHash, ExpiresAt: now})
	return domain.Session(s), err
}

func (r *SessionRepo) Delete(ctx context.Context, tokenHash string) error {
	return r.db.q(ctx).DeleteSession(ctx, tokenHash)
}

func (r *SessionRepo) DeleteOthers(ctx context.Context, userID int64, keepTokenHash string) error {
	return r.db.q(ctx).DeleteOtherSessions(ctx, gen.DeleteOtherSessionsParams{UserID: userID, TokenHash: keepTokenHash})
}

func (r *SessionRepo) DeleteExpired(ctx context.Context, now time.Time) error {
	return r.db.q(ctx).DeleteExpiredSessions(ctx, now)
}
