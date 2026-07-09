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

func TestGetDeploymentLogsPaginatesAndFormats(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/otel/deployment/dep-1/logs/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("offset") == "2" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					otelLogEntryJSON("error", "container exited"),
				},
				"next": "",
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				otelLogEntryJSON("info", "starting up"),
				otelLogEntryJSON("error", "failed to load model"),
			},
			"next": server.URL + "/otel/deployment/dep-1/logs/?offset=2",
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
		"[2026-07-09T16:14:50Z] ERROR: container exited",
	} {
		if !strings.Contains(logs, want) {
			t.Errorf("expected logs to contain %q, got:\n%s", want, logs)
		}
	}
}

func TestGetDeploymentLogsTailLinesEnvVar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		entries := make([]map[string]any, 0, 5)
		for i := 0; i < 5; i++ {
			entries = append(entries, otelLogEntryJSON("info", "line"))
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
		w.Header().Set("Content-Type", "application/json")
		entries := make([]map[string]any, 0, 40)
		for i := 0; i < 40; i++ {
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
