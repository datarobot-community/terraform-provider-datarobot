package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWorkloadDataSource(t *testing.T) {
	t.Parallel()

	name := "workload-ds-" + nameSalt
	dataSourceName := "data.datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadDataSourceAccConfig(name, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "id", "datarobot_workload.test", "id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "status"),
					resource.TestCheckResourceAttrSet(dataSourceName, "endpoint"),
					resource.TestCheckResourceAttrSet(dataSourceName, "artifact_id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "created_at"),
					resource.TestCheckResourceAttrSet(dataSourceName, "updated_at"),
					resource.TestCheckResourceAttr(dataSourceName, "importance", "low"),
					resource.TestCheckResourceAttr(dataSourceName, "type", "service"),
					resource.TestCheckResourceAttrSet(dataSourceName, "runtime.container_groups.0.name"),
				),
			},
		},
	})
}

func TestIntegrationWorkloadDataSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
		globalTestCfg.ApiKey = "fake"
	}

	workload := workloadDataSourceFixture()

	mockService.EXPECT().GetWorkload(gomock.Any(), workload.ID).Return(workload, nil).AnyTimes()

	dataSourceName := "data.datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			t.Setenv(DataRobotApiKeyEnvVar, "fake")
			globalTestCfg.ApiKey = "fake"
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadDataSourceConfig(workload.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					workloadDataSourceTestCheckFuncs(dataSourceName, workload)...,
				),
			},
		},
	})
}

func TestIntegrationWorkloadDataSourceNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
		globalTestCfg.ApiKey = "fake"
	}

	missingID := uuid.NewString()
	mockService.EXPECT().GetWorkload(gomock.Any(), missingID).Return(nil, client.NewNotFoundError("workload")).AnyTimes()

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			t.Setenv(DataRobotApiKeyEnvVar, "fake")
			globalTestCfg.ApiKey = "fake"
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      workloadDataSourceConfig(missingID),
				ExpectError: regexp.MustCompile("Failed to get Workload"),
			},
		},
	})
}

func workloadDataSourceAccConfig(name string, replicaCount int64) string {
	return fmt.Sprintf(`
%s

data "datarobot_workload" "test" {
  id = datarobot_workload.test.id
}
`, workloadAccConfig(name, "", "low", replicaCount))
}

func workloadDataSourceConfig(id string) string {
	return fmt.Sprintf(`
data "datarobot_workload" "test" {
  id = %q
}
`, id)
}

func workloadDataSourceTestCheckFuncs(dataSourceName string, w *client.Workload) []resource.TestCheckFunc {
	creator := w.Creator
	artifact := w.Artifact
	replacement := w.Replacement
	group := w.Runtime.ContainerGroups[0]
	autoscaling := group.Autoscaling
	policy := autoscaling.Policies[0]
	container := group.Containers[0]
	alloc := container.ResourceAllocation
	resolved := group.ResolvedBundle
	stats := w.RequestStats
	tag := w.Tags[0]
	owner := w.Owners[0]

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(dataSourceName, "id", w.ID),
		resource.TestCheckResourceAttr(dataSourceName, "name", w.Name),
		resource.TestCheckResourceAttr(dataSourceName, "description", *w.Description),
		resource.TestCheckResourceAttr(dataSourceName, "created_at", w.CreatedAt),
		resource.TestCheckResourceAttr(dataSourceName, "updated_at", w.UpdatedAt),
		resource.TestCheckResourceAttr(dataSourceName, "creator.id", creator.ID),
		resource.TestCheckResourceAttr(dataSourceName, "creator.full_name", *creator.FullName),
		resource.TestCheckResourceAttr(dataSourceName, "creator.email", *creator.Email),
		resource.TestCheckResourceAttr(dataSourceName, "creator.username", *creator.Username),
		resource.TestCheckResourceAttr(dataSourceName, "creator.userhash", *creator.Userhash),
		resource.TestCheckResourceAttr(dataSourceName, "proton_id", *w.ProtonID),
		resource.TestCheckResourceAttr(dataSourceName, "artifact_id", *w.ArtifactID),
		resource.TestCheckResourceAttr(dataSourceName, "artifact.id", artifact.ID),
		resource.TestCheckResourceAttr(dataSourceName, "artifact.name", *artifact.Name),
		resource.TestCheckResourceAttr(dataSourceName, "artifact.type", string(*artifact.Type)),
		resource.TestCheckResourceAttr(dataSourceName, "artifact.status", string(*artifact.Status)),
		resource.TestCheckResourceAttr(dataSourceName, "artifact.version", fmt.Sprintf("%d", *artifact.Version)),
		resource.TestCheckResourceAttr(dataSourceName, "artifact.artifact_repository_id", *artifact.ArtifactRepositoryID),
		resource.TestCheckResourceAttr(dataSourceName, "artifact.template_id", *artifact.TemplateID),
		resource.TestCheckResourceAttr(dataSourceName, "type", string(w.Type)),
		resource.TestCheckResourceAttr(dataSourceName, "status", string(w.Status)),
		resource.TestCheckResourceAttr(dataSourceName, "replacement.status", replacement.Status),
		resource.TestCheckResourceAttr(dataSourceName, "replacement.candidate_proton_ids.0", replacement.CandidateProtonIDs[0]),
		resource.TestCheckResourceAttr(dataSourceName, "replacement.strategy", replacement.Strategy),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.name", group.Name),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.replica_count", fmt.Sprintf("%d", *group.ReplicaCount)),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.bundle_selection_policy", *group.BundleSelectionPolicy),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.resource_bundles.0", group.ResourceBundles[0]),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.autoscaling.enabled", "true"),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.autoscaling.policies.0.scaling_metric", policy.ScalingMetric),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.autoscaling.policies.0.target", fmt.Sprintf("%g", policy.Target)),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.autoscaling.policies.0.min_count", fmt.Sprintf("%d", policy.MinCount)),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.autoscaling.policies.0.max_count", fmt.Sprintf("%d", policy.MaxCount)),
		resource.TestCheckNoResourceAttr(dataSourceName, "runtime.container_groups.0.autoscaling.policies.0.priority"),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.containers.0.name", container.Name),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.containers.0.resource_allocation.cpu", fmt.Sprintf("%g", *alloc.CPU)),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.containers.0.resource_allocation.memory", fmt.Sprintf("%d", *alloc.Memory)),
		resource.TestCheckNoResourceAttr(dataSourceName, "runtime.container_groups.0.containers.0.resource_allocation.gpu"),
		resource.TestCheckNoResourceAttr(dataSourceName, "runtime.container_groups.0.containers.0.resource_allocation.gpu_memory"),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.resolved_bundle.id", resolved.ID),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.resolved_bundle.cpu_count", fmt.Sprintf("%g", resolved.CPUCount)),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.resolved_bundle.memory_bytes", fmt.Sprintf("%d", resolved.MemoryBytes)),
		resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.resolved_bundle.gpu_count", fmt.Sprintf("%d", *resolved.GPUCount)),
		resource.TestCheckNoResourceAttr(dataSourceName, "runtime.container_groups.0.resolved_bundle.gpu_maker"),
		resource.TestCheckNoResourceAttr(dataSourceName, "runtime.container_groups.0.resolved_bundle.gpu_type_label"),
		resource.TestCheckResourceAttr(dataSourceName, "permissions.0", w.Permissions[0]),
		resource.TestCheckResourceAttr(dataSourceName, "permissions.1", w.Permissions[1]),
		resource.TestCheckResourceAttr(dataSourceName, "importance", string(w.Importance)),
		resource.TestCheckResourceAttr(dataSourceName, "request_stats.total_requests", fmt.Sprintf("%d", stats.TotalRequests)),
		resource.TestCheckResourceAttr(dataSourceName, "request_stats.concurrent_requests", fmt.Sprintf("%d", stats.ConcurrentRequests)),
		resource.TestCheckResourceAttr(dataSourceName, "request_stats.last_request_at", *stats.LastRequestAt),
		resource.TestCheckResourceAttr(dataSourceName, "request_stats.response_time", fmt.Sprintf("%d", stats.ResponseTime)),
		resource.TestCheckResourceAttr(dataSourceName, "request_stats.error_rate", fmt.Sprintf("%g", stats.ErrorRate)),
		resource.TestCheckResourceAttr(dataSourceName, "tags.0.id", tag.ID),
		resource.TestCheckResourceAttr(dataSourceName, "tags.0.name", tag.Name),
		resource.TestCheckResourceAttr(dataSourceName, "tags.0.value", tag.Value),
		resource.TestCheckResourceAttr(dataSourceName, "endpoint", *w.Endpoint),
		resource.TestCheckResourceAttr(dataSourceName, "last_response", *w.LastResponse),
		resource.TestCheckResourceAttr(dataSourceName, "owners.0.id", owner.ID),
		resource.TestCheckResourceAttr(dataSourceName, "owners.0.username", *owner.Username),
		resource.TestCheckNoResourceAttr(dataSourceName, "owners.0.full_name"),
		resource.TestCheckNoResourceAttr(dataSourceName, "owners.0.email"),
		resource.TestCheckNoResourceAttr(dataSourceName, "owners.0.userhash"),
	}

	for i, rate := range stats.RequestRates {
		checks = append(checks, resource.TestCheckResourceAttr(
			dataSourceName,
			fmt.Sprintf("request_stats.request_rates.%d", i),
			fmt.Sprintf("%d", rate),
		))
	}
	for i, rate := range stats.ErrorRates {
		checks = append(checks, resource.TestCheckResourceAttr(
			dataSourceName,
			fmt.Sprintf("request_stats.error_rates.%d", i),
			fmt.Sprintf("%d", rate),
		))
	}

	return checks
}

func workloadDataSourceFixture() *client.Workload {
	fullName := "Jane Doe"
	email := "jane@example.com"
	username := "jane"
	userhash := "abc123"
	artifactName := "my-artifact"
	artifactType := client.ArtifactTypeService
	artifactStatus := client.ArtifactStatusLocked
	version := 2
	artifactRepoID := uuid.NewString()
	templateID := uuid.NewString()
	protonID := uuid.NewString()
	artifactID := uuid.NewString()
	id := uuid.NewString()
	endpoint := "https://workloads.example.com/" + id
	lastResponse := "2026-01-16T11:59:30Z"
	lastRequestAt := "2026-01-16T11:59:00Z"
	replicaCount := int64(2)
	gpuCount := 0
	cpu := 1.0
	memory := int64(8589934592)
	enabled := true
	bundlePolicy := "availability"

	return &client.Workload{
		ID:          id,
		Name:        "my-workload",
		Description: strPtr("test workload"),
		CreatedAt:   "2026-01-15T10:30:00Z",
		UpdatedAt:   "2026-01-16T12:00:00Z",
		Creator: &client.UserData{
			ID:       uuid.NewString(),
			FullName: &fullName,
			Email:    &email,
			Username: &username,
			Userhash: &userhash,
		},
		ProtonID:   &protonID,
		ArtifactID: &artifactID,
		Artifact: &client.WorkloadArtifactInfo{
			ID:                   artifactID,
			Name:                 &artifactName,
			Type:                 &artifactType,
			Status:               &artifactStatus,
			Version:              &version,
			ArtifactRepositoryID: &artifactRepoID,
			TemplateID:           &templateID,
		},
		Type:   client.ArtifactTypeService,
		Status: client.ProtonStatusRunning,
		Replacement: &client.WorkloadReplacement{
			Status:             "in_progress",
			CandidateProtonIDs: []string{uuid.NewString()},
			Strategy:           "rolling",
		},
		Runtime: client.WorkloadRuntime{
			ContainerGroups: []client.GroupRuntime{
				{
					Name:                  "default",
					ReplicaCount:          &replicaCount,
					BundleSelectionPolicy: &bundlePolicy,
					ResourceBundles:       []string{"cpu.small"},
					Autoscaling: &client.AutoscalingProperties{
						Enabled: &enabled,
						Policies: []client.AutoscalingPolicy{
							{
								ScalingMetric: "cpuAverageUtilization",
								Target:        50.0,
								MinCount:      1,
								MaxCount:      3,
							},
						},
					},
					Containers: []client.ContainerOverride{
						{
							Name: "main",
							ResourceAllocation: &client.ResourceAllocation{
								CPU:    &cpu,
								Memory: &memory,
							},
						},
					},
					ResolvedBundle: &client.ResolvedBundle{
						ID:          "cpu.small",
						CPUCount:    1.0,
						MemoryBytes: 8589934592,
						GPUCount:    &gpuCount,
					},
				},
			},
		},
		Permissions: []string{"CAN_VIEW", "CAN_UPDATE"},
		Importance:  client.WorkloadImportanceHigh,
		RequestStats: &client.RequestStats{
			TotalRequests:      100,
			ConcurrentRequests: 3,
			LastRequestAt:      &lastRequestAt,
			ResponseTime:       42,
			ErrorRate:          1.5,
			RequestRates:       []int{10, 12, 8, 9, 11, 7, 13},
			ErrorRates:         []int{0, 1, 0, 2, 1, 0, 1},
		},
		Tags: []client.TagInfo{
			{ID: uuid.NewString(), Name: "env", Value: "staging"},
		},
		Endpoint:     &endpoint,
		LastResponse: &lastResponse,
		Owners: []client.UserData{
			{ID: uuid.NewString(), Username: &username},
		},
	}
}
