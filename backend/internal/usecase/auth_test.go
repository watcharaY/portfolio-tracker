package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type fakeUsers struct{ byID map[int64]domain.User }

func (f *fakeUsers) Count(context.Context) (int, error) { return len(f.byID), nil }
func (f *fakeUsers) GetByID(_ context.Context, id int64) (domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return u, domain.ErrNotFound
	}
	return u, nil
}
func (f *fakeUsers) GetByUsername(_ context.Context, name string) (domain.User, error) {
	for _, u := range f.byID {
		if u.Username == name {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}
func (f *fakeUsers) Create(_ context.Context, u domain.User) (int64, error) {
	u.ID = int64(len(f.byID) + 1)
	f.byID[u.ID] = u
	return u.ID, nil
}
func (f *fakeUsers) UpdatePassword(_ context.Context, id int64, hash string, must bool) error {
	u := f.byID[id]
	u.PasswordHash, u.MustChangePassword = hash, must
	f.byID[id] = u
	return nil
}

type fakeSessions struct{ byHash map[string]domain.Session }

func (f *fakeSessions) Create(_ context.Context, s domain.Session) error {
	f.byHash[s.TokenHash] = s
	return nil
}
func (f *fakeSessions) GetValid(_ context.Context, h string, now time.Time) (domain.Session, error) {
	s, ok := f.byHash[h]
	if !ok || s.Expired(now) {
		return s, domain.ErrNotFound
	}
	return s, nil
}
func (f *fakeSessions) Delete(_ context.Context, h string) error { delete(f.byHash, h); return nil }
func (f *fakeSessions) DeleteOthers(_ context.Context, uid int64, keep string) error {
	for h, s := range f.byHash {
		if s.UserID == uid && h != keep {
			delete(f.byHash, h)
		}
	}
	return nil
}
func (f *fakeSessions) DeleteExpired(_ context.Context, now time.Time) error {
	for h, s := range f.byHash {
		if s.Expired(now) {
			delete(f.byHash, h)
		}
	}
	return nil
}

type noTx struct{}

func (noTx) WithinTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type plainHasher struct{}

func (plainHasher) Hash(p string) (string, error) { return "h:" + p, nil }
func (plainHasher) Compare(h, p string) bool      { return h == "h:"+p }

type seqTokens struct{ n int }

func (s *seqTokens) New() (string, error) { s.n++; return "tok" + string(rune('0'+s.n)), nil }
func (*seqTokens) Hash(t string) string   { return "sha:" + t }

type env struct {
	svc      *usecase.AuthService
	users    *fakeUsers
	sessions *fakeSessions
	now      time.Time
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		users:    &fakeUsers{byID: map[int64]domain.User{}},
		sessions: &fakeSessions{byHash: map[string]domain.Session{}},
		now:      time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC),
	}
	svc, err := usecase.NewAuthService(usecase.AuthDeps{
		Users: e.users, Sessions: e.sessions, Tx: noTx{}, Hasher: plainHasher{}, Tokens: &seqTokens{},
		Clock: func() time.Time { return e.now }, TTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	e.svc = svc
	return e
}

func (e *env) login(t *testing.T, user, pw string) usecase.LoginResult {
	t.Helper()
	res, err := e.svc.Login(t.Context(), usecase.LoginInput{Username: user, Password: pw})
	if err != nil {
		t.Fatalf("login %s: %v", user, err)
	}
	return res
}

func TestEnsureDefaultAdmin_SeedsOnce(t *testing.T) {
	e := newEnv(t)
	ctx := t.Context()
	if seeded, err := e.svc.EnsureDefaultAdmin(ctx); err != nil || !seeded {
		t.Fatalf("first call: seeded=%v err=%v", seeded, err)
	}
	if seeded, _ := e.svc.EnsureDefaultAdmin(ctx); seeded {
		t.Fatal("second call must not seed again")
	}
	if u := e.users.byID[1]; u.Username != "admin" || !u.MustChangePassword {
		t.Fatalf("seeded user = %+v", u)
	}
}

func TestLogin(t *testing.T) {
	e := newEnv(t)
	ctx := t.Context()
	e.svc.EnsureDefaultAdmin(ctx)

	for _, in := range []usecase.LoginInput{{Username: "admin", Password: "nope"}, {Username: "ghost", Password: "admin"}} {
		if _, err := e.svc.Login(ctx, in); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Errorf("%+v: want ErrInvalidCredentials, got %v", in, err)
		}
	}

	res := e.login(t, "admin", "admin")
	if !res.ExpiresAt.Equal(e.now.Add(time.Hour)) {
		t.Errorf("expires at %v", res.ExpiresAt)
	}
	if _, ok := e.sessions.byHash["sha:"+res.Token]; !ok {
		t.Error("session must be stored by token hash, never the raw token")
	}
	u, err := e.svc.Authenticate(ctx, res.Token)
	if err != nil || u.Username != "admin" {
		t.Fatalf("authenticate: %v %v", u, err)
	}
}

func TestAuthenticate_ExpiredAndLoggedOut(t *testing.T) {
	e := newEnv(t)
	ctx := t.Context()
	e.svc.EnsureDefaultAdmin(ctx)

	res := e.login(t, "admin", "admin")
	e.now = e.now.Add(2 * time.Hour)
	if _, err := e.svc.Authenticate(ctx, res.Token); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("expired session: got %v", err)
	}

	e.now = e.now.Add(-2 * time.Hour)
	res = e.login(t, "admin", "admin")
	if err := e.svc.Logout(ctx, res.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Authenticate(ctx, res.Token); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("after logout: got %v", err)
	}
}

func TestChangePassword_RevokesOtherSessions(t *testing.T) {
	e := newEnv(t)
	ctx := t.Context()
	e.svc.EnsureDefaultAdmin(ctx)

	laptop := e.login(t, "admin", "admin")
	phone := e.login(t, "admin", "admin")

	if _, err := e.svc.ChangePassword(ctx, usecase.ChangePasswordInput{
		UserID: 1, CurrentPassword: "wrong", NewPassword: "New-Mango-42-Rain", CurrentToken: laptop.Token,
	}); err == nil {
		t.Fatal("wrong current password must fail")
	}
	_, err := e.svc.ChangePassword(ctx, usecase.ChangePasswordInput{
		UserID: 1, CurrentPassword: "admin", NewPassword: "admin", CurrentToken: laptop.Token,
	})
	if _, ok := errors.AsType[*domain.ValidationError](err); !ok {
		t.Fatalf("weak password: want ValidationError, got %v", err)
	}

	u, err := e.svc.ChangePassword(ctx, usecase.ChangePasswordInput{
		UserID: 1, CurrentPassword: "admin", NewPassword: "New-Mango-42-Rain", CurrentToken: laptop.Token,
	})
	if err != nil || u.MustChangePassword {
		t.Fatalf("change password: %+v %v", u, err)
	}
	if _, err := e.svc.Authenticate(ctx, laptop.Token); err != nil {
		t.Errorf("current session must survive: %v", err)
	}
	if _, err := e.svc.Authenticate(ctx, phone.Token); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Errorf("other sessions must be revoked, got %v", err)
	}
	e.login(t, "admin", "New-Mango-42-Rain")
}
