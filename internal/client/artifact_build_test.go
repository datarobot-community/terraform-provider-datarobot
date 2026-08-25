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

func TestWaitForArtifactBuildStreamsOtelLogs(t *testing.T) {
	buildCalls := 0
	otelCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/artifacts/art-1/builds/build-1/":
			buildCalls++
			status := ArtifactBuildStatusInProgress
			if buildCalls >= 2 {
				status = ArtifactBuildStatusCompleted
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(artifactBuildJSON(status))
		case "/otel/artifact/art-1/logs/":
			otelCalls++
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{
					"level":     "info",
					"message":   "building image",
					"timestamp": "2026-07-09T16:14:50Z",
				}},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	var lines []OtelLogEntry
	_, err := svc.WaitForArtifactBuild(context.Background(), "art-1", "build-1", &WaitForArtifactBuildOptions{
		PollInterval: 5 * time.Millisecond,
		Timeout:      time.Second,
		OnOtelLogLine: func(entry OtelLogEntry) {
			lines = append(lines, entry)
		},
	})
	if err != nil {
		t.Fatalf("WaitForArtifactBuild returned error: %v", err)
	}
	if otelCalls == 0 {
		t.Fatal("expected OTEL log polling during wait")
	}
	if len(lines) == 0 || lines[0].Message != "building image" {
		t.Fatalf("expected streamed OTEL logs, got %#v", lines)
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

func TestGetArtifactBuildLogsParsesPlainTextBuildKitOutput(t *testing.T) {
	body := "#1 [internal] load build definition from Dockerfile\n#2 ERROR: failed to solve: process \"/bin/sh\" did not complete\n#3 CANCELED\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetArtifactBuildLogs(context.Background(), "art-1", "build-1")
	if err != nil {
		t.Fatalf("GetArtifactBuildLogs returned error: %v", err)
	}

	if !strings.Contains(logs, "ERROR: failed to solve") {
		t.Fatalf("expected plain-text build logs, got:\n%s", logs)
	}
}

func TestGetArtifactBuildLogsFallsBackToOtelArtifactLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/artifacts/art-1/builds/build-1/logs":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"detail":"Failed to retrieve build logs"}`))
		case "/otel/artifact/art-1/logs/":
			if got := r.URL.Query().Get("limit"); got != "30" {
				t.Fatalf("expected limit=30, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{
					"level":     "error",
					"message":   "build failed in otel",
					"timestamp": "2026-07-09T16:14:50Z",
				}},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetArtifactBuildLogs(context.Background(), "art-1", "build-1")
	if err != nil {
		t.Fatalf("GetArtifactBuildLogs returned error: %v", err)
	}

	if !strings.Contains(logs, "build failed in otel") {
		t.Fatalf("expected OTEL fallback logs, got:\n%s", logs)
	}
}

func TestGetArtifactBuildLogsParsesAndTailsJSONL(t *testing.T) {
	body := `{"asctime":"2026-06-09 10:00:00","levelname":"INFO","message":"line-1"}
{"asctime":"2026-06-09 10:00:01","levelname":"ERROR","message":"line-2"}
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/artifacts/art-1/builds/build-1/logs" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetArtifactBuildLogs(context.Background(), "art-1", "build-1")
	if err != nil {
		t.Fatalf("GetArtifactBuildLogs returned error: %v", err)
	}

	for _, want := range []string{
		"[2026-06-09 10:00:00] INFO: line-1",
		"[2026-06-09 10:00:01] ERROR: line-2",
	} {
		if !strings.Contains(logs, want) {
			t.Errorf("expected logs to contain %q, got:\n%s", want, logs)
		}
	}
}

func TestGetArtifactBuildLogsTailLinesEnvVar(t *testing.T) {
	body := `{"asctime":"2026-06-09 10:00:00","levelname":"INFO","message":"line-1"}
{"asctime":"2026-06-09 10:00:01","levelname":"INFO","message":"line-2"}
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	t.Setenv(ArtifactBuildLogsTailLinesEnvVar, "1")

	logs, err := svc.GetArtifactBuildLogs(context.Background(), "art-1", "build-1")
	if err != nil {
		t.Fatalf("GetArtifactBuildLogs returned error: %v", err)
	}

	if !strings.Contains(logs, "line-2") {
		t.Errorf("expected tail to include last line, got:\n%s", logs)
	}
	if strings.Contains(logs, "line-1") {
		t.Errorf("expected tail to exclude first line, got:\n%s", logs)
	}
}

func TestArtifactBuildLogsTailLines(t *testing.T) {
	t.Run("uses env var override", func(t *testing.T) {
		t.Setenv(ArtifactBuildLogsTailLinesEnvVar, "7")
		if got := artifactBuildLogsTailLines(); got != 7 {
			t.Errorf("expected 7, got %d", got)
		}
	})

	t.Run("falls back to default when unset", func(t *testing.T) {
		if got := artifactBuildLogsTailLines(); got != defaultArtifactBuildLogsTailLines {
			t.Errorf("expected default %d, got %d", defaultArtifactBuildLogsTailLines, got)
		}
	})
}
