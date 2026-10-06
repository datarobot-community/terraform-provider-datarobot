package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The listing's `next` link drops the entityId and entityType filters, so following it would
// return unrelated Use Cases: one filtered page is all the client reads, and a `next` link only
// reports that there are more.
func TestListUseCasesForEntityReadsOneFilteredPage(t *testing.T) {
	for name, next := range map[string]bool{"last page": false, "more pages": true} {
		t.Run(name, func(t *testing.T) {
			var requests []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.URL.String())
				query := r.URL.Query()
				if r.Method != http.MethodGet || r.URL.Path != "/useCases/" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if query.Get("entityId") != "wl-1" || query.Get("entityType") != "workload" || query.Get("limit") != "100" {
					t.Errorf("query = %s, want entityId=wl-1, entityType=workload and limit=100", r.URL.RawQuery)
				}
				page := map[string]any{
					"data": []map[string]any{{"id": "uc-1", "name": "first"}, {"id": "uc-2", "name": "second"}},
					"next": nil,
				}
				if next {
					page["next"] = "http://" + r.Host + "/useCases/?offset=100&limit=100"
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(page)
			}))
			defer server.Close()

			cfg := NewConfiguration("fake-token")
			cfg.Endpoint = server.URL
			useCases, more, err := NewService(NewClient(cfg)).ListUseCasesForEntity(context.Background(), "workload", "wl-1")
			if err != nil {
				t.Fatalf("ListUseCasesForEntity returned error: %v", err)
			}
			if len(useCases) != 2 || useCases[0].ID != "uc-1" || useCases[1].ID != "uc-2" {
				t.Errorf("got %+v, want uc-1 and uc-2", useCases)
			}
			if more != next {
				t.Errorf("more = %v, want %v", more, next)
			}
			if len(requests) != 1 {
				t.Errorf("got %d requests, want 1: %v", len(requests), requests)
			}
		})
	}
}
