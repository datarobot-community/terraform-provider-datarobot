package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func otelLogEntryJSON(level, message string) map[string]any {
	return map[string]any{
		"level":     level,
		"message":   message,
		"timestamp": "2026-07-09T16:14:50Z",
	}
}

func TestGetDeploymentLogsRequestsLimitAndFormats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/otel/deployment/dep-1/logs/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "30" {
			t.Fatalf("expected limit=30 query param, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				otelLogEntryJSON("info", "starting up"),
				otelLogEntryJSON("error", "failed to load model"),
			},
			"next": "",
		})
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetDeploymentLogs(context.Background(), "dep-1")
	if err != nil {
		t.Fatalf("GetDeploymentLogs returned error: %v", err)
	}

	for _, want := range []string{
		"[2026-07-09T16:14:50Z] INFO: starting up",
		"[2026-07-09T16:14:50Z] ERROR: failed to load model",
	} {
		if !strings.Contains(logs, want) {
			t.Errorf("expected logs to contain %q, got:\n%s", want, logs)
		}
	}
}

func TestGetDeploymentLogsTailLinesEnvVar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "2" {
			t.Fatalf("expected limit=2 query param, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		entries := []map[string]any{
			otelLogEntryJSON("info", "line1"),
			otelLogEntryJSON("info", "line2"),
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": entries, "next": ""})
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	t.Setenv(DeploymentLogsTailLinesEnvVar, "2")

	logs, err := svc.GetDeploymentLogs(context.Background(), "dep-1")
	if err != nil {
		t.Fatalf("GetDeploymentLogs returned error: %v", err)
	}

	if got := strings.Count(logs, "\n") + 1; got != 2 {
		t.Errorf("expected 2 log lines with tail=2, got %d:\n%s", got, logs)
	}
}

func TestGetDeploymentLogsDefaultTailLines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "30" {
			t.Fatalf("expected default limit=30 query param, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		entries := make([]map[string]any, 0, defaultDeploymentLogsTailLines)
		for i := 0; i < defaultDeploymentLogsTailLines; i++ {
			entries = append(entries, otelLogEntryJSON("info", "line"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": entries, "next": ""})
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	logs, err := svc.GetDeploymentLogs(context.Background(), "dep-1")
	if err != nil {
		t.Fatalf("GetDeploymentLogs returned error: %v", err)
	}

	if got := strings.Count(logs, "\n") + 1; got != defaultDeploymentLogsTailLines {
		t.Errorf("expected %d log lines by default, got %d", defaultDeploymentLogsTailLines, got)
	}
}

func TestDeploymentLogsTailLines(t *testing.T) {
	t.Run("uses env var override", func(t *testing.T) {
		t.Setenv(DeploymentLogsTailLinesEnvVar, "7")
		if got := deploymentLogsTailLines(); got != 7 {
			t.Errorf("expected 7, got %d", got)
		}
	})

	t.Run("falls back to default on invalid value", func(t *testing.T) {
		t.Setenv(DeploymentLogsTailLinesEnvVar, "not-a-number")
		if got := deploymentLogsTailLines(); got != defaultDeploymentLogsTailLines {
			t.Errorf("expected default %d, got %d", defaultDeploymentLogsTailLines, got)
		}
	})

	t.Run("falls back to default on non-positive value", func(t *testing.T) {
		t.Setenv(DeploymentLogsTailLinesEnvVar, "0")
		if got := deploymentLogsTailLines(); got != defaultDeploymentLogsTailLines {
			t.Errorf("expected default %d, got %d", defaultDeploymentLogsTailLines, got)
		}
	})

	t.Run("falls back to default when unset", func(t *testing.T) {
		if got := deploymentLogsTailLines(); got != defaultDeploymentLogsTailLines {
			t.Errorf("expected default %d, got %d", defaultDeploymentLogsTailLines, got)
		}
	})
}
