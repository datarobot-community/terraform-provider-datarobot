package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	testWorkloadID     = "674a1b2c3d4e5f6789012345"
	testUserID         = "674a1b2c3d4e5f6789012000"
	testProtonID       = "674a1b2c3d4e5f6789013000"
	testArtifactID     = "674a1b2c3d4e5f6789014000"
	testArtifactRepoID = "674a1b2c3d4e5f6789015000"
	testArtifactName   = "happy-path-artifact"
	testUserFullName   = "Jane Doe"
	testUserEmail      = "jane.doe@example.com"
	testUserhash       = "abc123deadbeefcafebabe0123456789abcdef0123456789abcdef01234567"
)

func assertUserData(t *testing.T, label string, got *UserData) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is nil", label)
	}
	if got.ID != testUserID {
		t.Errorf("%s.id = %q, want %q", label, got.ID, testUserID)
	}
	if got.FullName == nil || *got.FullName != testUserFullName {
		t.Errorf("%s.fullName = %v, want %q", label, got.FullName, testUserFullName)
	}
	if got.Email == nil || *got.Email != testUserEmail {
		t.Errorf("%s.email = %v, want %q", label, got.Email, testUserEmail)
	}
	if got.Username == nil || *got.Username != testUserEmail {
		t.Errorf("%s.username = %v, want %q", label, got.Username, testUserEmail)
	}
	if got.Userhash == nil || *got.Userhash != testUserhash {
		t.Errorf("%s.userhash = %v, want %q", label, got.Userhash, testUserhash)
	}
}

func assertWorkloadArtifactInfo(t *testing.T, label string, got *WorkloadArtifactInfo) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is nil", label)
	}
	if got.ID != testArtifactID {
		t.Errorf("%s.id = %q, want %q", label, got.ID, testArtifactID)
	}
	if got.Name == nil || *got.Name != testArtifactName {
		t.Errorf("%s.name = %v, want %q", label, got.Name, testArtifactName)
	}
	if got.Type == nil || *got.Type != ArtifactTypeService {
		t.Errorf("%s.type = %v, want %q", label, got.Type, ArtifactTypeService)
	}
	if got.Status == nil || *got.Status != ArtifactStatusDraft {
		t.Errorf("%s.status = %v, want %q", label, got.Status, ArtifactStatusDraft)
	}
	if got.Version != nil {
		t.Errorf("%s.version = %v, want nil", label, got.Version)
	}
	if got.ArtifactRepositoryID == nil || *got.ArtifactRepositoryID != testArtifactRepoID {
		t.Errorf("%s.artifactRepositoryId = %v, want %q", label, got.ArtifactRepositoryID, testArtifactRepoID)
	}
	if got.TemplateID != nil {
		t.Errorf("%s.templateId = %v, want nil", label, got.TemplateID)
	}
}

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

func TestWorkloadUnmarshalJSON(t *testing.T) {
	t.Parallel()

	// GET /workloads/{id}/ response shape (single WorkloadFormatted object).
	payload := fmt.Sprintf(`{
		"id": %q,
		"name": "happy-path-workload",
		"createdAt": "2026-05-13T11:42:21.995000+00:00",
		"updatedAt": "2026-06-18T16:00:12.391000+00:00",
		"creator": {
			"id": %q,
			"fullName": %q,
			"email": %q,
			"username": %q,
			"userhash": %q
		},
		"description": null,
		"protonId": %q,
		"artifactId": %q,
		"artifact": {
			"id": %q,
			"name": "happy-path-artifact",
			"type": "service",
			"status": "draft",
			"version": null,
			"artifactRepositoryId": %q,
			"templateId": null
		},
		"type": "service",
		"status": "stopped",
		"replacement": null,
		"runtime": {
			"containerGroups": [
				{
					"name": "default",
					"resourceBundles": [
						"cpu.small"
					],
					"bundleSelectionPolicy": "availability",
					"replicaCount": 1,
					"autoscaling": null,
					"containers": [
						{
							"name": "Primary container",
							"resourceAllocation": {
								"gpu": null,
								"cpu": 1.0,
								"memory": 536870912
							}
						}
					],
					"resolvedBundle": {
						"id": "cpu.small",
						"cpuCount": 1.0,
						"memoryBytes": 536870912,
						"gpuCount": 0,
						"gpuMaker": null,
						"gpuTypeLabel": null
					}
				}
			]
		},
		"permissions": [
			"CAN_DELETE",
			"CAN_VIEW",
			"CAN_MAKE_PREDICTIONS",
			"CAN_SHARE",
			"CAN_UPDATE"
		],
		"importance": "low",
		"requestStats": {
			"totalRequests": 0,
			"concurrentRequests": 0,
			"lastRequestAt": null,
			"responseTime": 0,
			"errorRate": 0.0,
			"requestRates": [0, 0, 0, 0, 0, 0, 0],
			"errorRates": [0, 0, 0, 0, 0, 0, 0]
		},
		"tags": [],
		"endpoint": "https://workloads.example.com/api/v2/endpoints/workloads/%s/",
		"lastResponse": null,
		"owners": [
			{
				"id": %q,
				"fullName": %q,
				"email": %q,
				"username": %q,
				"userhash": %q
			}
		]
	}`,
		testWorkloadID,
		testUserID, testUserFullName, testUserEmail, testUserEmail, testUserhash,
		testProtonID,
		testArtifactID,
		testArtifactID,
		testArtifactRepoID,
		testWorkloadID,
		testUserID, testUserFullName, testUserEmail, testUserEmail, testUserhash,
	)

	var workload Workload
	if err := json.Unmarshal([]byte(payload), &workload); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if workload.ID != testWorkloadID {
		t.Errorf("id = %q, want %q", workload.ID, testWorkloadID)
	}
	if workload.Name != "happy-path-workload" {
		t.Errorf("name = %q, want %q", workload.Name, "happy-path-workload")
	}
	if workload.CreatedAt != "2026-05-13T11:42:21.995000+00:00" {
		t.Errorf("createdAt = %q, unexpected value", workload.CreatedAt)
	}
	if workload.UpdatedAt != "2026-06-18T16:00:12.391000+00:00" {
		t.Errorf("updatedAt = %q, unexpected value", workload.UpdatedAt)
	}
	if workload.Description != nil {
		t.Errorf("description = %v, want nil", workload.Description)
	}
	assertUserData(t, "creator", workload.Creator)
	if workload.ProtonID == nil || *workload.ProtonID != testProtonID {
		t.Errorf("protonId = %v, want %s", workload.ProtonID, testProtonID)
	}
	if workload.ArtifactID == nil || *workload.ArtifactID != testArtifactID {
		t.Errorf("artifactId = %v, want %s", workload.ArtifactID, testArtifactID)
	}
	assertWorkloadArtifactInfo(t, "artifact", workload.Artifact)
	if workload.Type != ArtifactTypeService {
		t.Errorf("type = %q, want service", workload.Type)
	}
	if workload.Status != ProtonStatusStopped {
		t.Errorf("status = %q, want stopped", workload.Status)
	}
	if workload.Replacement != nil {
		t.Errorf("replacement = %v, want nil", workload.Replacement)
	}
	if workload.Importance != WorkloadImportanceLow {
		t.Errorf("importance = %q, want low", workload.Importance)
	}
	wantEndpoint := fmt.Sprintf("https://workloads.example.com/api/v2/endpoints/workloads/%s/", testWorkloadID)
	if workload.Endpoint == nil || *workload.Endpoint != wantEndpoint {
		t.Errorf("endpoint = %v, want %s", workload.Endpoint, wantEndpoint)
	}
	if workload.LastResponse != nil {
		t.Errorf("lastResponse = %v, want nil", workload.LastResponse)
	}
	if len(workload.Tags) != 0 {
		t.Errorf("tags = %v, want empty slice", workload.Tags)
	}
	if len(workload.Permissions) != 5 {
		t.Fatalf("permissions len = %d, want 5", len(workload.Permissions))
	}
	wantPermissions := []string{"CAN_DELETE", "CAN_VIEW", "CAN_MAKE_PREDICTIONS", "CAN_SHARE", "CAN_UPDATE"}
	for i, want := range wantPermissions {
		if workload.Permissions[i] != want {
			t.Errorf("permissions[%d] = %q, want %q", i, workload.Permissions[i], want)
		}
	}

	if workload.RequestStats == nil {
		t.Fatal("requestStats not parsed")
	}
	stats := workload.RequestStats
	if stats.TotalRequests != 0 || stats.ConcurrentRequests != 0 || stats.ResponseTime != 0 || stats.ErrorRate != 0.0 {
		t.Errorf("requestStats counters = (%d, %d, %d, %v), want all zero", stats.TotalRequests, stats.ConcurrentRequests, stats.ResponseTime, stats.ErrorRate)
	}
	if stats.LastRequestAt != nil {
		t.Errorf("requestStats.lastRequestAt = %v, want nil", stats.LastRequestAt)
	}
	if len(stats.RequestRates) != 7 {
		t.Fatalf("requestStats.requestRates len = %d, want 7", len(stats.RequestRates))
	}
	for i, rate := range stats.RequestRates {
		if rate != 0 {
			t.Errorf("requestStats.requestRates[%d] = %d, want 0", i, rate)
		}
	}
	if len(stats.ErrorRates) != 7 {
		t.Fatalf("requestStats.errorRates len = %d, want 7", len(stats.ErrorRates))
	}
	for i, rate := range stats.ErrorRates {
		if rate != 0 {
			t.Errorf("requestStats.errorRates[%d] = %d, want 0", i, rate)
		}
	}

	if len(workload.Runtime.ContainerGroups) != 1 {
		t.Fatalf("containerGroups len = %d, want 1", len(workload.Runtime.ContainerGroups))
	}
	group := workload.Runtime.ContainerGroups[0]
	if group.Name != "default" {
		t.Errorf("container group name = %q, want default", group.Name)
	}
	if group.BundleSelectionPolicy == nil || *group.BundleSelectionPolicy != "availability" {
		t.Errorf("bundleSelectionPolicy = %v, want availability", group.BundleSelectionPolicy)
	}
	if len(group.ResourceBundles) != 1 || group.ResourceBundles[0] != "cpu.small" {
		t.Errorf("resourceBundles = %v, want [cpu.small]", group.ResourceBundles)
	}
	if group.Autoscaling != nil {
		t.Errorf("autoscaling = %v, want nil when API returns null", group.Autoscaling)
	}
	if group.ReplicaCount == nil || *group.ReplicaCount != 1 {
		t.Errorf("replicaCount = %v, want 1", group.ReplicaCount)
	}
	if len(group.Containers) != 1 || group.Containers[0].Name != "Primary container" {
		t.Errorf("container not parsed correctly: %+v", group.Containers)
	}
	ra := group.Containers[0].ResourceAllocation
	if ra == nil || ra.GPU != nil || ra.CPU == nil || *ra.CPU != 1.0 || ra.Memory == nil || *ra.Memory != 536870912 {
		t.Errorf("resourceAllocation not parsed correctly: %+v", ra)
	}
	if group.ResolvedBundle == nil {
		t.Fatal("resolvedBundle not parsed")
	}
	if group.ResolvedBundle.ID != "cpu.small" || group.ResolvedBundle.CPUCount != 1.0 || group.ResolvedBundle.MemoryBytes != 536870912 {
		t.Errorf("resolvedBundle = %+v, unexpected values", group.ResolvedBundle)
	}
	if group.ResolvedBundle.GPUMaker != nil || group.ResolvedBundle.GPUTypeLabel != nil {
		t.Errorf("resolvedBundle gpu fields should be nil, got %+v", group.ResolvedBundle)
	}
	if group.ResolvedBundle.GPUCount == nil || *group.ResolvedBundle.GPUCount != 0 {
		t.Errorf("resolvedBundle.gpuCount = %v, want 0", group.ResolvedBundle.GPUCount)
	}

	if len(workload.Owners) != 1 {
		t.Fatalf("owners len = %d, want 1", len(workload.Owners))
	}
	assertUserData(t, "owners[0]", &workload.Owners[0])
}

func TestWorkloadUnmarshalJSON_autoscaling(t *testing.T) {
	t.Parallel()

	t.Run("null autoscaling in container group", func(t *testing.T) {
		t.Parallel()

		payload := `{
			"runtime": {
				"containerGroups": [
					{
						"name": "default",
						"replicaCount": 1,
						"autoscaling": null
					}
				]
			}
		}`

		var workload Workload
		if err := json.Unmarshal([]byte(payload), &workload); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if len(workload.Runtime.ContainerGroups) != 1 {
			t.Fatalf("containerGroups len = %d, want 1", len(workload.Runtime.ContainerGroups))
		}
		if workload.Runtime.ContainerGroups[0].Autoscaling != nil {
			t.Errorf("autoscaling = %v, want nil when API returns null", workload.Runtime.ContainerGroups[0].Autoscaling)
		}
	})

	t.Run("populated autoscaling in container group", func(t *testing.T) {
		t.Parallel()

		payload := `{
			"runtime": {
				"containerGroups": [
					{
						"name": "default",
						"autoscaling": {
							"enabled": true,
							"policies": [
								{
									"scalingMetric": "cpuAverageUtilization",
									"target": 50.0,
									"minCount": 1,
									"maxCount": 3,
									"priority": 1
								}
							]
						}
					}
				]
			}
		}`

		var workload Workload
		if err := json.Unmarshal([]byte(payload), &workload); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		group := workload.Runtime.ContainerGroups[0]
		if group.Autoscaling == nil {
			t.Fatal("autoscaling not parsed")
		}
		if group.Autoscaling.Enabled == nil || !*group.Autoscaling.Enabled {
			t.Errorf("autoscaling.enabled = %v, want true", group.Autoscaling.Enabled)
		}
		if len(group.Autoscaling.Policies) != 1 {
			t.Fatalf("autoscaling.policies len = %d, want 1", len(group.Autoscaling.Policies))
		}
		policy := group.Autoscaling.Policies[0]
		if policy.ScalingMetric != "cpuAverageUtilization" {
			t.Errorf("scalingMetric = %q, want cpuAverageUtilization", policy.ScalingMetric)
		}
		if policy.Target != 50.0 || policy.MinCount != 1 || policy.MaxCount != 3 {
			t.Errorf("policy values = (%v, %d, %d), want (50.0, 1, 3)", policy.Target, policy.MinCount, policy.MaxCount)
		}
		if policy.Priority == nil || *policy.Priority != 1 {
			t.Errorf("priority = %v, want 1", policy.Priority)
		}
	})
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

// workloadJSON builds a GET /workloads/{id}/ response body. Pass replacement as
// nil to represent a cleared/absent replacement (JSON null).
func workloadJSON(status ProtonStatus, replacement map[string]any) map[string]any {
	return map[string]any{
		"id":          "wl-1",
		"name":        "wl-1",
		"status":      status,
		"importance":  WorkloadImportanceLow,
		"runtime":     map[string]any{},
		"replacement": replacement,
	}
}

func TestWaitForWorkloadReplacementSucceedsWhenRecordClearsAfterActive(t *testing.T) {
	// The real-world completion path: an active record is observed, then it is
	// cleaned up to null while the workload is running. The transient
	// "completed" status is gone before we can see it, but null-after-active on
	// a running workload unambiguously means done.
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(workloadJSON(ProtonStatusRunning, replacementJSON(ReplacementStatusPromoting, "")))
			return
		}
		_ = json.NewEncoder(w).Encode(workloadJSON(ProtonStatusRunning, nil))
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	_, err := svc.WaitForWorkloadReplacement(context.Background(), "wl-1", &WaitForWorkloadReplacementOptions{
		PollInterval: 5 * time.Millisecond,
		Timeout:      time.Second,
	})
	if err != nil {
		t.Fatalf("WaitForWorkloadReplacement returned error: %v", err)
	}
	if calls < 2 {
		t.Fatalf("expected the active record to be observed before it cleared, got %d calls", calls)
	}
}

func TestWaitForWorkloadReplacementSucceedsOnCompletedStatus(t *testing.T) {
	// If the brief "completed" window happens to be caught, that is also success.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(workloadJSON(ProtonStatusRunning, replacementJSON(ReplacementStatusCompleted, "")))
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
	if resp == nil || resp.Status != ReplacementStatusCompleted {
		t.Fatalf("expected completed replacement, got %+v", resp)
	}
}

func TestWaitForWorkloadReplacementIgnoresInitialNullGap(t *testing.T) {
	// A null replacement on the first poll is the gap before the API creates
	// the record, not a completed replacement, so it must not short-circuit to
	// success: null -> active -> null(running) still waits for the active
	// record before settling.
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		switch calls {
		case 1:
			_ = json.NewEncoder(w).Encode(workloadJSON(ProtonStatusRunning, nil))
		case 2:
			_ = json.NewEncoder(w).Encode(workloadJSON(ProtonStatusInitializing, replacementJSON(ReplacementStatusInitializing, "")))
		default:
			_ = json.NewEncoder(w).Encode(workloadJSON(ProtonStatusRunning, nil))
		}
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	_, err := svc.WaitForWorkloadReplacement(context.Background(), "wl-1", &WaitForWorkloadReplacementOptions{
		PollInterval: 5 * time.Millisecond,
		Timeout:      time.Second,
	})
	if err != nil {
		t.Fatalf("WaitForWorkloadReplacement returned error: %v", err)
	}
	if calls < 3 {
		t.Fatalf("expected the initial null to be ignored (>=3 calls), got %d", calls)
	}
}

func TestWaitForWorkloadReplacementReturnsReplacementFailedError(t *testing.T) {
	// An errored replacement record persists on the workload and must surface as
	// a ReplacementFailedError (it never clears to null).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(workloadJSON(ProtonStatusRunning, replacementJSON(ReplacementStatusErrored, "candidate proton failed health checks")))
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
	// A replacement that never settles (stuck non-terminal) must time out.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(workloadJSON(ProtonStatusInitializing, replacementJSON(ReplacementStatusFinalizing, "")))
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
