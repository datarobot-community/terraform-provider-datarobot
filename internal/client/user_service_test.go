package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newEntitlementsTestService(t *testing.T, handler http.HandlerFunc) (Service, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	return NewService(NewClient(cfg)), server.Close
}

func TestIsFeatureFlagEnabledEvaluatesEntitlements(t *testing.T) {
	svc, closeServer := newEntitlementsTestService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/entitlements/evaluate/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		var req EvaluateEntitlementsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if len(req.Entitlements) != 1 || req.Entitlements[0].Name != "ENABLE_AGENTIC_MEMORY_API" {
			t.Fatalf("unexpected request entitlements: %+v", req.Entitlements)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(EvaluateEntitlementsResponse{
			Entitlements: []EntitlementEvaluation{
				{Name: "ENABLE_AGENTIC_MEMORY_API", Value: true},
			},
		})
	})
	defer closeServer()

	enabled, err := svc.IsFeatureFlagEnabled(context.Background(), "ENABLE_AGENTIC_MEMORY_API")
	if err != nil {
		t.Fatalf("IsFeatureFlagEnabled returned error: %v", err)
	}
	if !enabled {
		t.Error("expected flag to be enabled")
	}
}

func TestIsFeatureFlagEnabledDisabledFlag(t *testing.T) {
	svc, closeServer := newEntitlementsTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(EvaluateEntitlementsResponse{
			Entitlements: []EntitlementEvaluation{
				{Name: "SOME_FLAG", Value: false},
			},
		})
	})
	defer closeServer()

	enabled, err := svc.IsFeatureFlagEnabled(context.Background(), "SOME_FLAG")
	if err != nil {
		t.Fatalf("IsFeatureFlagEnabled returned error: %v", err)
	}
	if enabled {
		t.Error("expected flag to be disabled")
	}
}

func TestIsFeatureFlagEnabledMissingFromResponse(t *testing.T) {
	svc, closeServer := newEntitlementsTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(EvaluateEntitlementsResponse{})
	})
	defer closeServer()

	if _, err := svc.IsFeatureFlagEnabled(context.Background(), "SOME_FLAG"); err == nil {
		t.Error("expected error when flag is missing from the response")
	}
}
