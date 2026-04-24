package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccWorkloadResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_workload.test"
	name := "workload-" + nameSalt
	var initialID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadAccConfig(name, "", "low", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "endpoint"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "importance", "low"),
					captureAttr(resourceName, "id", &initialID),
					checkWorkloadExistsInAPI(resourceName, name, false),
				),
			},
			{
				Config: workloadAccConfig("updated-"+name, "test description", "high", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "updated-"+name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "importance", "high"),
					checkWorkloadIDPreserved(resourceName, &initialID),
					checkWorkloadExistsInAPI(resourceName, "updated-"+name, false),
				),
			},
		},
	})
}

func TestIntegrationWorkloadResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	id := uuid.NewString()
	artifactID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount := int64(2)
	endpoint := "https://workloads.example.com/" + id

	workload := workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)
	updatedName := "updated-" + name
	updatedWorkload := workloadFixture(id, artifactID, updatedName, "test description", client.WorkloadImportanceHigh, &replicaCount, &endpoint)

	// Create
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload, nil) // waitForRunning poll
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload, nil) // waitForRunning final
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload, nil) // post-create Read

	// Pre-update Read (step 2 plan refresh)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload, nil)

	// Update metadata
	mockService.EXPECT().UpdateWorkload(gomock.Any(), id, gomock.Any()).Return(updatedWorkload, nil)

	// Destroy
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(updatedWorkload, nil) // pre-destroy plan refresh
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).Return(nil)

	var initialID string
	resourceName := "datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "endpoint"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "importance", "low"),
					resource.TestCheckResourceAttr(resourceName, "artifact_id", artifactID),
					captureAttr(resourceName, "id", &initialID),
					checkWorkloadExistsInAPI(resourceName, name, true),
				),
			},
			{
				Config: workloadConfigWithReplicas(updatedName, "test description", "high", artifactID, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "importance", "high"),
					checkWorkloadIDPreserved(resourceName, &initialID),
					checkWorkloadExistsInAPI(resourceName, updatedName, true),
				),
			},
		},
	})
}

func TestIntegrationWorkloadReplaceOnArtifactIDChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	id1 := uuid.NewString()
	id2 := uuid.NewString()
	artifactID1 := uuid.NewString()
	artifactID2 := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount := int64(1)
	endpoint1 := "https://workloads.example.com/" + id1
	endpoint2 := "https://workloads.example.com/" + id2

	workload1 := workloadFixture(id1, artifactID1, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint1)
	workload2 := workloadFixture(id2, artifactID2, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint2)

	// Step 1: Create
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // waitForRunning poll
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // waitForRunning final
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // post-create Read

	// Pre-replace plan refresh
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)

	// Step 2: Replace (new artifact_id triggers replacement)
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id1).Return(nil)
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload2, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id2).Return(workload2, nil) // waitForRunning poll
	mockService.EXPECT().GetWorkload(gomock.Any(), id2).Return(workload2, nil) // waitForRunning final
	mockService.EXPECT().GetWorkload(gomock.Any(), id2).Return(workload2, nil) // post-replace perpetual diff check

	// Destroy
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id2).Return(nil)

	var initialID string
	resourceName := "datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID1, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "artifact_id", artifactID1),
					captureAttr(resourceName, "id", &initialID),
					checkWorkloadExistsInAPI(resourceName, name, true),
				),
			},
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID2, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "artifact_id", artifactID2),
					checkWorkloadIDChanged(resourceName, &initialID),
					checkWorkloadExistsInAPI(resourceName, name, true),
				),
			},
		},
	})
}

func TestIntegrationWorkloadReplaceOnReplicaCountChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	id1 := uuid.NewString()
	id2 := uuid.NewString()
	artifactID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount1 := int64(1)
	replicaCount2 := int64(3)
	endpoint1 := "https://workloads.example.com/" + id1
	endpoint2 := "https://workloads.example.com/" + id2

	workload1 := workloadFixture(id1, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount1, &endpoint1)
	workload2 := workloadFixture(id2, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount2, &endpoint2)

	// Step 1: Create
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // waitForRunning poll
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // waitForRunning final
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // post-create Read

	// Pre-replace plan refresh
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)

	// Step 2: Replace (changed replica_count triggers replacement via runtime.RequiresReplace)
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id1).Return(nil)
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload2, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id2).Return(workload2, nil) // waitForRunning poll
	mockService.EXPECT().GetWorkload(gomock.Any(), id2).Return(workload2, nil) // waitForRunning final
	mockService.EXPECT().GetWorkload(gomock.Any(), id2).Return(workload2, nil) // post-replace perpetual diff check

	// Destroy
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id2).Return(nil)

	var initialID string
	resourceName := "datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.replica_count", "1"),
					captureAttr(resourceName, "id", &initialID),
					checkWorkloadExistsInAPI(resourceName, name, true),
				),
			},
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID, 3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.replica_count", "3"),
					checkWorkloadIDChanged(resourceName, &initialID),
					checkWorkloadExistsInAPI(resourceName, name, true),
				),
			},
		},
	})
}

func TestIntegrationWorkloadReplaceOnResourcesChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	id1 := uuid.NewString()
	id2 := uuid.NewString()
	artifactID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount := int64(1)
	endpoint1 := "https://workloads.example.com/" + id1
	endpoint2 := "https://workloads.example.com/" + id2

	workload1 := workloadFixtureWithResources(id1, artifactID, name, &replicaCount, &endpoint1, nil)
	workload2 := workloadFixtureWithResources(id2, artifactID, name, &replicaCount, &endpoint2, []client.ResourceBundleResources{
		{Type: "resource_bundle", ResourceBundleID: "cpu.small"},
	})

	// Step 1: Create without resources
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // waitForRunning poll
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // waitForRunning final
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // post-create Read

	// Pre-replace plan refresh
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)

	// Step 2: Replace (adding resources triggers replacement via runtime.RequiresReplace)
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id1).Return(nil)
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload2, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id2).Return(workload2, nil) // waitForRunning poll
	mockService.EXPECT().GetWorkload(gomock.Any(), id2).Return(workload2, nil) // waitForRunning final
	mockService.EXPECT().GetWorkload(gomock.Any(), id2).Return(workload2, nil) // post-replace perpetual diff check

	// Destroy
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id2).Return(nil)

	var initialID string
	resourceName := "datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					captureAttr(resourceName, "id", &initialID),
					checkWorkloadExistsInAPI(resourceName, name, true),
				),
			},
			{
				Config: workloadConfigWithReplicasAndResources(name, "", "low", artifactID, 1, "cpu.small"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.resources.0.resource_bundle_id", "cpu.small"),
					checkWorkloadIDChanged(resourceName, &initialID),
					checkWorkloadExistsInAPI(resourceName, name, true),
				),
			},
		},
	})
}

func TestIntegrationWorkloadReplaceOnAutoscalingChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	id1 := uuid.NewString()
	id2 := uuid.NewString()
	artifactID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	endpoint1 := "https://workloads.example.com/" + id1
	endpoint2 := "https://workloads.example.com/" + id2

	workload1 := workloadFixtureWithAutoscaling(id1, artifactID, name, &endpoint1, 1, 3, 50.0)
	workload2 := workloadFixtureWithAutoscaling(id2, artifactID, name, &endpoint2, 2, 5, 70.0)

	// Step 1: Create with autoscaling
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // waitForRunning poll
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // waitForRunning final
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // post-create Read

	// Pre-replace plan refresh
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)

	// Step 2: Replace (changed autoscaling triggers replacement via runtime.RequiresReplace)
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id1).Return(nil)
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload2, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id2).Return(workload2, nil) // waitForRunning poll
	mockService.EXPECT().GetWorkload(gomock.Any(), id2).Return(workload2, nil) // waitForRunning final
	mockService.EXPECT().GetWorkload(gomock.Any(), id2).Return(workload2, nil) // post-replace perpetual diff check

	// Destroy
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id2).Return(nil)

	var initialID string
	resourceName := "datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithAutoscaling(name, "", "low", artifactID, 1, 3, 50.0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "runtime.autoscaling.policies.0.min_count", "1"),
					captureAttr(resourceName, "id", &initialID),
					checkWorkloadExistsInAPI(resourceName, name, true),
				),
			},
			{
				Config: workloadConfigWithAutoscaling(name, "", "low", artifactID, 2, 5, 70.0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.autoscaling.policies.0.min_count", "2"),
					checkWorkloadIDChanged(resourceName, &initialID),
					checkWorkloadExistsInAPI(resourceName, name, true),
				),
			},
		},
	})
}

func TestWorkloadConflictingRuntimeConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	artifactID := uuid.NewString()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      workloadConfigConflictingRuntime(artifactID),
				ExpectError: regexp.MustCompile("Conflicting runtime configuration"),
			},
		},
	})
}

// ─── check functions ───────────────────────────────────────────────────────────

func checkWorkloadExistsInAPI(resourceName, expectedName string, isMock bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("workload ID is not set in state")
		}
		if isMock {
			return nil
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("provider not found")
		}
		p.service = NewService(cl)

		workload, err := p.service.GetWorkload(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("GetWorkload(%s): %w", rs.Primary.ID, err)
		}
		if workload.Name != expectedName {
			return fmt.Errorf("expected workload name %q, got %q", expectedName, workload.Name)
		}
		return nil
	}
}

func checkWorkloadIDPreserved(resourceName string, initialID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		if *initialID != "" && rs.Primary.ID != *initialID {
			return fmt.Errorf("workload ID changed after metadata update: %q → %q", *initialID, rs.Primary.ID)
		}
		return nil
	}
}

func checkWorkloadIDChanged(resourceName string, initialID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		if *initialID != "" && rs.Primary.ID == *initialID {
			return fmt.Errorf("expected workload ID to change after replacement, but it stayed %q", *initialID)
		}
		return nil
	}
}

// ─── config helpers ────────────────────────────────────────────────────────────

func workloadConfigWithReplicas(name, description, importance, artifactID string, replicaCount int64) string {
	desc := ""
	if description != "" {
		desc = fmt.Sprintf("description = %q", description)
	}
	return fmt.Sprintf(`
resource "datarobot_workload" "test" {
  name        = %q
  importance  = %q
  artifact_id = %q
  %s
  runtime = {
    replica_count = %d
  }
}
`, name, importance, artifactID, desc, replicaCount)
}

func workloadConfigWithReplicasAndResources(name, description, importance, artifactID string, replicaCount int64, resourceBundleID string) string {
	desc := ""
	if description != "" {
		desc = fmt.Sprintf("description = %q", description)
	}
	return fmt.Sprintf(`
resource "datarobot_workload" "test" {
  name        = %q
  importance  = %q
  artifact_id = %q
  %s
  runtime = {
    replica_count = %d
    resources = [
      { resource_bundle_id = %q }
    ]
  }
}
`, name, importance, artifactID, desc, replicaCount, resourceBundleID)
}

func workloadConfigWithAutoscaling(name, description, importance, artifactID string, minCount, maxCount int64, target float64) string {
	desc := ""
	if description != "" {
		desc = fmt.Sprintf("description = %q", description)
	}
	return fmt.Sprintf(`
resource "datarobot_workload" "test" {
  name        = %q
  importance  = %q
  artifact_id = %q
  %s
  runtime = {
    autoscaling = {
      enabled = true
      policies = [
        {
          scaling_metric = "cpuAverageUtilization"
          target         = %g
          min_count      = %d
          max_count      = %d
        }
      ]
    }
  }
}
`, name, importance, artifactID, desc, target, minCount, maxCount)
}

func workloadConfigConflictingRuntime(artifactID string) string {
	return fmt.Sprintf(`
resource "datarobot_workload" "test" {
  name        = "conflict-test"
  artifact_id = %q
  runtime = {
    replica_count = 2
    autoscaling = {
      enabled = true
      policies = [
        {
          scaling_metric = "cpuAverageUtilization"
          target         = 50
          min_count      = 1
          max_count      = 4
        }
      ]
    }
  }
}
`, artifactID)
}

func workloadAccConfig(name, description, importance string, replicaCount int64) string {
	artifactName := "acc-artifact-" + nameSalt
	desc := ""
	if description != "" {
		desc = fmt.Sprintf("description = %q", description)
	}
	return fmt.Sprintf(`
resource "datarobot_artifact" "test_artifact" {
  name = %q
  type = "service"

  spec = {
    container_groups = [
      {
        containers = [
          {
            name      = "main"
            image_uri = "nginx:latest"
            port      = 8080
            primary   = true
            resource_request = {
              cpu    = 1
              memory = 512
            }
          }
        ]
      }
    ]
  }
}

resource "datarobot_workload" "test" {
  name        = %q
  importance  = %q
  artifact_id = datarobot_artifact.test_artifact.id
  %s
  runtime = {
    replica_count = %d
  }
}
`, artifactName, name, importance, desc, replicaCount)
}

// ─── fixture helpers ───────────────────────────────────────────────────────────

func workloadFixture(id, artifactID, name, description string, importance client.WorkloadImportance, replicaCount *int64, endpoint *string) *client.Workload {
	return &client.Workload{
		ID:          id,
		Name:        name,
		Description: description,
		Status:      client.ProtonStatusRunning,
		Importance:  importance,
		ArtifactID:  &artifactID,
		Endpoint:    endpoint,
		Runtime: client.ProtonRuntime{
			ReplicaCount: replicaCount,
		},
	}
}

func workloadFixtureWithResources(id, artifactID, name string, replicaCount *int64, endpoint *string, resources []client.ResourceBundleResources) *client.Workload {
	w := workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, replicaCount, endpoint)
	w.Runtime.Resources = resources
	return w
}

func workloadFixtureWithAutoscaling(id, artifactID, name string, endpoint *string, minCount, maxCount int64, target float64) *client.Workload {
	enabled := true
	return &client.Workload{
		ID:         id,
		Name:       name,
		Status:     client.ProtonStatusRunning,
		Importance: client.WorkloadImportanceLow,
		ArtifactID: &artifactID,
		Endpoint:   endpoint,
		Runtime: client.ProtonRuntime{
			Autoscaling: &client.AutoscalingProperties{
				Enabled: &enabled,
				Policies: []client.AutoscalingPolicy{
					{
						ScalingMetric: "cpuAverageUtilization",
						Target:        target,
						MinCount:      minCount,
						MaxCount:      maxCount,
					},
				},
			},
		},
	}
}
