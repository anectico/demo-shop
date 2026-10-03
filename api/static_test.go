package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStaticSiteNeverLeavesItsDirectory(t *testing.T) {
	root := t.TempDir()
	site := filepath.Join(root, "site")
	if err := os.MkdirAll(filepath.Join(site, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		filepath.Join(site, "index.html"):       "INDEX",
		filepath.Join(site, "assets", "app.js"): "APP",
		filepath.Join(root, "secret.txt"):       "SECRET",
	} {
		if err := os.WriteFile(name, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.NoRoute(staticSite(site))

	get := func(p string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.URL.Path = p
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	for p, want := range map[string]string{"/assets/app.js": "APP", "/checkout": "INDEX", "/%2e%2e/secret.txt": "INDEX"} {
		if got := get(p).Body.String(); got != want {
			t.Errorf("GET %s = %q, want %q", p, got, want)
		}
	}
	// A path that climbs out of the site is refused or answered with the app
	// shell; the file beside the site is never served.
	for _, p := range []string{"/../secret.txt", "/assets/../../secret.txt", "/../site/../secret.txt"} {
		if rec := get(p); strings.Contains(rec.Body.String(), "SECRET") {
			t.Errorf("GET %s served the file outside the site", p)
		}
	}
}
