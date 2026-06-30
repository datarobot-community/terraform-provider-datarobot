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
					resource.TestCheckResourceAttr(dataSourceName, "id", workload.ID),
					resource.TestCheckResourceAttr(dataSourceName, "name", workload.Name),
					resource.TestCheckResourceAttr(dataSourceName, "description", *workload.Description),
					resource.TestCheckResourceAttr(dataSourceName, "status", string(workload.Status)),
					resource.TestCheckResourceAttr(dataSourceName, "importance", string(workload.Importance)),
					resource.TestCheckResourceAttr(dataSourceName, "type", string(workload.Type)),
					resource.TestCheckResourceAttr(dataSourceName, "creator.username", "jane"),
					resource.TestCheckResourceAttr(dataSourceName, "artifact.name", "my-artifact"),
					resource.TestCheckResourceAttr(dataSourceName, "replacement.strategy", "rolling"),
					resource.TestCheckResourceAttr(dataSourceName, "permissions.0", "CAN_VIEW"),
					resource.TestCheckResourceAttr(dataSourceName, "request_stats.total_requests", "100"),
					resource.TestCheckResourceAttr(dataSourceName, "tags.0.name", "env"),
					resource.TestCheckResourceAttr(dataSourceName, "owners.0.username", "jane"),
					resource.TestCheckResourceAttr(dataSourceName, "runtime.container_groups.0.resolved_bundle.id", "cpu.small"),
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
