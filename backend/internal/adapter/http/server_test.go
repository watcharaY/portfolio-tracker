package http

import (
	"context"
	"io"
	"io/fs"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type response struct {
	status       int
	contentType  string
	cacheControl string
	body         string
}

// These stubs make every token a valid session, so an unknown /api path
// reaches the API's own 404 instead of stopping at the auth guard's 401.
type anyUser struct{ usecase.UserRepository }

func (anyUser) GetByID(context.Context, int64) (domain.User, error) { return domain.User{ID: 1}, nil }

type anySession struct{ usecase.SessionRepository }

func (anySession) GetValid(context.Context, string, time.Time) (domain.Session, error) {
	return domain.Session{UserID: 1}, nil
}

type noHasher struct{ usecase.PasswordHasher }

func (noHasher) Hash(string) (string, error) { return "", nil }

type plainTokens struct{ usecase.TokenIssuer }

func (plainTokens) Hash(token string) string { return token }

func get(t *testing.T, webUI fs.FS, path string) response {
	t.Helper()
	auth, err := usecase.NewAuthService(usecase.AuthDeps{
		Users: anyUser{}, Sessions: anySession{}, Hasher: noHasher{}, Tokens: plainTokens{},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("Cookie", sessionCookie+"=token")
	resp, err := NewServer(Config{WebUI: webUI}, Services{Auth: auth}).Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	return response{resp.StatusCode, resp.Header.Get("Content-Type"), resp.Header.Get("Cache-Control"), string(body)}
}

func TestServer_WithWebUI(t *testing.T) {
	webUI := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><div id=root></div>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}
	index := response{200, "text/html; charset=utf-8", "no-cache", "<!doctype html><div id=root></div>"}
	cases := []struct {
		path string
		want response
	}{
		{"/", index},
		{"/brokerage", index},
		{"/index.html", index},
		{"/assets/app.js", response{200, "text/javascript; charset=utf-8", "", "console.log(1)"}},
		{"/api/health", response{200, "application/json; charset=utf-8", "no-store", `{"ok":true}`}},
		{"/api/x", response{404, "application/json; charset=utf-8", "no-store", `{"error":"Not Found"}`}},
	}
	for _, c := range cases {
		if got := get(t, webUI, c.path); got != c.want {
			t.Errorf("GET %s = %+v, want %+v", c.path, got, c.want)
		}
	}
}

func TestServer_APIOnlyWithoutWebUI(t *testing.T) {
	for _, path := range []string{"/", "/brokerage", "/index.html", "/api/x"} {
		want := response{404, "application/json; charset=utf-8", "", `{"error":"Not Found"}`}
		if strings.HasPrefix(path, "/api/") {
			want.cacheControl = "no-store"
		}
		if got := get(t, nil, path); got != want {
			t.Errorf("GET %s = %+v, want %+v", path, got, want)
		}
	}
}
