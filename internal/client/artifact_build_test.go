package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func artifactBuildJSON(status string) map[string]any {
	return map[string]any{
		"id":         "build-1",
		"artifactId": "art-1",
		"status":     status,
		"createdAt":  "2026-01-01T00:00:00Z",
		"updatedAt":  "2026-01-01T00:01:00Z",
	}
}

func TestIsTerminalArtifactBuildStatus(t *testing.T) {
	terminal := []string{
		ArtifactBuildStatusCompleted,
		ArtifactBuildStatusFailed,
		ArtifactBuildStatusCancelled,
	}
	for _, status := range terminal {
		if !IsTerminalArtifactBuildStatus(status) {
			t.Errorf("expected %q to be terminal", status)
		}
	}

	nonTerminal := []string{
		ArtifactBuildStatusPending,
		ArtifactBuildStatusInProgress,
		ArtifactBuildStatusBuilt,
		"UNKNOWN",
	}
	for _, status := range nonTerminal {
		if IsTerminalArtifactBuildStatus(status) {
			t.Errorf("expected %q to be non-terminal", status)
		}
	}
}

func TestTriggerArtifactBuild(t *testing.T) {
	var method, path string
	var body map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"buildIds": []string{"build-1"}})
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	resp, err := svc.TriggerArtifactBuild(context.Background(), "art-1")
	if err != nil {
		t.Fatalf("TriggerArtifactBuild returned error: %v", err)
	}
	if method != http.MethodPost {
		t.Fatalf("expected POST, got %s", method)
	}
	if path != "/artifacts/art-1/builds/" {
		t.Fatalf("unexpected path: %s", path)
	}
	if len(body) != 0 {
		t.Fatalf("expected empty body, got %#v", body)
	}
	if len(resp.BuildIDs) != 1 || resp.BuildIDs[0] != "build-1" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestGetArtifactBuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/artifacts/art-1/builds/build-1/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(artifactBuildJSON(ArtifactBuildStatusCompleted))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	build, err := svc.GetArtifactBuild(context.Background(), "art-1", "build-1")
	if err != nil {
		t.Fatalf("GetArtifactBuild returned error: %v", err)
	}
	if build.Status != ArtifactBuildStatusCompleted {
		t.Fatalf("expected completed status, got %s", build.Status)
	}
}

func TestWaitForArtifactBuildCompletesOnTerminalStatus(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		status := ArtifactBuildStatusInProgress
		if calls >= 2 {
			status = ArtifactBuildStatusCompleted
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(artifactBuildJSON(status))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	build, err := svc.WaitForArtifactBuild(context.Background(), "art-1", "build-1", &WaitForArtifactBuildOptions{
		PollInterval: 5 * time.Millisecond,
		Timeout:      time.Second,
	})
	if err != nil {
		t.Fatalf("WaitForArtifactBuild returned error: %v", err)
	}
	if build.Status != ArtifactBuildStatusCompleted {
		t.Fatalf("expected completed status, got %s", build.Status)
	}
	if calls < 2 {
		t.Fatalf("expected at least 2 polls, got %d", calls)
	}
}

func TestWaitForArtifactBuildReturnsFailedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(artifactBuildJSON(ArtifactBuildStatusFailed))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	_, err := svc.WaitForArtifactBuild(context.Background(), "art-1", "build-1", &WaitForArtifactBuildOptions{
		PollInterval: 5 * time.Millisecond,
		Timeout:      time.Second,
	})
	if err == nil {
		t.Fatal("expected error for failed build")
	}

	var failedErr *ArtifactBuildFailedError
	if !errors.As(err, &failedErr) {
		t.Fatalf("expected ArtifactBuildFailedError, got %T: %v", err, err)
	}
	if failedErr.Status != ArtifactBuildStatusFailed {
		t.Fatalf("unexpected status: %q", failedErr.Status)
	}
}

func TestWaitForArtifactBuildTimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(artifactBuildJSON(ArtifactBuildStatusBuilt))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	_, err := svc.WaitForArtifactBuild(context.Background(), "art-1", "build-1", &WaitForArtifactBuildOptions{
		PollInterval: 5 * time.Millisecond,
		Timeout:      20 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	var timeoutErr *ArtifactBuildTimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("expected ArtifactBuildTimeoutError, got %T: %v", err, err)
	}
	if timeoutErr.ArtifactID != "art-1" || timeoutErr.BuildID != "build-1" {
		t.Fatalf("unexpected timeout error fields: %+v", timeoutErr)
	}
}

func TestArtifactBuildPollSettings(t *testing.T) {
	t.Run("uses env var override for interval", func(t *testing.T) {
		t.Setenv(ArtifactBuildPollIntervalEnvVar, "7s")
		if got := artifactBuildPollInterval(); got != 7*time.Second {
			t.Fatalf("expected 7s, got %s", got)
		}
	})

	t.Run("uses env var override for timeout", func(t *testing.T) {
		t.Setenv(ArtifactBuildPollTimeoutEnvVar, "45m")
		if got := artifactBuildPollTimeout(); got != 45*time.Minute {
			t.Fatalf("expected 45m, got %s", got)
		}
	})

	t.Run("falls back to defaults for invalid values", func(t *testing.T) {
		t.Setenv(ArtifactBuildPollIntervalEnvVar, "not-a-duration")
		if got := artifactBuildPollInterval(); got != defaultArtifactBuildPollInterval {
			t.Errorf("expected default %s, got %s", defaultArtifactBuildPollInterval, got)
		}
	})
}
