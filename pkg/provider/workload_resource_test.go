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
					checkWorkloadExistsInAPI(name, false),
				),
			},
			{
				Config: workloadAccConfig("updated-"+name, "test description", "high", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "updated-"+name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "importance", "high"),
					checkWorkloadIDPreserved(&initialID),
					checkWorkloadExistsInAPI("updated-"+name, false),
				),
			},
			{
				Config: workloadAccConfig("updated-"+name, "test description", "high", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.container_groups.0.replica_count", "2"),
					checkWorkloadIDPreserved(&initialID),
					checkWorkloadExistsInAPI("updated-"+name, false),
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
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload, nil) // waitForRunning
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload, nil) // post-create Read

	// Pre-update Read (step 2 plan refresh)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload, nil)

	// Update metadata
	mockService.EXPECT().UpdateWorkload(gomock.Any(), id, gomock.Any()).Return(updatedWorkload, nil)

	// Destroy
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(updatedWorkload, nil) // pre-destroy plan refresh
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(nil, client.NewNotFoundError("workload")) // poll after delete

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
					checkWorkloadExistsInAPI(name, true),
				),
			},
			{
				Config: workloadConfigWithReplicas(updatedName, "test description", "high", artifactID, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "importance", "high"),
					checkWorkloadIDPreserved(&initialID),
					checkWorkloadExistsInAPI(updatedName, true),
				),
			},
		},
	})
}

func TestIntegrationWorkloadClearDescription(t *testing.T) {
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
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id

	withDesc := workloadFixture(id, artifactID, name, "hello", client.WorkloadImportanceLow, &replicaCount, &endpoint)
	withoutDesc := workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)

	// Step 1: Create with description
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(withDesc, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(withDesc, nil) // waitForRunning
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(withDesc, nil) // post-create Read

	// Step 2: Remove description — expect PATCH with description="" to clear it
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(withDesc, nil) // pre-update plan refresh
	mockService.EXPECT().UpdateWorkload(gomock.Any(), id, updateDescriptionMatcher("")).Return(withoutDesc, nil)

	// Destroy
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(withoutDesc, nil) // pre-destroy plan refresh
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(nil, client.NewNotFoundError("workload")) // poll after delete

	resourceName := "datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithReplicas(name, "hello", "low", artifactID, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "hello"),
				),
			},
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "description"),
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
	artifactID1 := uuid.NewString()
	artifactID2 := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id1

	workload1 := workloadFixture(id1, artifactID1, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)
	workload2 := workloadFixture(id1, artifactID2, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)

	// Step 1: Create
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // waitForRunning
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // post-create Read

	// Pre-update plan refresh
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)

	// Step 2: In-place replacement (new artifact_id)
	expectWorkloadArtifactReplacement(mockService, id1, workload2)

	// Destroy
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id1).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(nil, client.NewNotFoundError("workload")) // poll after delete

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
					checkWorkloadExistsInAPI(name, true),
				),
			},
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID2, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "artifact_id", artifactID2),
					checkWorkloadIDPreserved(&initialID),
					checkWorkloadExistsInAPI(name, true),
				),
			},
		},
	})
}

func TestIntegrationWorkloadReplaceWithReplacementPolicyOnArtifactChange(t *testing.T) {
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
	artifactID1 := uuid.NewString()
	artifactID2 := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id1

	workload1 := workloadFixture(id1, artifactID1, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)
	workload2 := workloadFixture(id1, artifactID2, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)

	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)

	replacement := workloadReplacementFixture(id1)
	mockService.EXPECT().StartWorkloadReplacement(gomock.Any(), id1, startReplacementMatcher{
		artifactID:            artifactID2,
		strategy:              client.ReplacementStrategyRolling,
		warmupDurationMinutes: 5,
		keepOldVersionMinutes: 10,
	}).Return(replacement, nil)
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), id1, gomock.Any()).Return(replacement, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload2, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload2, nil)

	mockService.EXPECT().DeleteWorkload(gomock.Any(), id1).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(nil, client.NewNotFoundError("workload"))

	resourceName := "datarobot_workload.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithReplacementPolicy(name, artifactID1, 1, 5, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.replacement_policy.warmup_minutes", "5"),
					resource.TestCheckResourceAttr(resourceName, "runtime.replacement_policy.keep_old_version_minutes", "10"),
				),
			},
			{
				Config: workloadConfigWithReplacementPolicy(name, artifactID2, 1, 5, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "artifact_id", artifactID2),
				),
			},
		},
	})
}

func TestIntegrationWorkloadReplaceOnReplacementPolicyChange(t *testing.T) {
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
	artifactID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id1

	workload1 := workloadFixture(id1, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)

	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)

	replacement := workloadReplacementFixture(id1)
	mockService.EXPECT().StartWorkloadReplacement(gomock.Any(), id1, startReplacementMatcher{
		artifactID:            artifactID,
		strategy:              client.ReplacementStrategyRolling,
		warmupDurationMinutes: 15,
	}).Return(replacement, nil)
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), id1, gomock.Any()).Return(replacement, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)

	mockService.EXPECT().DeleteWorkload(gomock.Any(), id1).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(nil, client.NewNotFoundError("workload"))

	resourceName := "datarobot_workload.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID, 1),
			},
			{
				Config: workloadConfigWithReplacementPolicy(name, artifactID, 1, 15, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.replacement_policy.warmup_minutes", "15"),
				),
			},
		},
	})
}

func TestIntegrationWorkloadUpdateMetadataAndArtifactChange(t *testing.T) {
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
	artifactID1 := uuid.NewString()
	artifactID2 := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	updatedName := "updated-" + name
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id

	workload1 := workloadFixture(id, artifactID1, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)
	metadataWorkload := workloadFixture(id, artifactID1, updatedName, "new description", client.WorkloadImportanceHigh, &replicaCount, &endpoint)
	workload2 := workloadFixture(id, artifactID2, updatedName, "new description", client.WorkloadImportanceHigh, &replicaCount, &endpoint)

	// Step 1: Create
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload1, nil)

	// Pre-update plan refresh
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload1, nil)

	// Step 2: metadata + artifact in same apply
	mockService.EXPECT().UpdateWorkload(gomock.Any(), id, gomock.Any()).Return(metadataWorkload, nil)
	expectWorkloadArtifactReplacement(mockService, id, workload2)

	// Destroy
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload2, nil)
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(nil, client.NewNotFoundError("workload"))

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
					captureAttr(resourceName, "id", &initialID),
				),
			},
			{
				Config: workloadConfigWithReplicas(updatedName, "new description", "high", artifactID2, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckResourceAttr(resourceName, "artifact_id", artifactID2),
					checkWorkloadIDPreserved(&initialID),
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
	artifactID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount1 := int64(1)
	replicaCount2 := int64(3)
	endpoint := "https://workloads.example.com/" + id1

	workload1 := workloadFixture(id1, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount1, &endpoint)
	workload2 := workloadFixture(id1, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount2, &endpoint)

	// Step 1: Create
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // waitForRunning
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // post-create Read

	// Pre-update plan refresh
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)

	// Step 2: In-place replacement (changed replica_count)
	expectWorkloadRuntimeReplacement(mockService, id1, workload2)

	// Destroy
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id1).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(nil, client.NewNotFoundError("workload")) // poll after delete

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
					resource.TestCheckResourceAttr(resourceName, "runtime.container_groups.0.replica_count", "1"),
					captureAttr(resourceName, "id", &initialID),
					checkWorkloadExistsInAPI(name, true),
				),
			},
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID, 3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.container_groups.0.replica_count", "3"),
					checkWorkloadIDPreserved(&initialID),
					checkWorkloadExistsInAPI(name, true),
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
	artifactID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id1

	workload1 := workloadFixtureWithResources(id1, artifactID, name, &replicaCount, &endpoint, []string{"cpu.small"})
	workload2 := workloadFixtureWithResources(id1, artifactID, name, &replicaCount, &endpoint, []string{"cpu.large"})

	// Step 1: Create with baseline resource bundle
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // waitForRunning
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // post-create Read

	// Pre-update plan refresh
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)

	// Step 2: In-place replacement (changed resource bundles)
	expectWorkloadRuntimeReplacement(mockService, id1, workload2)

	// Destroy
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id1).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(nil, client.NewNotFoundError("workload")) // poll after delete

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
					checkWorkloadExistsInAPI(name, true),
				),
			},
			{
				Config: workloadConfigWithReplicasAndResources(name, "", "low", artifactID, 1, "cpu.large"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.container_groups.0.resource_bundles.0", "cpu.large"),
					checkWorkloadIDPreserved(&initialID),
					checkWorkloadExistsInAPI(name, true),
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
	artifactID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	endpoint := "https://workloads.example.com/" + id1

	workload1 := workloadFixtureWithAutoscaling(id1, artifactID, name, &endpoint, 1, 3, 50.0)
	workload2 := workloadFixtureWithAutoscaling(id1, artifactID, name, &endpoint, 2, 5, 70.0)

	// Step 1: Create with autoscaling
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // waitForRunning
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil) // post-create Read

	// Pre-update plan refresh
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload1, nil)

	// Step 2: In-place replacement (changed autoscaling)
	expectWorkloadRuntimeReplacement(mockService, id1, workload2)

	// Destroy
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id1).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(nil, client.NewNotFoundError("workload")) // poll after delete

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
					resource.TestCheckResourceAttr(resourceName, "runtime.container_groups.0.autoscaling.policies.0.min_count", "1"),
					captureAttr(resourceName, "id", &initialID),
					checkWorkloadExistsInAPI(name, true),
				),
			},
			{
				Config: workloadConfigWithAutoscaling(name, "", "low", artifactID, 2, 5, 70.0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.container_groups.0.autoscaling.policies.0.min_count", "2"),
					checkWorkloadIDPreserved(&initialID),
					checkWorkloadExistsInAPI(name, true),
				),
			},
		},
	})
}

func TestIntegrationWorkloadImportState(t *testing.T) {
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
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id

	workload := workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)

	// Step 1: Create
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload, nil) // waitForRunning
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload, nil) // post-create Read

	// Step 2: ImportState — ImportState fetches workload, then framework calls Read again
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload, nil) // ImportState fetch
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload, nil) // post-import Read

	// Destroy
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(nil, client.NewNotFoundError("workload")) // poll after delete

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
					resource.TestCheckResourceAttrSet(resourceName, "endpoint"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "importance", "low"),
					resource.TestCheckResourceAttr(resourceName, "artifact_id", artifactID),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
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

func TestWorkloadTooManyContainerGroups(t *testing.T) {
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
				Config:      workloadConfigWithMultipleGroups(artifactID),
				ExpectError: regexp.MustCompile("Too many container groups"),
			},
		},
	})
}

// ─── matchers ─────────────────────────────────────────────────────────────────

type updateDescriptionMatcher string

func (m updateDescriptionMatcher) Matches(x interface{}) bool {
	req, ok := x.(*client.UpdateWorkloadRequest)
	return ok && req.Description != nil && *req.Description == string(m)
}

func (m updateDescriptionMatcher) String() string {
	return fmt.Sprintf("UpdateWorkloadRequest with description=%q", string(m))
}

// ─── check functions ───────────────────────────────────────────────────────────

func checkWorkloadExistsInAPI(expectedName string, isMock bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const rn = "datarobot_workload.test"
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource %s not found in state", rn)
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

func checkWorkloadIDPreserved(initialID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const rn = "datarobot_workload.test"
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource %s not found in state", rn)
		}
		if *initialID != "" && rs.Primary.ID != *initialID {
			return fmt.Errorf("workload ID changed after in-place update: %q → %q", *initialID, rs.Primary.ID)
		}
		return nil
	}
}

func workloadReplacementFixture(workloadID string) *client.WorkloadReplacement {
	return &client.WorkloadReplacement{
		ID:         uuid.NewString(),
		WorkloadID: workloadID,
		Status:     client.ReplacementStatusCompleted,
		Strategy:   client.ReplacementStrategyRolling,
	}
}

func expectWorkloadArtifactReplacement(mockService *mock_client.MockService, workloadID string, updatedWorkload *client.Workload) {
	replacement := workloadReplacementFixture(workloadID)
	mockService.EXPECT().StartWorkloadReplacement(gomock.Any(), workloadID, gomock.Any()).Return(replacement, nil)
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), workloadID, gomock.Any()).Return(replacement, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), workloadID).Return(updatedWorkload, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), workloadID).Return(updatedWorkload, nil) // post-apply refresh Read
}

func expectWorkloadRuntimeReplacement(mockService *mock_client.MockService, workloadID string, updatedWorkload *client.Workload) {
	replacement := workloadReplacementFixture(workloadID)
	mockService.EXPECT().UpdateWorkloadSettings(gomock.Any(), workloadID, gomock.Any()).Return(replacement, nil)
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), workloadID, gomock.Any()).Return(replacement, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), workloadID).Return(updatedWorkload, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), workloadID).Return(updatedWorkload, nil) // post-apply refresh Read
}

type startReplacementMatcher struct {
	artifactID            string
	strategy              client.ReplacementStrategy
	warmupDurationMinutes int64
	keepOldVersionMinutes int64
}

func (m startReplacementMatcher) Matches(x any) bool {
	req, ok := x.(*client.StartReplacementRequest)
	if !ok || req == nil {
		return false
	}
	if req.ArtifactID != m.artifactID || req.Strategy != m.strategy {
		return false
	}
	if req.Config.WarmupDurationMinutes != m.warmupDurationMinutes {
		return false
	}
	if m.keepOldVersionMinutes != 0 && req.Config.KeepOldVersionMinutes != m.keepOldVersionMinutes {
		return false
	}
	if m.keepOldVersionMinutes == 0 && req.Config.KeepOldVersionMinutes != 0 {
		return false
	}
	return true
}

func (m startReplacementMatcher) String() string {
	return fmt.Sprintf(
		"StartReplacementRequest{artifactId=%q strategy=%q warmup=%d keepOld=%d}",
		m.artifactID, m.strategy, m.warmupDurationMinutes, m.keepOldVersionMinutes,
	)
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
    container_groups = [
      {
        replica_count    = %d
        resource_bundles = ["cpu.small"]
      }
    ]
  }
}
`, name, importance, artifactID, desc, replicaCount)
}

func workloadConfigWithReplacementPolicy(name, artifactID string, replicaCount, warmupMinutes, keepOldVersionMinutes int64) string {
	return fmt.Sprintf(`
resource "datarobot_workload" "test" {
  name        = %q
  importance  = "low"
  artifact_id = %q
  runtime = {
    container_groups = [
      {
        replica_count    = %d
        resource_bundles = ["cpu.small"]
      }
    ]
    replacement_policy = {
      warmup_minutes           = %d
      keep_old_version_minutes = %d
    }
  }
}
`, name, artifactID, replicaCount, warmupMinutes, keepOldVersionMinutes)
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
    container_groups = [
      {
        replica_count    = %d
        resource_bundles = [%q]
      }
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
    container_groups = [
      {
        resource_bundles = ["cpu.small"]
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
    ]
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
    container_groups = [
      {
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
    ]
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
            image_uri = "containous/whoami:latest"
            port      = 8080
            primary   = true
            entrypoint = ["/whoami", "--port", "8080"]
          }
        ]
      }
    ]
  }
}

resource "datarobot_workload" "test" {
  name        = %q
  importance  = %q
  artifact_id = datarobot_artifact.test_artifact.artifact_id
  %s
  runtime = {
    container_groups = [
      {
        replica_count    = %d
        resource_bundles = ["cpu.small"]
      }
    ]
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
		Runtime: client.WorkloadRuntime{
			ContainerGroups: []client.GroupRuntime{
				{Name: "default", ReplicaCount: replicaCount, ResourceBundles: []string{"cpu.small"}},
			},
		},
	}
}

func workloadFixtureWithResources(id, artifactID, name string, replicaCount *int64, endpoint *string, resourceBundles []string) *client.Workload {
	w := workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, replicaCount, endpoint)
	w.Runtime.ContainerGroups[0].ResourceBundles = resourceBundles
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
		Runtime: client.WorkloadRuntime{
			ContainerGroups: []client.GroupRuntime{
				{
					Name:            "default",
					ResourceBundles: []string{"cpu.small"},
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
			},
		},
	}
}

func TestWorkloadMissingResourceConfig(t *testing.T) {
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
				Config:      workloadConfigMissingResourceAllocation(artifactID),
				ExpectError: regexp.MustCompile("Missing resource configuration"),
			},
		},
	})
}

func workloadConfigMissingResourceAllocation(artifactID string) string {
	return fmt.Sprintf(`
resource "datarobot_workload" "test" {
  name        = "missing-resource-test"
  artifact_id = %q
  runtime = {
    container_groups = [
      {
        replica_count = 1
        containers = [
          { name = "main" }
        ]
      }
    ]
  }
}
`, artifactID)
}

func TestWorkloadEmptyContainers(t *testing.T) {
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
				Config:      workloadConfigEmptyContainers(artifactID),
				ExpectError: regexp.MustCompile("Missing containers"),
			},
		},
	})
}

func TestIntegrationWorkloadResourceBundlesSentinel(t *testing.T) {
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
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id

	// API injects resource_bundles even though the plan has none.
	apiWorkload := workloadFixtureWithResources(id, artifactID, name, &replicaCount, &endpoint, []string{"api-injected-bundle"})
	cpu := 1.0
	mem := int64(536870912)
	apiWorkload.Runtime.ContainerGroups[0].Containers = []client.ContainerOverride{
		{Name: "main", ResourceAllocation: &client.ResourceAllocation{CPU: &cpu, Memory: &mem}},
	}

	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(apiWorkload, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(apiWorkload, nil) // waitForRunning
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(apiWorkload, nil) // post-create Read

	// Destroy
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(apiWorkload, nil)
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(nil, client.NewNotFoundError("workload"))

	resourceName := "datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithResourceAllocation(name, artifactID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "runtime.container_groups.0.resource_bundles.0"),
				),
			},
		},
	})
}

func TestIntegrationWorkloadBundleSelectionPolicySentinel(t *testing.T) {
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
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id

	// API returns a different bundle_selection_policy than what the plan has.
	apiWorkload := workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)
	apiPolicy := "latency"
	apiWorkload.Runtime.ContainerGroups[0].BundleSelectionPolicy = &apiPolicy
	cpu := 1.0
	mem := int64(536870912)
	apiWorkload.Runtime.ContainerGroups[0].Containers = []client.ContainerOverride{
		{Name: "main", ResourceAllocation: &client.ResourceAllocation{CPU: &cpu, Memory: &mem}},
	}

	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(apiWorkload, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(apiWorkload, nil) // waitForRunning
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(apiWorkload, nil) // post-create Read

	// Destroy
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(apiWorkload, nil)
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(nil, client.NewNotFoundError("workload"))

	resourceName := "datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// User omits bundle_selection_policy; schema Default gives "availability".
				// API returns "latency". State must reflect the plan value, not the API value.
				Config: workloadConfigWithResourceAllocation(name, artifactID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.container_groups.0.bundle_selection_policy", "availability"),
				),
			},
		},
	})
}

func TestIntegrationWorkloadStringMemoryNormalization(t *testing.T) {
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
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id

	// "512Mi" normalizes to 536870912 bytes; API returns that as int64.
	cpu := 1.0
	mem := int64(536870912) // 512 * 1024^2
	apiWorkload := workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)
	apiWorkload.Runtime.ContainerGroups[0].Containers = []client.ContainerOverride{
		{Name: "main", ResourceAllocation: &client.ResourceAllocation{CPU: &cpu, Memory: &mem}},
	}

	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(apiWorkload, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(apiWorkload, nil) // waitForRunning
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(apiWorkload, nil) // post-create Read

	// Destroy
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(apiWorkload, nil)
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(nil, client.NewNotFoundError("workload"))

	resourceName := "datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// "512Mi" is sent as bytes to the API; the original string is preserved in state
				// because the sentinel restores it when the byte values match.
				Config: workloadConfigWithStringMemory(name, artifactID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName,
						"runtime.container_groups.0.containers.0.resource_allocation.memory",
						"512Mi"),
				),
			},
		},
	})
}

func workloadConfigWithResourceAllocation(name, artifactID string) string {
	return fmt.Sprintf(`
resource "datarobot_workload" "test" {
  name        = %q
  artifact_id = %q
  runtime = {
    container_groups = [
      {
        replica_count = 1
        containers = [
          {
            name = "main"
            resource_allocation = {
              cpu    = 1
              memory = 536870912
            }
          }
        ]
      }
    ]
  }
}
`, name, artifactID)
}

func workloadConfigWithStringMemory(name, artifactID string) string {
	return fmt.Sprintf(`
resource "datarobot_workload" "test" {
  name        = %q
  artifact_id = %q
  runtime = {
    container_groups = [
      {
        replica_count = 1
        containers = [
          {
            name = "main"
            resource_allocation = {
              cpu    = 1
              memory = "512Mi"
            }
          }
        ]
      }
    ]
  }
}
`, name, artifactID)
}

func workloadConfigEmptyContainers(artifactID string) string {
	return fmt.Sprintf(`
resource "datarobot_workload" "test" {
  name        = "empty-containers-test"
  artifact_id = %q
  runtime = {
    container_groups = [
      {
        replica_count = 1
        containers    = []
      }
    ]
  }
}
`, artifactID)
}

func workloadConfigWithMultipleGroups(artifactID string) string {
	return fmt.Sprintf(`
resource "datarobot_workload" "test" {
  name        = "multi-group-test"
  artifact_id = %q
  runtime = {
    container_groups = [
      { replica_count = 1 },
      { replica_count = 2 }
    ]
  }
}
`, artifactID)
}
