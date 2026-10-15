package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A redirect (e.g. to an SSO login page) must not be silently followed and its body
// returned as if it were the requested resource.
func TestGetRejectsRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/artifacts/art-1/builds/build-1/" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "login-page"}`))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL

	build, err := Get[ArtifactBuild](NewClient(cfg), context.Background(), "/artifacts/art-1/builds/build-1/")
	if err == nil || !strings.Contains(err.Error(), "redirected") {
		t.Fatalf("expected a redirect error, got %+v, %v", build, err)
	}
}
