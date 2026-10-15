package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The listing's `next` link keeps the filters, so every page is read.
func TestListKeyValuesReadsEveryFilteredPage(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.String())
		query := r.URL.Query()
		if r.Method != http.MethodGet || r.URL.Path != "/keyValues/" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if query.Get("entityType") != "workload" || query.Get("entityId") != "wl-1" || query.Get("category") != "tag" {
			t.Errorf("query = %s, want entityType=workload, entityId=wl-1 and category=tag", r.URL.RawQuery)
		}
		page := map[string]any{
			"data": []map[string]any{{"id": "kv-1", "name": "team", "value": "ml", "category": "tag", "isEditable": true}},
			"next": "http://" + r.Host + "/api/v2/keyValues/?entityType=workload&entityId=wl-1&category=tag&offset=1&limit=1",
		}
		if query.Get("offset") == "1" {
			page = map[string]any{
				"data": []map[string]any{{"id": "kv-2", "name": "datarobot.type", "value": "custom-application", "category": "tag", "isEditable": false}},
				"next": nil,
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(page)
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	keyValues, err := NewService(NewClient(cfg)).ListKeyValues(context.Background(), &ListKeyValuesRequest{
		EntityType: "workload", EntityID: "wl-1", Category: KeyValueCategoryTag,
	})
	if err != nil {
		t.Fatalf("ListKeyValues returned error: %v", err)
	}
	if len(keyValues) != 2 || keyValues[0].ID != "kv-1" || keyValues[0].Name != "team" || keyValues[0].Value != "ml" || keyValues[1].ID != "kv-2" {
		t.Fatalf("got %+v, want kv-1 then kv-2", keyValues)
	}
	if keyValues[0].IsEditable == nil || !*keyValues[0].IsEditable || keyValues[1].IsEditable == nil || *keyValues[1].IsEditable {
		t.Errorf("isEditable not decoded: %+v", keyValues)
	}
	if len(requests) != 2 {
		t.Errorf("got %d requests, want 2: %v", len(requests), requests)
	}
}

// Create and update send every field, which the platform requires on an update too, and a
// string value type: only a string tag is added to the Workload's searchable tags.
func TestKeyValueWritesSendTheWholeEntry(t *testing.T) {
	req := &KeyValueRequest{
		EntityID: "wl-1", EntityType: "workload", Category: KeyValueCategoryTag,
		Name: "team", Value: "ml", ValueType: KeyValueTypeString,
	}
	want := map[string]any{
		"entityId": "wl-1", "entityType": "workload", "category": "tag",
		"name": "team", "value": "ml", "valueType": "string",
	}
	for name, tc := range map[string]struct {
		method, path string
		status       int
		response     map[string]any
		call         func(Service) error
	}{
		"create": {http.MethodPost, "/keyValues/", http.StatusCreated, map[string]any{"id": "kv-1"}, func(s Service) error {
			resp, err := s.CreateKeyValue(context.Background(), req)
			if err == nil && resp.ID != "kv-1" {
				t.Errorf("id = %q, want kv-1", resp.ID)
			}
			return err
		}},
		"update": {http.MethodPatch, "/keyValues/kv-1/", http.StatusOK, map[string]any{"id": "kv-1", "name": "team", "value": "ml"}, func(s Service) error {
			_, err := s.UpdateKeyValue(context.Background(), "kv-1", req)
			return err
		}},
		"delete": {http.MethodDelete, "/keyValues/kv-1/", http.StatusNoContent, nil, func(s Service) error {
			return s.DeleteKeyValue(context.Background(), "kv-1")
		}},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, tc.method, tc.path)
				}
				if tc.method != http.MethodDelete {
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("decode body: %v", err)
						return
					}
					for key, value := range want {
						if body[key] != value {
							t.Errorf("body[%q] = %#v, want %#v", key, body[key], value)
						}
					}
				}
				if tc.response == nil {
					w.WriteHeader(tc.status)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(tc.response)
			}))
			defer server.Close()

			cfg := NewConfiguration("fake-token")
			cfg.Endpoint = server.URL
			if err := tc.call(NewService(NewClient(cfg))); err != nil {
				t.Fatalf("%s returned error: %v", name, err)
			}
		})
	}
}

// A refused request carries its HTTP status, so callers need not match its text.
func TestHTTPStatusOfRefusedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"message": "The name is already in use for the entity ID"}`))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	_, err := NewService(NewClient(cfg)).CreateKeyValue(context.Background(), &KeyValueRequest{Name: "team"})
	if status := HTTPStatus(err); status != http.StatusConflict {
		t.Errorf("HTTPStatus(%v) = %d, want %d", err, status, http.StatusConflict)
	}
	if status := HTTPStatus(NewNotFoundError("keyValue")); status != 0 {
		t.Errorf("HTTPStatus of an error that reports no response = %d, want 0", status)
	}
}
