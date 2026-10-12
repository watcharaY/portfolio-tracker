package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
)

type AuthService struct {
	users    UserRepository
	sessions SessionRepository
	tx       TxManager
	hasher   PasswordHasher
	tokens   TokenIssuer
	now      Clock
	ttl      time.Duration

	dummyHash string
}

type AuthDeps struct {
	Users    UserRepository
	Sessions SessionRepository
	Tx       TxManager
	Hasher   PasswordHasher
	Tokens   TokenIssuer
	Clock    Clock
	TTL      time.Duration
}

func NewAuthService(d AuthDeps) (*AuthService, error) {
	dummy, err := d.Hasher.Hash("timing-equaliser")
	if err != nil {
		return nil, err
	}
	if d.Clock == nil {
		d.Clock = time.Now
	}
	return &AuthService{
		users: d.Users, sessions: d.Sessions, tx: d.Tx, hasher: d.Hasher, tokens: d.Tokens,
		now: d.Clock, ttl: d.TTL, dummyHash: dummy,
	}, nil
}

type LoginInput struct {
	Username  string
	Password  string
	IP        string
	UserAgent string
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	User      domain.User
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (LoginResult, error) {
	user, err := s.users.GetByUsername(ctx, in.Username)
	if errors.Is(err, domain.ErrNotFound) {
		s.hasher.Compare(s.dummyHash, in.Password)
		return LoginResult{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, err
	}
	if !s.hasher.Compare(user.PasswordHash, in.Password) {
		return LoginResult{}, domain.ErrInvalidCredentials
	}

	token, err := s.tokens.New()
	if err != nil {
		return LoginResult{}, err
	}
	now := s.now()
	session := domain.Session{
		TokenHash: s.tokens.Hash(token),
		UserID:    user.ID,
		IP:        in.IP,
		UserAgent: in.UserAgent,
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl),
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return LoginResult{}, err
	}
	if err := s.sessions.DeleteExpired(ctx, now); err != nil {
		slog.WarnContext(ctx, "cleanup expired sessions", "err", err)
	}
	return LoginResult{Token: token, ExpiresAt: session.ExpiresAt, User: user}, nil
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (domain.User, error) {
	if token == "" {
		return domain.User{}, domain.ErrUnauthenticated
	}
	session, err := s.sessions.GetValid(ctx, s.tokens.Hash(token), s.now())
	if errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, domain.ErrUnauthenticated
	}
	if err != nil {
		return domain.User{}, err
	}
	return s.users.GetByID(ctx, session.UserID)
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, s.tokens.Hash(token))
}

type ChangePasswordInput struct {
	UserID          int64
	CurrentPassword string
	NewPassword     string
	CurrentToken    string
}

func (s *AuthService) ChangePassword(ctx context.Context, in ChangePasswordInput) (domain.User, error) {
	user, err := s.users.GetByID(ctx, in.UserID)
	if err != nil {
		return domain.User{}, err
	}
	if !s.hasher.Compare(user.PasswordHash, in.CurrentPassword) {
		return domain.User{}, &domain.ValidationError{Field: "currentPassword", Message: "is incorrect"}
	}
	if err := domain.ValidateNewPassword(user.Username, in.NewPassword); err != nil {
		return domain.User{}, err
	}
	hash, err := s.hasher.Hash(in.NewPassword)
	if err != nil {
		return domain.User{}, err
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.users.UpdatePassword(ctx, user.ID, hash, false); err != nil {
			return err
		}
		return s.sessions.DeleteOthers(ctx, user.ID, s.tokens.Hash(in.CurrentToken))
	})
	if err != nil {
		return domain.User{}, err
	}
	user.PasswordHash, user.MustChangePassword = hash, false
	return user, nil
}

func (s *AuthService) EnsureDefaultAdmin(ctx context.Context) (bool, error) {
	n, err := s.users.Count(ctx)
	if err != nil || n > 0 {
		return false, err
	}
	hash, err := s.hasher.Hash(domain.DefaultAdminPassword)
	if err != nil {
		return false, err
	}
	_, err = s.users.Create(ctx, domain.User{
		Username:           domain.DefaultAdminUsername,
		PasswordHash:       hash,
		MustChangePassword: true,
	})
	if errors.Is(err, domain.ErrConflict) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("seed admin: %w", err)
	}
	return true, nil
}
