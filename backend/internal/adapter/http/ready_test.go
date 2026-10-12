package http

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
)

func TestReady(t *testing.T) {
	cases := []struct {
		name   string
		ready  func(context.Context) error
		status int
		body   string
	}{
		{"database reachable", func(context.Context) error { return nil }, 200, `{"ok":true}`},
		{"database down", func(context.Context) error { return errors.New("dial tcp: connection refused") }, 503, `{"error":"database unavailable"}`},
	}
	for _, c := range cases {
		resp, err := NewServer(Config{}, Services{Ready: c.ready}).Test(httptest.NewRequest("GET", "/api/ready", nil))
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != c.status || string(body) != c.body {
			t.Errorf("%s: GET /api/ready = %d %s, want %d %s", c.name, resp.StatusCode, body, c.status, c.body)
		}
	}
}
