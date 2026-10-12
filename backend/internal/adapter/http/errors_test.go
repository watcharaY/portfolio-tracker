package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
)

func TestErrorHandler_MapsDomainErrors(t *testing.T) {
	cases := []struct {
		err    error
		status int
		field  string
	}{
		{domain.ErrNotFound, 404, ""},
		{fmt.Errorf("wrapped: %w", domain.ErrConflict), 409, ""},
		{domain.ErrReferenceNotFound, 409, ""},
		{domain.ErrInvalidCredentials, 401, ""},
		{domain.ErrUnauthenticated, 401, ""},
		{domain.ErrPasswordChangeRequired, 403, ""},
		{&domain.ValidationError{Field: "amount", Message: "must be greater than 0"}, 400, "amount"},
		{fiber.ErrTooManyRequests, 429, ""},
		{errors.New("db exploded"), 500, ""},
	}
	for _, c := range cases {
		app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
		app.Get("/", func(fiber.Ctx) error { return c.err })

		resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
		if err != nil {
			t.Fatal(err)
		}
		var body errorBody
		raw, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("%v: body %q is not JSON", c.err, raw)
		}
		if resp.StatusCode != c.status || body.Field != c.field {
			t.Errorf("%v: got %d field=%q, want %d field=%q", c.err, resp.StatusCode, body.Field, c.status, c.field)
		}
		if c.status == 500 && body.Error != "internal server error" {
			t.Errorf("internal error details leaked: %q", body.Error)
		}
	}
}

func TestDecodeJSON(t *testing.T) {
	type req struct {
		Name   string       `json:"name"`
		Amount domain.Money `json:"amount"`
	}
	if _, err := decodeJSON[req]([]byte(`{"name":"a","amount":100}`)); err != nil {
		t.Fatalf("valid body rejected: %v", err)
	}
	var ve *domain.ValidationError
	if _, err := decodeJSON[req]([]byte(`{"amount":1.5}`)); !errors.As(err, &ve) || ve.Field != "amount" {
		t.Errorf("fractional money: got %v", err)
	}
	for _, body := range []string{`{"name":"a","extra":1}`, `{"name":"a"} {}`, `not json`} {
		if _, err := decodeJSON[req]([]byte(body)); err == nil {
			t.Errorf("%s: want error", body)
		}
	}
}

func TestRequireJSONBody(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Use(requireJSONBody)
	app.All("/", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })

	cases := []struct {
		method, contentType string
		status              int
	}{
		{"POST", "application/json", 204},
		{"PUT", "application/json; charset=utf-8", 204},
		{"POST", "text/plain", 415},
		{"POST", "application/x-www-form-urlencoded", 415},
		{"PUT", "", 415},
		{"GET", "", 204},
		{"DELETE", "", 204},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, "/", nil)
		if c.contentType != "" {
			req.Header.Set("Content-Type", c.contentType)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != c.status {
			t.Errorf("%s %q: got %d, want %d", c.method, c.contentType, resp.StatusCode, c.status)
		}
	}
}
