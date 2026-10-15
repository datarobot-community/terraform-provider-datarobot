package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The account of a failed Workload quotes Kubernetes' reason for each container that is not
// ready, once however many replicas report it, and the log in time order; a part that cannot be
// read is left out.
func TestDescribeWorkloadFailure(t *testing.T) {
	for name, tc := range map[string]struct {
		statusDetails int
		want, notWant []string
	}{
		"status details and log": {
			statusDetails: http.StatusOK,
			want: []string{
				"Container main is not ready: ErrImagePull: failed to pull image \"whoami:missing\": not found.",
				"Last lines of the Workload's log:\n[2026-10-15 12:40:05+00:00] ERROR: main: Waiting reason=ContainerCreating\n" +
					"[2026-10-15 12:40:07+00:00] ERROR: main: Image pull failed reason=ErrImagePull",
			},
		},
		"no status yet": {
			statusDetails: http.StatusNoContent,
			want:          []string{"Last lines of the Workload's log:"},
			notWant:       []string{"is not ready"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/workloads/wl-1/":
					_, _ = w.Write([]byte(`{"id": "wl-1", "status": "errored", "protonId": "p-1"}`))
				case "/workloads/wl-1/protons/p-1/statusDetails":
					w.WriteHeader(tc.statusDetails)
					if tc.statusDetails == http.StatusOK {
						container := `{"name": "lrs-p-1-main", "ready": false, "restartCount": 0, "reason": "ErrImagePull", ` +
							`"message": "failed to pull image \"whoami:missing\": not found", "exitCode": null, "lastState": null}`
						_, _ = w.Write([]byte(`{"replicas": [{"name": "r1", "containers": [` + container + `]}, {"name": "r2", "containers": [` + container + `]}]}`))
					}
				case "/otel/workload/wl-1/logs/":
					if r.URL.Query().Get("limit") != "20" {
						t.Errorf("log limit = %q, want 20", r.URL.Query().Get("limit"))
					}
					_, _ = w.Write([]byte(`{"data": [` +
						`{"timestamp": "2026-10-15 12:40:07+00:00", "level": "ERROR", "message": "main: Image pull failed reason=ErrImagePull"},` +
						`{"timestamp": "2026-10-15 12:40:05+00:00", "level": "ERROR", "message": "main: Waiting reason=ContainerCreating"}]}`))
				default:
					t.Errorf("unexpected request %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			cfg := NewConfiguration("fake-token")
			cfg.Endpoint = server.URL
			got := NewService(NewClient(cfg)).DescribeWorkloadFailure(context.Background(), "wl-1")
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("account = %q, want it to contain %q", got, want)
				}
			}
			for _, notWant := range tc.notWant {
				if strings.Contains(got, notWant) {
					t.Errorf("account = %q, want no %q", got, notWant)
				}
			}
			if strings.Count(got, "Container main") > 1 {
				t.Errorf("account repeats a container reported by several replicas: %q", got)
			}
		})
	}
}
