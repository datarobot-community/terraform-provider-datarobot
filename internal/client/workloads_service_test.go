package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func replacementJSON(status ReplacementStatus, message string) map[string]any {
	payload := map[string]any{
		"id":                  "repl-1",
		"workloadId":          "wl-1",
		"candidateArtifactId": "art-2",
		"status":              status,
		"strategy":            ReplacementStrategyRolling,
		"config": map[string]any{
			"warmupDurationMinutes": 5,
		},
	}
	if message != "" {
		payload["message"] = message
	}
	return payload
}

func TestStartWorkloadReplacementPostsExpectedPayload(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(bodyBytes, &gotBody); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(replacementJSON(ReplacementStatusSubmitted, ""))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	replicaCount := int64(2)
	resp, err := svc.StartWorkloadReplacement(context.Background(), "wl-1", &StartReplacementRequest{
		ArtifactID: "art-2",
		Strategy:   ReplacementStrategyRolling,
		Config: ReplacementConfig{
			WarmupDurationMinutes: 5,
		},
		Runtime: &WorkloadRuntime{
			ContainerGroups: []GroupRuntime{
				{Name: "default", ReplicaCount: &replicaCount},
			},
		},
	})
	if err != nil {
		t.Fatalf("StartWorkloadReplacement returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/workloads/wl-1/replacement" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotBody["artifactId"] != "art-2" {
		t.Fatalf("expected artifactId art-2, got %#v", gotBody["artifactId"])
	}
	if gotBody["strategy"] != string(ReplacementStrategyRolling) {
		t.Fatalf("expected rolling strategy, got %#v", gotBody["strategy"])
	}
	if resp == nil || resp.Status != ReplacementStatusSubmitted {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestGetWorkloadReplacementUsesExpectedPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/workloads/wl-1/replacement" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(replacementJSON(ReplacementStatusPromoting, ""))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	resp, err := svc.GetWorkloadReplacement(context.Background(), "wl-1")
	if err != nil {
		t.Fatalf("GetWorkloadReplacement returned error: %v", err)
	}
	if resp.Status != ReplacementStatusPromoting {
		t.Fatalf("expected promoting status, got %s", resp.Status)
	}
}

func TestUpdateWorkloadSettingsPatchesExpectedPayload(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(bodyBytes, &gotBody); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(replacementJSON(ReplacementStatusSubmitted, ""))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	replicaCount := int64(4)
	resp, err := svc.UpdateWorkloadSettings(context.Background(), "wl-1", &UpdateWorkloadSettingsRequest{
		Runtime: WorkloadRuntime{
			ContainerGroups: []GroupRuntime{
				{Name: "default", ReplicaCount: &replicaCount},
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateWorkloadSettings returned error: %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Fatalf("expected PATCH, got %s", gotMethod)
	}
	if gotPath != "/workloads/wl-1/settings" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	runtime, ok := gotBody["runtime"].(map[string]any)
	if !ok {
		t.Fatalf("expected runtime object in body, got %#v", gotBody)
	}
	groups, ok := runtime["containerGroups"].([]any)
	if !ok || len(groups) != 1 {
		t.Fatalf("expected one container group, got %#v", runtime["containerGroups"])
	}
	if resp == nil || resp.WorkloadID != "wl-1" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestWaitForWorkloadReplacementCompletesOnTerminalStatus(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		status := ReplacementStatusPromoting
		if calls >= 2 {
			status = ReplacementStatusCompleted
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(replacementJSON(status, ""))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	resp, err := svc.WaitForWorkloadReplacement(context.Background(), "wl-1", &WaitForWorkloadReplacementOptions{
		PollInterval: 5 * time.Millisecond,
		Timeout:      time.Second,
	})
	if err != nil {
		t.Fatalf("WaitForWorkloadReplacement returned error: %v", err)
	}
	if resp.Status != ReplacementStatusCompleted {
		t.Fatalf("expected completed status, got %s", resp.Status)
	}
	if calls < 2 {
		t.Fatalf("expected at least 2 polls, got %d", calls)
	}
}

func TestWaitForWorkloadReplacementReturnsReplacementFailedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(replacementJSON(ReplacementStatusErrored, "candidate proton failed health checks"))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	_, err := svc.WaitForWorkloadReplacement(context.Background(), "wl-1", &WaitForWorkloadReplacementOptions{
		PollInterval: 5 * time.Millisecond,
		Timeout:      time.Second,
	})
	if err == nil {
		t.Fatal("expected error for errored replacement")
	}

	var failedErr *ReplacementFailedError
	if !errors.As(err, &failedErr) {
		t.Fatalf("expected ReplacementFailedError, got %T: %v", err, err)
	}
	if !strings.Contains(failedErr.Message, "candidate proton failed health checks") {
		t.Fatalf("unexpected error message: %q", failedErr.Message)
	}
}

func TestWaitForWorkloadReplacementTimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(replacementJSON(ReplacementStatusFinalizing, ""))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	_, err := svc.WaitForWorkloadReplacement(context.Background(), "wl-1", &WaitForWorkloadReplacementOptions{
		PollInterval: 5 * time.Millisecond,
		Timeout:      20 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout waiting for workload wl-1 replacement") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkloadReplacementPollSettings(t *testing.T) {
	t.Run("uses env var override for interval", func(t *testing.T) {
		t.Setenv(WorkloadReplacementPollIntervalEnvVar, "7s")
		if got := workloadReplacementPollInterval(); got != 7*time.Second {
			t.Errorf("expected 7s, got %s", got)
		}
	})

	t.Run("uses env var override for timeout", func(t *testing.T) {
		t.Setenv(WorkloadReplacementPollTimeoutEnvVar, "45m")
		if got := workloadReplacementPollTimeout(); got != 45*time.Minute {
			t.Errorf("expected 45m, got %s", got)
		}
	})

	t.Run("falls back to default on invalid interval", func(t *testing.T) {
		t.Setenv(WorkloadReplacementPollIntervalEnvVar, "not-a-duration")
		if got := workloadReplacementPollInterval(); got != defaultReplacementPollInterval {
			t.Errorf("expected default %s, got %s", defaultReplacementPollInterval, got)
		}
	})

	t.Run("falls back to default on non-positive interval", func(t *testing.T) {
		t.Setenv(WorkloadReplacementPollIntervalEnvVar, "0s")
		if got := workloadReplacementPollInterval(); got != defaultReplacementPollInterval {
			t.Errorf("expected default %s, got %s", defaultReplacementPollInterval, got)
		}
	})

	t.Run("falls back to default on non-positive timeout", func(t *testing.T) {
		t.Setenv(WorkloadReplacementPollTimeoutEnvVar, "0s")
		if got := workloadReplacementPollTimeout(); got != defaultReplacementPollTimeout {
			t.Errorf("expected default %s, got %s", defaultReplacementPollTimeout, got)
		}
	})

	t.Run("falls back to default when unset", func(t *testing.T) {
		if got := workloadReplacementPollInterval(); got != defaultReplacementPollInterval {
			t.Errorf("expected default %s, got %s", defaultReplacementPollInterval, got)
		}
		if got := workloadReplacementPollTimeout(); got != defaultReplacementPollTimeout {
			t.Errorf("expected default %s, got %s", defaultReplacementPollTimeout, got)
		}
	})
}

func TestReplacementStatusHelpers(t *testing.T) {
	if !IsReplacementTerminal(ReplacementStatusCompleted) {
		t.Fatal("completed should be terminal")
	}
	if !IsReplacementTerminal(ReplacementStatusErrored) {
		t.Fatal("errored should be terminal")
	}
	if IsReplacementActive(ReplacementStatusCompleted) {
		t.Fatal("completed should not be active")
	}
	if !IsReplacementActive(ReplacementStatusFinalizing) {
		t.Fatal("finalizing should be active")
	}
}

func TestArtifactContainerOmitsEmptyImageURI(t *testing.T) {
	t.Parallel()

	container := ArtifactContainer{
		Primary: func() *bool { v := true; return &v }(),
		Port:    func() *int64 { v := int64(8080); return &v }(),
		ImageBuildConfig: &ArtifactImageBuildConfig{
			Dockerfile: &ArtifactDockerfileConfig{Source: "provided", Path: "./Dockerfile"},
		},
	}

	raw, err := json.Marshal(container)
	if err != nil {
		t.Fatalf("marshal container: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal container: %v", err)
	}
	if _, ok := payload["imageUri"]; ok {
		t.Fatalf("expected imageUri to be omitted, got payload: %s", string(raw))
	}

	container.ImageURI = "nginx:latest"
	raw, err = json.Marshal(container)
	if err != nil {
		t.Fatalf("marshal container with imageUri: %v", err)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal container with imageUri: %v", err)
	}
	if payload["imageUri"] != "nginx:latest" {
		t.Fatalf("imageUri = %v, want %q", payload["imageUri"], "nginx:latest")
	}
}
