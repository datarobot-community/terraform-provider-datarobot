package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func executionEnvironmentTestServer(t *testing.T, otel, legacy http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/otel/execution_environment/") {
			if otel == nil {
				t.Fatalf("unexpected OTel request: %s", r.URL.Path)
			}
			otel(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/buildLog/") {
			if legacy == nil {
				t.Fatalf("unexpected legacy build log request: %s", r.URL.Path)
			}
			legacy(w, r)
			return
		}
		t.Fatalf("unexpected path: %s", r.URL.Path)
	}))
}

func TestGetExecutionEnvironmentVersionBuildLogPrefersOtelWhenNonEmpty(t *testing.T) {
	server := executionEnvironmentTestServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("searchKeys"); got != "build_id" {
				t.Fatalf("expected searchKeys=build_id, got %q", got)
			}
			if got := r.URL.Query().Get("searchValues"); got != "build-1" {
				t.Fatalf("expected searchValues=build-1 (the build ID, not the version ID), got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{otelLogEntryJSON("error", "pip install failed")},
				"next": "",
			})
		},
		nil, // legacy must not be called
	)
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetExecutionEnvironmentVersionBuildLog(context.Background(), "env-1", "ver-1", "build-1")
	if err != nil {
		t.Fatalf("GetExecutionEnvironmentVersionBuildLog returned error: %v", err)
	}
	if !strings.Contains(logs, "pip install failed") {
		t.Errorf("expected logs to contain %q, got:\n%s", "pip install failed", logs)
	}
}

func TestGetExecutionEnvironmentVersionBuildLogReversesOtelOldestFirstToNewestFirst(t *testing.T) {
	server := executionEnvironmentTestServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				// The API returns entries oldest-first; "line3" is the most recent.
				// The assertion below checks that our output reverses this to
				// newest-first — that's what's under test, not the API's own order.
				"data": []map[string]any{
					otelLogEntryJSON("info", "line1"),
					otelLogEntryJSON("info", "line2"),
					otelLogEntryJSON("error", "line3"),
				},
				"next": "",
			})
		},
		nil,
	)
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetExecutionEnvironmentVersionBuildLog(context.Background(), "env-1", "ver-1", "build-1")
	if err != nil {
		t.Fatalf("GetExecutionEnvironmentVersionBuildLog returned error: %v", err)
	}

	want := "[2026-07-09T16:14:50Z] ERROR: line3\n[2026-07-09T16:14:50Z] INFO: line2\n[2026-07-09T16:14:50Z] INFO: line1"
	if logs != want {
		t.Errorf("expected output reversed to newest-first:\n%s\ngot:\n%s", want, logs)
	}
}

func TestGetExecutionEnvironmentVersionBuildLogSkipsOtelWhenBuildIDEmpty(t *testing.T) {
	server := executionEnvironmentTestServer(t,
		nil, // OTel must not be called when buildId is empty
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/executionEnvironments/env-1/versions/ver-1/buildLog/" {
				t.Fatalf("unexpected legacy path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"log": "legacy line"})
		},
	)
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetExecutionEnvironmentVersionBuildLog(context.Background(), "env-1", "ver-1", "")
	if err != nil {
		t.Fatalf("GetExecutionEnvironmentVersionBuildLog returned error: %v", err)
	}
	if logs != "legacy line" {
		t.Errorf("expected legacy logs, got:\n%s", logs)
	}
}

func TestGetExecutionEnvironmentVersionBuildLogFallsBackToLegacyWhenOtelEmpty(t *testing.T) {
	server := executionEnvironmentTestServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}, "next": ""})
		},
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/executionEnvironments/env-1/versions/ver-1/buildLog/" {
				t.Fatalf("unexpected legacy path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"log": "legacy line1\nlegacy line2"})
		},
	)
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetExecutionEnvironmentVersionBuildLog(context.Background(), "env-1", "ver-1", "build-1")
	if err != nil {
		t.Fatalf("GetExecutionEnvironmentVersionBuildLog returned error: %v", err)
	}
	if logs != "legacy line1\nlegacy line2" {
		t.Errorf("expected legacy logs, got:\n%s", logs)
	}
}

func TestGetExecutionEnvironmentVersionBuildLogFallsBackToLegacyWhenOtelErrors(t *testing.T) {
	server := executionEnvironmentTestServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"log": "legacy line"})
		},
	)
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetExecutionEnvironmentVersionBuildLog(context.Background(), "env-1", "ver-1", "build-1")
	if err != nil {
		t.Fatalf("GetExecutionEnvironmentVersionBuildLog returned error: %v", err)
	}
	if logs != "legacy line" {
		t.Errorf("expected legacy logs, got:\n%s", logs)
	}
}

func TestGetExecutionEnvironmentVersionBuildLogIncludesLegacyBuildError(t *testing.T) {
	server := executionEnvironmentTestServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}, "next": ""})
		},
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"log": "line1", "error": "pip install failed"})
		},
	)
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetExecutionEnvironmentVersionBuildLog(context.Background(), "env-1", "ver-1", "build-1")
	if err != nil {
		t.Fatalf("GetExecutionEnvironmentVersionBuildLog returned error: %v", err)
	}
	for _, want := range []string{"line1", "ERROR: pip install failed"} {
		if !strings.Contains(logs, want) {
			t.Errorf("expected logs to contain %q, got:\n%s", want, logs)
		}
	}
}

func TestGetExecutionEnvironmentVersionBuildLogEmptyWhenBothSourcesEmpty(t *testing.T) {
	server := executionEnvironmentTestServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}, "next": ""})
		},
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"log": ""})
		},
	)
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetExecutionEnvironmentVersionBuildLog(context.Background(), "env-1", "ver-1", "build-1")
	if err != nil {
		t.Fatalf("GetExecutionEnvironmentVersionBuildLog returned error: %v", err)
	}
	if logs != "" {
		t.Errorf("expected empty logs, got:\n%s", logs)
	}
}

func TestGetExecutionEnvironmentVersionBuildLogReturnsErrorWhenBothSourcesFail(t *testing.T) {
	server := executionEnvironmentTestServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
	)
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	_, err := svc.GetExecutionEnvironmentVersionBuildLog(context.Background(), "env-1", "ver-1", "build-1")
	if err == nil {
		t.Fatal("expected an error when both OTel and legacy sources fail")
	}
}

func TestGetExecutionEnvironmentVersionBuildLogTailLinesEnvVar(t *testing.T) {
	server := executionEnvironmentTestServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("limit"); got != "2" {
				t.Fatalf("expected limit=2 query param, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			entries := []map[string]any{
				otelLogEntryJSON("info", "line1"),
				otelLogEntryJSON("info", "line2"),
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": entries, "next": ""})
		},
		nil,
	)
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	t.Setenv(ExecutionEnvironmentBuildLogTailLinesEnvVar, "2")

	logs, err := svc.GetExecutionEnvironmentVersionBuildLog(context.Background(), "env-1", "ver-1", "build-1")
	if err != nil {
		t.Fatalf("GetExecutionEnvironmentVersionBuildLog returned error: %v", err)
	}
	if got := strings.Count(logs, "\n") + 1; got != 2 {
		t.Errorf("expected 2 log lines with tail=2, got %d:\n%s", got, logs)
	}
}

func TestGetExecutionEnvironmentVersionBuildLogLegacyDefaultTailLines(t *testing.T) {
	lines := make([]string, 0, defaultExecutionEnvironmentBuildLogTailLines+5)
	for i := 0; i < defaultExecutionEnvironmentBuildLogTailLines+5; i++ {
		lines = append(lines, "line")
	}
	fullLog := strings.Join(lines, "\n")

	server := executionEnvironmentTestServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}, "next": ""})
		},
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"log": fullLog})
		},
	)
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetExecutionEnvironmentVersionBuildLog(context.Background(), "env-1", "ver-1", "build-1")
	if err != nil {
		t.Fatalf("GetExecutionEnvironmentVersionBuildLog returned error: %v", err)
	}
	if got := strings.Count(logs, "\n") + 1; got != defaultExecutionEnvironmentBuildLogTailLines {
		t.Errorf("expected %d log lines by default, got %d", defaultExecutionEnvironmentBuildLogTailLines, got)
	}
}

func TestExecutionEnvironmentBuildLogTailLines(t *testing.T) {
	t.Run("uses env var override", func(t *testing.T) {
		t.Setenv(ExecutionEnvironmentBuildLogTailLinesEnvVar, "7")
		if got := executionEnvironmentBuildLogTailLines(); got != 7 {
			t.Errorf("expected 7, got %d", got)
		}
	})

	t.Run("falls back to default on invalid value", func(t *testing.T) {
		t.Setenv(ExecutionEnvironmentBuildLogTailLinesEnvVar, "not-a-number")
		if got := executionEnvironmentBuildLogTailLines(); got != defaultExecutionEnvironmentBuildLogTailLines {
			t.Errorf("expected default %d, got %d", defaultExecutionEnvironmentBuildLogTailLines, got)
		}
	})

	t.Run("falls back to default on non-positive value", func(t *testing.T) {
		t.Setenv(ExecutionEnvironmentBuildLogTailLinesEnvVar, "0")
		if got := executionEnvironmentBuildLogTailLines(); got != defaultExecutionEnvironmentBuildLogTailLines {
			t.Errorf("expected default %d, got %d", defaultExecutionEnvironmentBuildLogTailLines, got)
		}
	})

	t.Run("falls back to default when unset", func(t *testing.T) {
		if got := executionEnvironmentBuildLogTailLines(); got != defaultExecutionEnvironmentBuildLogTailLines {
			t.Errorf("expected default %d, got %d", defaultExecutionEnvironmentBuildLogTailLines, got)
		}
	})
}

func TestTailLastLines(t *testing.T) {
	t.Run("returns input unchanged when empty", func(t *testing.T) {
		if got := tailLastLines("", 5); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("returns input unchanged when fewer lines than n", func(t *testing.T) {
		if got := tailLastLines("a\nb", 5); got != "a\nb" {
			t.Errorf("expected %q, got %q", "a\nb", got)
		}
	})

	t.Run("tails to last n lines", func(t *testing.T) {
		if got := tailLastLines("a\nb\nc\nd", 2); got != "c\nd" {
			t.Errorf("expected %q, got %q", "c\nd", got)
		}
	})
}
