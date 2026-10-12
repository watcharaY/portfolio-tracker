package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

// testServer builds the API with an anonymous auth service and captures the
// access log as JSON lines.
func testServer(t *testing.T, cfg Config) (*fiber.App, *bytes.Buffer) {
	t.Helper()
	auth, err := usecase.NewAuthService(usecase.AuthDeps{
		Users: anyUser{}, Sessions: noSession{}, Hasher: noHasher{}, Tokens: plainTokens{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	cfg.Logger = slog.New(slog.NewJSONHandler(&logs, nil))
	ready := func(context.Context) error { return nil }
	return NewServer(cfg, Services{Auth: auth, Ready: ready}), &logs
}

type noSession struct{ usecase.SessionRepository }

func (noSession) GetValid(context.Context, string, time.Time) (domain.Session, error) {
	return domain.Session{}, domain.ErrNotFound
}

func send(t *testing.T, app *fiber.App, req *http.Request) *http.Response {
	t.Helper()
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func lastLog(t *testing.T, logs *bytes.Buffer) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &entry); err != nil {
		t.Fatalf("access log is not JSON: %q", lines[len(lines)-1])
	}
	return entry
}

func TestAccessLog_FieldsWithoutSecrets(t *testing.T) {
	app, logs := testServer(t, Config{})
	req := httptest.NewRequest("GET", "/api/funds/7?token=abc", nil)
	req.Header.Set("Cookie", sessionCookie+"=secret-session-token")
	req.Header.Set("User-Agent", "e2e-agent")
	resp := send(t, app, req)
	if resp.StatusCode != 401 {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}

	entry := lastLog(t, logs)
	for key, want := range map[string]any{
		"msg": "request", "level": "INFO", "method": "GET", "path": "/api/funds/7",
		"status": float64(401), "user_agent": "e2e-agent",
	} {
		if entry[key] != want {
			t.Errorf("log %s = %v, want %v", key, entry[key], want)
		}
	}
	for _, key := range []string{"request_id", "route", "latency_ms", "bytes", "ip"} {
		if _, ok := entry[key]; !ok {
			t.Errorf("log has no %s: %v", key, entry)
		}
	}
	if entry["request_id"] != resp.Header.Get("X-Request-ID") {
		t.Errorf("log request_id %v differs from response header %q", entry["request_id"], resp.Header.Get("X-Request-ID"))
	}
	for _, secret := range []string{"secret-session-token", "token=abc", "Cookie"} {
		if strings.Contains(logs.String(), secret) {
			t.Errorf("access log leaks %q: %s", secret, logs.String())
		}
	}
}

func TestAccessLog_Levels(t *testing.T) {
	for status, want := range map[int]slog.Level{200: slog.LevelInfo, 401: slog.LevelInfo, 404: slog.LevelInfo, 415: slog.LevelWarn, 500: slog.LevelError} {
		if got := accessLevel(status); got != want {
			t.Errorf("accessLevel(%d) = %v, want %v", status, got, want)
		}
	}
}

func TestClientIP_TrustedProxyOnly(t *testing.T) {
	// app.Test connections come from 0.0.0.0.
	cases := []struct {
		name    string
		proxies []string
		want    string
	}{
		{"no trusted proxy ignores X-Forwarded-For", nil, "0.0.0.0"},
		{"untrusted peer ignores X-Forwarded-For", []string{"10.0.0.0/8"}, "0.0.0.0"},
		{"trusted peer sets the client IP", []string{"0.0.0.0/32"}, "203.0.113.7"},
	}
	for _, c := range cases {
		app, logs := testServer(t, Config{TrustedProxies: c.proxies})
		req := httptest.NewRequest("GET", "/api/health", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.7")
		send(t, app, req)
		if got := lastLog(t, logs)["ip"]; got != c.want {
			t.Errorf("%s: ip = %v, want %s", c.name, got, c.want)
		}
	}
}

func TestRequestID(t *testing.T) {
	app, _ := testServer(t, Config{})
	cases := []struct{ incoming, want string }{
		{"abc-123", "abc-123"},
		{"", ""},
		{"ไม่ใช่ ASCII", ""},
	}
	for _, c := range cases {
		req := httptest.NewRequest("GET", "/api/health", nil)
		if c.incoming != "" {
			req.Header.Set("X-Request-ID", c.incoming)
		}
		got := send(t, app, req).Header.Get("X-Request-ID")
		switch {
		case c.want != "" && got != c.want:
			t.Errorf("incoming %q: X-Request-ID = %q, want it echoed", c.incoming, got)
		case c.want == "" && (got == "" || got == c.incoming):
			t.Errorf("incoming %q: X-Request-ID = %q, want a freshly generated id", c.incoming, got)
		}
	}
}

func TestRateLimit(t *testing.T) {
	app, _ := testServer(t, Config{RateLimitPerMinute: 2})
	for i := 1; i <= 3; i++ {
		resp := send(t, app, httptest.NewRequest("GET", "/api/auth/me", nil))
		if i <= 2 {
			if resp.StatusCode != 401 || resp.Header.Get("X-RateLimit-Limit") != "2" {
				t.Fatalf("request %d = %d (limit header %q), want 401 under the limit", i, resp.StatusCode, resp.Header.Get("X-RateLimit-Limit"))
			}
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 429 || resp.Header.Get("Retry-After") == "" || string(body) != `{"error":"too many requests, retry later"}` {
			t.Fatalf("request %d = %d %s (Retry-After %q), want 429 JSON with Retry-After", i, resp.StatusCode, body, resp.Header.Get("Retry-After"))
		}
	}
	for i := 0; i < 5; i++ {
		if resp := send(t, app, httptest.NewRequest("GET", "/api/health", nil)); resp.StatusCode != 200 {
			t.Fatalf("health check %d = %d, want 200: health checks are not rate limited", i, resp.StatusCode)
		}
	}
}

func TestAPIResponsesAreNotCached(t *testing.T) {
	app, _ := testServer(t, Config{})
	for _, path := range []string{"/api/health", "/api/auth/me", "/api/nope"} {
		resp := send(t, app, httptest.NewRequest("GET", path, nil))
		if got := resp.Header.Get("Cache-Control"); got != "no-store" {
			t.Errorf("GET %s Cache-Control = %q, want no-store", path, got)
		}
		if resp.Header.Get("Server") != "" || resp.Header.Get("X-Powered-By") != "" {
			t.Errorf("GET %s reveals the server: %v", path, resp.Header)
		}
	}
}
