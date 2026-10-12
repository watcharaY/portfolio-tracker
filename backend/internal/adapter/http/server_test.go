package http

import (
	"io"
	"io/fs"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

type response struct {
	status       int
	contentType  string
	cacheControl string
	body         string
}

func get(t *testing.T, webUI fs.FS, path string) response {
	t.Helper()
	var cfg Config
	if webUI != nil {
		cfg.WebUI = webUI
	}
	resp, err := NewServer(cfg, Services{}).Test(httptest.NewRequest("GET", path, nil))
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
	cases := []struct {
		path string
		want response
	}{
		{"/", response{200, "text/html; charset=utf-8", "no-cache", "<!doctype html><div id=root></div>"}},
		{"/brokerage", response{200, "text/html; charset=utf-8", "no-cache", "<!doctype html><div id=root></div>"}},
		{"/assets/app.js", response{200, "text/javascript; charset=utf-8", "", "console.log(1)"}},
		{"/index.html", response{200, "text/html; charset=utf-8", "no-cache", "<!doctype html><div id=root></div>"}},
		{"/api/health", response{200, "application/json; charset=utf-8", "", `{"ok":true}`}},
		{"/api/x", response{401, "application/json; charset=utf-8", "", `{"error":"not authenticated"}`}},
	}
	for _, c := range cases {
		if got := get(t, webUI, c.path); got != c.want {
			t.Errorf("GET %s = %+v, want %+v", c.path, got, c.want)
		}
	}
}

func TestServer_APIOnlyWithoutWebUI(t *testing.T) {
	want := response{404, "application/json; charset=utf-8", "", `{"error":"Not Found"}`}
	for _, path := range []string{"/", "/brokerage", "/index.html"} {
		if got := get(t, nil, path); got != want {
			t.Errorf("GET %s = %+v, want %+v", path, got, want)
		}
	}
}
