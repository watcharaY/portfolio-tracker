package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"

	httpadapter "github.com/watcharaY/portfolio-tracker/backend/internal/adapter/http"
	"github.com/watcharaY/portfolio-tracker/backend/internal/adapter/postgres"
	"github.com/watcharaY/portfolio-tracker/backend/internal/platform/config"
)

// These tests drive the real HTTP server, wired exactly as Run wires it, against
// a Postgres loaded with testdata/e2e-seed.sql.

type expectedUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type e2eExpected struct {
	Dashboard json.RawMessage `json:"dashboard"`
	Users     struct {
		ForcedChange expectedUser `json:"forcedChange"`
		Ready        expectedUser `json:"ready"`
	} `json:"users"`
}

type apiClient struct {
	t      *testing.T
	app    *fiber.App
	cookie string
}

type apiResponse struct {
	status int
	body   []byte
}

func (r apiResponse) object(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(r.body, &m); err != nil {
		t.Fatalf("decode object %s: %v", r.body, err)
	}
	return m
}

func (r apiResponse) list(t *testing.T) []map[string]any {
	t.Helper()
	var l []map[string]any
	if err := json.Unmarshal(r.body, &l); err != nil {
		t.Fatalf("decode list %s: %v", r.body, err)
	}
	return l
}

func (c *apiClient) do(method, path, contentType string, body []byte) apiResponse {
	c.t.Helper()
	req := httptest.NewRequest(method, "/api"+path, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	// bcrypt under -race can take over the default 1s per request.
	resp, err := c.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second, FailOnTimeout: true})
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	for _, ck := range resp.Cookies() {
		if ck.Name == "pt_session" {
			c.cookie = ck.Name + "=" + ck.Value
		}
	}
	out, _ := io.ReadAll(resp.Body)
	return apiResponse{resp.StatusCode, out}
}

func (c *apiClient) json(method, path string, body any) apiResponse {
	c.t.Helper()
	var raw []byte
	if body != nil {
		var err error
		if raw, err = json.Marshal(body); err != nil {
			c.t.Fatal(err)
		}
	}
	ct := ""
	if method == http.MethodPost || method == http.MethodPut {
		ct = "application/json"
	}
	return c.do(method, path, ct, raw)
}

func (c *apiClient) expect(method, path string, body any, want int) apiResponse {
	c.t.Helper()
	r := c.json(method, path, body)
	if r.status != want {
		c.t.Fatalf("%s %s = %d %s, want %d", method, path, r.status, r.body, want)
	}
	return r
}

func (c *apiClient) login(u expectedUser) {
	c.t.Helper()
	c.expect(http.MethodPost, "/auth/login", map[string]string{"username": u.Username, "password": u.Password}, http.StatusOK)
}

// newAPI reloads the e2e seed and returns an anonymous client for a fresh server.
func newAPI(t *testing.T) (*apiClient, e2eExpected) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := postgres.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	if err := postgres.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	seed, err := os.ReadFile("../../testdata/e2e-seed.sql")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, string(seed)); err != nil {
		t.Fatalf("load e2e seed: %v", err)
	}

	raw, err := os.ReadFile("../../testdata/e2e-expected.json")
	if err != nil {
		t.Fatal(err)
	}
	var want e2eExpected
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}

	services, err := buildServices(db, config.Config{SessionTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return &apiClient{t: t, app: httpadapter.NewServer(httpadapter.Config{}, services)}, want
}

func TestAPI_Auth(t *testing.T) {
	c, want := newAPI(t)

	c.expect(http.MethodGet, "/health", nil, http.StatusOK)
	c.expect(http.MethodGet, "/auth/me", nil, http.StatusUnauthorized)
	c.expect(http.MethodGet, "/funds", nil, http.StatusUnauthorized)
	c.expect(http.MethodPost, "/auth/login", map[string]string{"username": "e2e", "password": "wrong"}, http.StatusUnauthorized)
	if r := c.do(http.MethodPost, "/auth/login", "text/plain", []byte("e2e")); r.status != http.StatusUnsupportedMediaType {
		t.Fatalf("non-JSON login = %d, want 415", r.status)
	}

	c.login(want.Users.Ready)
	if me := c.expect(http.MethodGet, "/auth/me", nil, http.StatusOK).object(t); me["username"] != "e2e" || me["mustChangePassword"] != false {
		t.Fatalf("me = %v", me)
	}
	c.expect(http.MethodGet, "/funds", nil, http.StatusOK)
	c.expect(http.MethodPost, "/auth/logout", nil, http.StatusNoContent)
	c.expect(http.MethodGet, "/funds", nil, http.StatusUnauthorized)
}

func TestAPI_ForcedPasswordChange(t *testing.T) {
	c, want := newAPI(t)
	admin := want.Users.ForcedChange

	c.login(admin)
	if me := c.expect(http.MethodGet, "/auth/me", nil, http.StatusOK).object(t); me["mustChangePassword"] != true {
		t.Fatalf("me = %v", me)
	}
	c.expect(http.MethodGet, "/funds", nil, http.StatusForbidden)
	c.expect(http.MethodGet, "/dashboard", nil, http.StatusForbidden)

	c.expect(http.MethodPost, "/auth/password",
		map[string]string{"currentPassword": admin.Password, "newPassword": admin.Password}, http.StatusBadRequest)
	c.expect(http.MethodPost, "/auth/password",
		map[string]string{"currentPassword": "wrong", "newPassword": "Another-Pass-77"}, http.StatusBadRequest)
	c.expect(http.MethodPost, "/auth/password",
		map[string]string{"currentPassword": admin.Password, "newPassword": "Another-Pass-77"}, http.StatusOK)

	c.expect(http.MethodGet, "/funds", nil, http.StatusOK)
	c.expect(http.MethodPost, "/auth/logout", nil, http.StatusNoContent)
	c.expect(http.MethodPost, "/auth/login", map[string]string{"username": admin.Username, "password": admin.Password}, http.StatusUnauthorized)
	c.login(expectedUser{admin.Username, "Another-Pass-77"})
}

func TestAPI_Dashboard(t *testing.T) {
	c, want := newAPI(t)
	c.login(want.Users.Ready)

	var got, exp struct {
		Total                     map[string]any  `json:"total"`
		Classes                   json.RawMessage `json:"classes"`
		EstSavingsInterestPerYear float64         `json:"estSavingsInterestPerYear"`
	}
	if err := json.Unmarshal(c.expect(http.MethodGet, "/dashboard", nil, http.StatusOK).body, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want.Dashboard, &exp); err != nil {
		t.Fatal(err)
	}

	var gotClasses []map[string]any
	var expClasses map[string]map[string]float64
	if err := json.Unmarshal(got.Classes, &gotClasses); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(exp.Classes, &expClasses); err != nil {
		t.Fatal(err)
	}
	compare := func(name string, got map[string]any, want map[string]float64) {
		t.Helper()
		for k, w := range want {
			g, ok := got[k].(float64)
			if !ok || math.Abs(g-w) > 5e-7 {
				t.Errorf("%s.%s = %v, want %v", name, k, got[k], w)
			}
		}
	}
	var expTotal map[string]float64
	if err := json.Unmarshal(mustJSON(t, exp.Total), &expTotal); err != nil {
		t.Fatal(err)
	}
	compare("total", got.Total, expTotal)
	if len(gotClasses) != len(expClasses) {
		t.Fatalf("got %d classes, want %d", len(gotClasses), len(expClasses))
	}
	for _, cl := range gotClasses {
		key, _ := cl["key"].(string)
		compare(key, cl, expClasses[key])
	}
	if got.EstSavingsInterestPerYear != exp.EstSavingsInterestPerYear {
		t.Errorf("interest = %v, want %v", got.EstSavingsInterestPerYear, exp.EstSavingsInterestPerYear)
	}
}

// resourceCase describes one CRUD endpoint. seeded is the row count from the seed;
// update must change field to value so the test can see the write land.
type resourceCase struct {
	path    string
	seeded  int
	create  map[string]any
	update  map[string]any
	field   string
	value   float64
	invalid map[string]any
}

var resourceCases = []resourceCase{
	{
		path: "/brokers", seeded: 2,
		create: map[string]any{"name": "Broker New", "current_value": 100, "note": ""},
		update: map[string]any{"name": "Broker New", "current_value": 200, "note": "edited"},
		field:  "current_value", value: 200,
		invalid: map[string]any{"name": " ", "current_value": 0, "note": ""},
	},
	{
		path: "/broker-transactions", seeded: 5,
		create: map[string]any{"broker_id": 1, "txn_date": "2026-05-01", "type": "deposit", "amount": 100, "note": ""},
		update: map[string]any{"broker_id": 1, "txn_date": "2026-05-02", "type": "withdraw", "amount": 200, "note": ""},
		field:  "amount", value: 200,
		invalid: map[string]any{"broker_id": 1, "txn_date": "2026-05-01", "type": "buy", "amount": 100, "note": ""},
	},
	{
		path: "/funds", seeded: 9,
		create: map[string]any{"name": "Fund New", "category": "Gold", "cost": 100, "market_value": 110, "note": ""},
		update: map[string]any{"name": "Fund New", "category": "Gold", "cost": 100, "market_value": 120, "note": ""},
		field:  "market_value", value: 120,
		invalid: map[string]any{"name": "Fund New", "category": "Crypto", "cost": 100, "market_value": 110, "note": ""},
	},
	{
		path: "/pvd-years", seeded: 2,
		create: map[string]any{"year": 2030, "self_contribution": 100, "company_contribution": 50, "self_profit": 10, "company_profit": 5, "fund_allocation": ""},
		update: map[string]any{"year": 2030, "self_contribution": 300, "company_contribution": 50, "self_profit": -10, "company_profit": 5, "fund_allocation": "edited"},
		field:  "self_contribution", value: 300,
		invalid: map[string]any{"year": 1800, "self_contribution": 100, "company_contribution": 50, "self_profit": 0, "company_profit": 0, "fund_allocation": ""},
	},
	{
		path: "/savings-accounts", seeded: 3,
		create: map[string]any{"bank": "Bank New", "balance": 100, "interest_bp": 100, "note": ""},
		update: map[string]any{"bank": "Bank New", "balance": 200, "interest_bp": 100, "note": ""},
		field:  "balance", value: 200,
		invalid: map[string]any{"bank": "Bank New", "balance": 100, "interest_bp": 10001, "note": ""},
	},
	{
		path: "/insurers", seeded: 3,
		create: map[string]any{"company": "Insurer New", "note": ""},
		update: map[string]any{"company": "Insurer New", "note": "edited"},
		field:  "payments", value: 0,
		invalid: map[string]any{"company": "", "note": ""},
	},
	{
		path: "/insurance-payments", seeded: 4,
		create: map[string]any{"insurer_id": 1, "year": 2030, "premium": 100, "note": ""},
		update: map[string]any{"insurer_id": 1, "year": 2030, "premium": 200, "note": ""},
		field:  "premium", value: 200,
		invalid: map[string]any{"insurer_id": 1, "year": 2030, "premium": 0, "note": ""},
	},
	{
		path: "/salary-months", seeded: 15,
		create: map[string]any{"year": 2030, "month": 1, "salary": 100, "bonus": 0, "withholding_tax": 0, "pvd": 0, "sso": 0},
		update: map[string]any{"year": 2030, "month": 1, "salary": 200, "bonus": 0, "withholding_tax": 0, "pvd": 0, "sso": 0},
		field:  "salary", value: 200,
		invalid: map[string]any{"year": 2030, "month": 13, "salary": 100, "bonus": 0, "withholding_tax": 0, "pvd": 0, "sso": 0},
	},
	{
		path: "/tax-deductions", seeded: 4,
		create: map[string]any{"year": 2030, "item": "New item", "amount": 100},
		update: map[string]any{"year": 2030, "item": "New item", "amount": 200},
		field:  "amount", value: 200,
		invalid: map[string]any{"year": 2030, "item": " ", "amount": 100},
	},
}

func TestAPI_ResourceCRUD(t *testing.T) {
	for _, rc := range resourceCases {
		t.Run(strings.TrimPrefix(rc.path, "/"), func(t *testing.T) {
			c, want := newAPI(t)
			c.login(want.Users.Ready)

			if n := len(c.expect(http.MethodGet, rc.path, nil, http.StatusOK).list(t)); n != rc.seeded {
				t.Fatalf("list = %d rows, want %d from the seed", n, rc.seeded)
			}
			first := c.expect(http.MethodGet, rc.path+"/1", nil, http.StatusOK).body
			c.expect(http.MethodGet, rc.path+"/9999", nil, http.StatusNotFound)

			c.expect(http.MethodPost, rc.path, rc.invalid, http.StatusBadRequest)
			withUnknown := map[string]any{"unknown_field": 1}
			for k, v := range rc.create {
				withUnknown[k] = v
			}
			c.expect(http.MethodPost, rc.path, withUnknown, http.StatusBadRequest)

			created := c.expect(http.MethodPost, rc.path, rc.create, http.StatusCreated).object(t)
			id := int64(created["id"].(float64))
			item := rc.path + "/" + jsonNumber(id)

			c.expect(http.MethodPut, item, rc.update, http.StatusOK)
			if got := c.expect(http.MethodGet, item, nil, http.StatusOK).object(t)[rc.field]; got != rc.value {
				t.Fatalf("after update %s = %v, want %v", rc.field, got, rc.value)
			}
			c.expect(http.MethodPut, item, rc.invalid, http.StatusBadRequest)
			c.expect(http.MethodPut, rc.path+"/9999", rc.update, http.StatusNotFound)
			if after := c.expect(http.MethodGet, rc.path+"/1", nil, http.StatusOK).body; !bytes.Equal(after, first) {
				t.Fatalf("writes to row %d changed row 1:\nbefore %s\nafter  %s", id, first, after)
			}

			c.expect(http.MethodDelete, item, nil, http.StatusNoContent)
			c.expect(http.MethodGet, item, nil, http.StatusNotFound)
			c.expect(http.MethodDelete, item, nil, http.StatusNotFound)
			if n := len(c.expect(http.MethodGet, rc.path, nil, http.StatusOK).list(t)); n != rc.seeded {
				t.Fatalf("list after delete = %d rows, want %d", n, rc.seeded)
			}
		})
	}
}

// The seeded rows themselves must be editable, not only rows the test created.
func TestAPI_UpdateSeededRows(t *testing.T) {
	for _, rc := range resourceCases {
		t.Run(strings.TrimPrefix(rc.path, "/"), func(t *testing.T) {
			c, want := newAPI(t)
			c.login(want.Users.Ready)
			row := c.expect(http.MethodGet, rc.path+"/1", nil, http.StatusOK).object(t)
			body := map[string]any{}
			for k := range rc.update {
				body[k] = row[k]
			}
			c.expect(http.MethodPut, rc.path+"/1", body, http.StatusOK)
		})
	}
}

func TestAPI_Filters(t *testing.T) {
	c, want := newAPI(t)
	c.login(want.Users.Ready)
	for path, n := range map[string]int{
		"/salary-months?year=2025":         12,
		"/salary-months?year=2026":         3,
		"/salary-months?year=2031":         0,
		"/tax-deductions?year=2025":        3,
		"/tax-deductions?year=2026":        1,
		"/broker-transactions?broker_id=1": 3,
		"/broker-transactions?broker_id=2": 2,
		"/insurance-payments?insurer_id=1": 3,
		"/insurance-payments?insurer_id=3": 0,
	} {
		if got := len(c.expect(http.MethodGet, path, nil, http.StatusOK).list(t)); got != n {
			t.Errorf("GET %s = %d rows, want %d", path, got, n)
		}
	}
	c.expect(http.MethodGet, "/salary-months?year=abc", nil, http.StatusBadRequest)
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func jsonNumber(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
