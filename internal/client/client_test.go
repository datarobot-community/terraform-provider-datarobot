package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A redirect (e.g. to an SSO login page) must not be silently followed and
// its body returned as if it were the requested resource - see the
// artifact-build-log-tail PR review on getRaw.
func TestGetRawRejectsRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/artifacts/art-1/builds/build-1/logs" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("<html>please log in</html>"))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	c := NewClient(cfg)

	body, err := getRaw(c, context.Background(), "/artifacts/art-1/builds/build-1/logs")
	if err == nil {
		t.Fatalf("expected error for redirected request, got body: %s", body)
	}
	if !strings.Contains(err.Error(), "redirected") {
		t.Fatalf("expected a redirect error, got: %v", err)
	}
}
