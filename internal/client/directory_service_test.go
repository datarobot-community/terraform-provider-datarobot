package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestListDirectoryEntitiesSendsTheNameAndTypeAsQueryParams(t *testing.T) {
	var query url.Values
	var path string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		query = r.URL.Query()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"count":      1,
			"totalCount": 1,
			"next":       nil,
			"previous":   nil,
			"data": []map[string]any{
				{"id": "grp-1", "name": "Finance Team", "entityType": "group", "provisioningSource": "scim"},
			},
		})
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	resp, err := svc.ListDirectoryEntities(context.Background(), &ListDirectoryEntitiesRequest{
		EntityType: "group",
		Name:       "Finance Team",
	})
	if err != nil {
		t.Fatalf("ListDirectoryEntities returned error: %v", err)
	}

	if path != "/directoryEntities/" {
		t.Errorf("expected /directoryEntities/, got %s", path)
	}
	if got := query.Get("entityType"); got != "group" {
		t.Errorf("expected entityType=group, got %q", got)
	}
	// A name with a space has to survive encoding, since group names carry them.
	if got := query.Get("name"); got != "Finance Team" {
		t.Errorf("expected the name to round trip, got %q", got)
	}
	// Zero valued paging fields carry omitempty, so they should not be sent.
	if _, ok := query["offset"]; ok {
		t.Errorf("expected no offset param, got %v", query["offset"])
	}
	if _, ok := query["limit"]; ok {
		t.Errorf("expected no limit param, got %v", query["limit"])
	}

	if resp.TotalCount != 1 {
		t.Errorf("expected a total count of 1, got %d", resp.TotalCount)
	}
	if len(resp.Data) != 1 || resp.Data[0].ID != "grp-1" {
		t.Fatalf("expected the single match to be parsed, got %+v", resp.Data)
	}
}

// The endpoint returns every match with a count rather than erroring, so the
// count has to survive parsing for the caller to reject an ambiguous name.
func TestListDirectoryEntitiesKeepsTheTotalCountForAmbiguousNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"count":      2,
			"totalCount": 2,
			"data": []map[string]any{
				{"id": "grp-1", "name": "Finance", "entityType": "group"},
				{"id": "grp-2", "name": "Finance", "entityType": "group"},
			},
		})
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	resp, err := svc.ListDirectoryEntities(context.Background(), &ListDirectoryEntitiesRequest{
		EntityType: "group",
		Name:       "Finance",
	})
	if err != nil {
		t.Fatalf("ListDirectoryEntities returned error: %v", err)
	}

	if resp.TotalCount != 2 {
		t.Errorf("expected a total count of 2, got %d", resp.TotalCount)
	}
}

func TestListDirectoryEntitiesToleratesANilRequest(t *testing.T) {
	var raw string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 0, "totalCount": 0, "data": []map[string]any{}})
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	if _, err := svc.ListDirectoryEntities(context.Background(), nil); err != nil {
		t.Fatalf("ListDirectoryEntities returned error: %v", err)
	}

	if raw != "/directoryEntities/" {
		t.Errorf("expected no query string when there is no request, got %s", raw)
	}
}
