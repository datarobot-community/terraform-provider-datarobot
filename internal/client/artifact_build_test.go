package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

// fastArtifactBuildLogsSettle shrinks GetArtifactBuildLogs' settle wait so a test
// reading the tail takes milliseconds, not the production 10s per read.
func fastArtifactBuildLogsSettle(t *testing.T, timeout time.Duration) {
	t.Helper()
	interval, oldTimeout := artifactBuildLogsSettleInterval, artifactBuildLogsSettleTimeout
	artifactBuildLogsSettleInterval, artifactBuildLogsSettleTimeout = time.Millisecond, timeout
	t.Cleanup(func() {
		artifactBuildLogsSettleInterval, artifactBuildLogsSettleTimeout = interval, oldTimeout
	})
}

func otelLogsJSON(w http.ResponseWriter, entries ...map[string]any) {
	if entries == nil {
		entries = []map[string]any{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": entries})
}

func otelLogRecord(timestamp, message string) map[string]any {
	return map[string]any{"level": "info", "message": message, "timestamp": timestamp}
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

func TestWaitForArtifactBuildCallsOnPoll(t *testing.T) {
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

	var pollStatuses []string
	_, err := svc.WaitForArtifactBuild(context.Background(), "art-1", "build-1", &WaitForArtifactBuildOptions{
		PollInterval: 5 * time.Millisecond,
		Timeout:      time.Second,
		OnPoll: func(build *ArtifactBuild) {
			pollStatuses = append(pollStatuses, build.Status)
		},
	})
	if err != nil {
		t.Fatalf("WaitForArtifactBuild returned error: %v", err)
	}
	if len(pollStatuses) != 1 || pollStatuses[0] != ArtifactBuildStatusInProgress {
		t.Fatalf("OnPoll statuses = %#v, want one IN_PROGRESS callback before completion", pollStatuses)
	}
}

func TestWaitForArtifactBuildReturnsFailedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		build := artifactBuildJSON(ArtifactBuildStatusFailed)
		build["failureReason"] = "The build failed; it was last seen executing the Dockerfile."
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(build)
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
	want := "artifact build build-1 ended with status FAILED: The build failed; it was last seen executing the Dockerfile."
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
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
			if got := r.URL.Query()["searchKeys"]; len(got) != 1 || got[0] != "external_build_id" {
				t.Errorf("expected searchKeys=external_build_id, got %v", got)
			}
			if got := r.URL.Query()["searchValues"]; len(got) != 1 || got[0] != "build-1" {
				t.Errorf("expected searchValues=build-1, got %v", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{
					"level":     "info",
					"message":   "building image",
					"timestamp": "2026-07-09T16:14:50Z",
				}},
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
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

func TestGetArtifactBuildLogsReadsOtelTailOldestFirst(t *testing.T) {
	fastArtifactBuildLogsSettle(t, time.Second)
	t.Setenv(ArtifactBuildLogsTailLinesEnvVar, "7")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/otel/artifact/art-1/logs/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		q := r.URL.Query()
		if got := q.Get("limit"); got != "7" {
			t.Errorf("expected limit=7 from %s, got %q", ArtifactBuildLogsTailLinesEnvVar, got)
		}
		if got := q["searchKeys"]; len(got) != 1 || got[0] != "external_build_id" {
			t.Errorf("expected searchKeys=external_build_id, got %v", got)
		}
		if got := q["searchValues"]; len(got) != 1 || got[0] != "build-1" {
			t.Errorf("expected searchValues=build-1, got %v", got)
		}
		w.Header().Set("Content-Type", "application/json")
		// Newest first, as the server sends them, and like it ordered only to the
		// millisecond: the last two share one, in the wrong order.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"level": "info", "message": "Image build FAILED in 4.3 seconds.", "timestamp": "2026-10-07 09:26:14.974976+00:00"},
				{"level": "info", "message": "#12 [5/5] RUN exit 7", "timestamp": "2026-10-07 09:26:14.268356+00:00"},
				{"level": "info", "message": "#12 ERROR: exit code: 7", "timestamp": "2026-10-07 09:26:14.268449+00:00"},
			},
		})
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetArtifactBuildLogs(context.Background(), "art-1", "build-1")
	if err != nil {
		t.Fatalf("GetArtifactBuildLogs returned error: %v", err)
	}

	want := "[2026-10-07 09:26:14.268356+00:00] INFO: #12 [5/5] RUN exit 7\n" +
		"[2026-10-07 09:26:14.268449+00:00] INFO: #12 ERROR: exit code: 7\n" +
		"[2026-10-07 09:26:14.974976+00:00] INFO: Image build FAILED in 4.3 seconds."
	if logs != want {
		t.Fatalf("logs =\n%s\nwant\n%s", logs, want)
	}
}

func TestGetArtifactBuildLogsEmptyOtelIsNotAnError(t *testing.T) {
	fastArtifactBuildLogsSettle(t, 20*time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		otelLogsJSON(w)
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetArtifactBuildLogs(context.Background(), "art-1", "build-1")
	if err != nil || logs != "" {
		t.Fatalf("GetArtifactBuildLogs = %q, %v; want empty logs and no error", logs, err)
	}
}

// The removed per-build log endpoint must never be read, nor named in the error.
func TestGetArtifactBuildLogsReportsOtelError(t *testing.T) {
	fastArtifactBuildLogsSettle(t, time.Second)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/otel/artifact/art-1/logs/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	_, err := svc.GetArtifactBuildLogs(context.Background(), "art-1", "build-1")
	if err == nil {
		t.Fatal("expected an error when the OTEL read fails")
	}
	if strings.Contains(err.Error(), "/builds/build-1/logs") {
		t.Fatalf("error names the removed per-build log endpoint: %v", err)
	}
}

// A build reported FAILED can have no records yet, then only the lines before its error:
// the tail is the one two consecutive reads agree on.
func TestGetArtifactBuildLogsWaitsForIngestionToSettle(t *testing.T) {
	fastArtifactBuildLogsSettle(t, time.Second)

	step := otelLogRecord("2026-10-07 09:26:14.268356+00:00", "#12 [5/5] RUN exit 7")
	failure := otelLogRecord("2026-10-07 09:26:14.268449+00:00", "#12 ERROR: exit code: 7")
	reads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads++
		switch reads {
		case 1:
			otelLogsJSON(w)
		case 2:
			otelLogsJSON(w, step)
		default:
			otelLogsJSON(w, failure, step)
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
	if !strings.Contains(logs, "exit code: 7") {
		t.Fatalf("expected the settled tail with the error line, got:\n%s", logs)
	}
	if reads != 4 {
		t.Fatalf("expected 4 reads (empty, partial, full, full again), got %d", reads)
	}
}

func TestGetArtifactBuildLogsStopsWaitingAtTimeout(t *testing.T) {
	fastArtifactBuildLogsSettle(t, 30*time.Millisecond)

	reads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads++
		// A build still running: every read has a newer line.
		otelLogsJSON(w, otelLogRecord(fmt.Sprintf("2026-10-07 09:26:%02d", reads%60), fmt.Sprintf("line %d", reads)))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	start := time.Now()
	logs, err := svc.GetArtifactBuildLogs(context.Background(), "art-1", "build-1")
	if err != nil {
		t.Fatalf("GetArtifactBuildLogs returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("expected to give up near the 30ms timeout, took %s", elapsed)
	}
	if !strings.Contains(logs, "line ") {
		t.Fatalf("expected the last read's lines, got %q", logs)
	}
}

// A failed re-read must not throw away lines an earlier read already returned.
func TestGetArtifactBuildLogsKeepsEarlierReadOnError(t *testing.T) {
	fastArtifactBuildLogsSettle(t, time.Second)

	reads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads++
		if reads == 1 {
			otelLogsJSON(w, otelLogRecord("2026-10-07 09:26:14.268356+00:00", "#12 [5/5] RUN exit 7"))
			return
		}
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetArtifactBuildLogs(context.Background(), "art-1", "build-1")
	if err != nil {
		t.Fatalf("GetArtifactBuildLogs returned error: %v", err)
	}
	if !strings.Contains(logs, "RUN exit 7") {
		t.Fatalf("expected the first read's lines, got %q", logs)
	}
}

// A context that ends during the wait returns what was read so far. The production 10s
// interval is far longer than the context, so the wait is cut short by it.
func TestGetArtifactBuildLogsStopsWaitingWhenContextEnds(t *testing.T) {
	reads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads++
		otelLogsJSON(w, otelLogRecord("2026-10-07 09:26:14.268356+00:00", "#12 [5/5] RUN exit 7"))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	logs, err := svc.GetArtifactBuildLogs(ctx, "art-1", "build-1")
	if err != nil {
		t.Fatalf("GetArtifactBuildLogs returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("expected to stop when the context ended, took %s", elapsed)
	}
	if reads != 1 || !strings.Contains(logs, "RUN exit 7") {
		t.Fatalf("expected the first read's lines after one read, got %d reads: %q", reads, logs)
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
