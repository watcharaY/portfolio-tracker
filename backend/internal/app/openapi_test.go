package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	httpadapter "github.com/watcharaY/portfolio-tracker/backend/internal/adapter/http"
	"github.com/watcharaY/portfolio-tracker/backend/internal/adapter/http/openapi"
)

// Every call the API tests make is checked against openapi.yaml, so the document
// cannot drift from the code without failing CI.

var (
	specOnce   sync.Once
	specDoc    *openapi3.T
	specRouter routers.Router
	specErr    error
)

func loadSpec(t *testing.T) (*openapi3.T, routers.Router) {
	t.Helper()
	specOnce.Do(func() {
		loader := openapi3.NewLoader()
		specDoc, specErr = loader.LoadFromData(openapi.Spec)
		if specErr == nil {
			specErr = specDoc.Validate(context.Background())
		}
		if specErr == nil {
			specRouter, specErr = gorillamux.NewRouter(specDoc)
		}
	})
	if specErr != nil {
		t.Fatalf("openapi.yaml: %v", specErr)
	}
	return specDoc, specRouter
}

// checkContract validates one exchange. A request the server accepted (2xx) must
// also be valid per the document; every response must match its documented status
// and, for JSON, its schema. Swagger UI's static files are not API operations.
func checkContract(t *testing.T, method, path, contentType string, reqBody []byte, status int, header http.Header, respBody []byte) {
	t.Helper()
	if strings.HasPrefix(path, "/docs/") {
		return
	}
	_, router := loadSpec(t)
	req := httptest.NewRequest(method, "/api"+path, bytes.NewReader(reqBody))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	route, params, err := router.FindRoute(req)
	if err != nil {
		t.Errorf("%s %s answered %d but has no operation in openapi.yaml: %v", method, path, status, err)
		return
	}
	input := &openapi3filter.RequestValidationInput{
		Request: req, PathParams: params, Route: route,
		Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
	}
	if status < 300 {
		if err := openapi3filter.ValidateRequest(context.Background(), input); err != nil {
			t.Errorf("%s %s: the server accepted a request openapi.yaml rejects: %v", method, path, err)
		}
	}
	isJSON := strings.HasPrefix(header.Get("Content-Type"), "application/json")
	err = openapi3filter.ValidateResponse(context.Background(), &openapi3filter.ResponseValidationInput{
		RequestValidationInput: input,
		Status:                 status,
		Header:                 header,
		Body:                   io.NopCloser(bytes.NewReader(respBody)),
		Options:                &openapi3filter.Options{IncludeResponseStatus: true, ExcludeResponseBody: !isJSON},
	})
	if err != nil {
		t.Errorf("%s %s -> %d does not match openapi.yaml: %v", method, path, status, err)
	}
}

func TestOpenAPI_DocumentsEveryRoute(t *testing.T) {
	doc, _ := loadSpec(t)
	app := httpadapter.NewServer(httpadapter.Config{}, httpadapter.Services{})

	var served []string
	for _, r := range app.GetRoutes(true) {
		if !strings.HasPrefix(r.Path, "/api/") || !slices.Contains([]string{"GET", "POST", "PUT", "DELETE"}, r.Method) {
			continue
		}
		p := strings.TrimPrefix(r.Path, "/api")
		if p == "/docs/*" {
			continue
		}
		served = append(served, r.Method+" "+strings.ReplaceAll(p, ":id", "{id}"))
	}

	var documented []string
	for p, item := range doc.Paths.Map() {
		for m := range item.Operations() {
			documented = append(documented, m+" "+p)
		}
	}
	slices.Sort(served)
	served = slices.Compact(served)
	slices.Sort(documented)

	for _, op := range served {
		if !slices.Contains(documented, op) {
			t.Errorf("%s is served but missing from openapi.yaml", op)
		}
	}
	for _, op := range documented {
		if !slices.Contains(served, op) {
			t.Errorf("%s is in openapi.yaml but not served", op)
		}
	}
}

func TestAPI_Docs(t *testing.T) {
	c, want := newAPI(t)
	c.expect(http.MethodGet, "/docs", nil, http.StatusUnauthorized)
	c.expect(http.MethodGet, "/openapi.yaml", nil, http.StatusUnauthorized)

	c.login(want.Users.Ready)
	for path, typ := range map[string]string{
		"/openapi.yaml":                         "application/yaml",
		"/docs":                                 "text/html",
		"/docs/docs.js":                         "text/javascript",
		"/docs/swagger-ui/swagger-ui-bundle.js": "text/javascript",
		"/docs/swagger-ui/swagger-ui.css":       "text/css",
	} {
		r := c.expect(http.MethodGet, path, nil, http.StatusOK)
		if got := r.header.Get("Content-Type"); !strings.HasPrefix(got, typ) {
			t.Errorf("GET %s Content-Type = %q, want %s", path, got, typ)
		}
	}
	c.expect(http.MethodGet, "/docs/missing.js", nil, http.StatusNotFound)
	c.expect(http.MethodGet, "/docs/../openapi.go", nil, http.StatusNotFound)
}
