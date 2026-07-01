package client

import (
	"encoding/json"
	"fmt"
	"testing"
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
