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

func TestAccWorkloadArtifactReplacement(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_workload.test"
	artifactResourceName := "datarobot_artifact.test_artifact"
	name := "workload-artifact-repl-" + nameSalt
	var initialWorkloadID, initialArtifactID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadAccConfigWithImage(name, "", "low", "containous/whoami:latest", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "artifact_id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					captureAttr(resourceName, "id", &initialWorkloadID),
					captureAttr(resourceName, "artifact_id", &initialArtifactID),
					checkWorkloadExistsInAPI(name, false),
				),
			},
			{
				Config: workloadAccConfigWithImage(name, "", "low", "containous/whoami:v1.5.0", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkWorkloadIDPreserved(&initialWorkloadID),
					checkWorkloadArtifactIDChanged(&initialArtifactID),
					resource.TestCheckResourceAttrPair(resourceName, "artifact_id", artifactResourceName, "artifact_id"),
					checkWorkloadExistsInAPI(name, false),
				),
			},
		},
	})
}

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

func TestAccWorkloadMetadataPreservesReplacementPolicy(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_workload.test"
	name := "workload-metadata-rp-" + nameSalt
	updatedName := "updated-" + name
	var initialID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadAccConfigWithReplacementPolicy(name, "", "low", 1, 5, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "runtime.replacement_policy.warmup_minutes", "5"),
					resource.TestCheckResourceAttr(resourceName, "runtime.replacement_policy.keep_old_version_minutes", "10"),
					captureAttr(resourceName, "id", &initialID),
					checkWorkloadExistsInAPI(name, false),
				),
			},
			{
				Config: workloadAccConfigWithReplacementPolicy(updatedName, "", "low", 1, 5, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckResourceAttr(resourceName, "runtime.replacement_policy.warmup_minutes", "5"),
					resource.TestCheckResourceAttr(resourceName, "runtime.replacement_policy.keep_old_version_minutes", "10"),
					checkWorkloadIDPreserved(&initialID),
					checkWorkloadExistsInAPI(updatedName, false),
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

	mockAPIKey(t)

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
	mockService.EXPECT().UpdateWorkloadMetadata(gomock.Any(), id, gomock.Any()).Return(updatedWorkload, nil)

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
					resource.TestCheckResourceAttr(resourceName, "type", "service"),
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

	mockAPIKey(t)

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
	mockService.EXPECT().UpdateWorkloadMetadata(gomock.Any(), id, updateDescriptionMatcher("")).Return(withoutDesc, nil)

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

	mockAPIKey(t)

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

	mockAPIKey(t)

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
				Config: workloadConfigWithReplacementPolicy(name, artifactID1, 5, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.replacement_policy.warmup_minutes", "5"),
					resource.TestCheckResourceAttr(resourceName, "runtime.replacement_policy.keep_old_version_minutes", "10"),
				),
			},
			{
				Config: workloadConfigWithReplacementPolicy(name, artifactID2, 5, 10),
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

	mockAPIKey(t)

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
				Config: workloadConfigWithReplacementPolicy(name, artifactID, 15, 0),
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

	mockAPIKey(t)

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
	mockService.EXPECT().UpdateWorkloadMetadata(gomock.Any(), id, gomock.Any()).Return(metadataWorkload, nil)
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

func TestIntegrationWorkloadUpdateMetadataPreservesReplacementPolicy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)

	id := uuid.NewString()
	artifactID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	updatedName := "updated-" + name
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id

	workload1 := workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)
	metadataWorkload := workloadFixture(id, artifactID, updatedName, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)

	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload1, nil)
	mockService.EXPECT().UpdateWorkloadMetadata(gomock.Any(), id, gomock.Any()).Return(metadataWorkload, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(metadataWorkload, nil)
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(nil, client.NewNotFoundError("workload"))

	resourceName := "datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithReplacementPolicy(name, artifactID, 5, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.replacement_policy.warmup_minutes", "5"),
					resource.TestCheckResourceAttr(resourceName, "runtime.replacement_policy.keep_old_version_minutes", "10"),
				),
			},
			{
				Config: workloadConfigWithReplacementPolicy(updatedName, artifactID, 5, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckResourceAttr(resourceName, "runtime.replacement_policy.warmup_minutes", "5"),
					resource.TestCheckResourceAttr(resourceName, "runtime.replacement_policy.keep_old_version_minutes", "10"),
				),
			},
		},
	})
}

func TestIntegrationWorkloadReplaceOnArtifactAndRuntimeChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)

	id := uuid.NewString()
	artifactID1 := uuid.NewString()
	artifactID2 := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount1 := int64(1)
	replicaCount2 := int64(3)
	endpoint := "https://workloads.example.com/" + id

	workload1 := workloadFixture(id, artifactID1, name, "", client.WorkloadImportanceLow, &replicaCount1, &endpoint)
	workload2 := workloadFixture(id, artifactID2, name, "", client.WorkloadImportanceLow, &replicaCount2, &endpoint)

	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload1, nil)

	replacement := workloadReplacementFixture(id)
	mockService.EXPECT().StartWorkloadReplacement(gomock.Any(), id, startReplacementWithRuntimeMatcher{
		artifactID:   artifactID2,
		replicaCount: replicaCount2,
	}).Return(replacement, nil)
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), id, gomock.Any()).Return(replacement, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload2, nil)
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
				Config: workloadConfigWithReplicas(name, "", "low", artifactID2, 3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "artifact_id", artifactID2),
					resource.TestCheckResourceAttr(resourceName, "runtime.container_groups.0.replica_count", "3"),
					checkWorkloadIDPreserved(&initialID),
				),
			},
		},
	})
}

func TestIntegrationWorkloadReplacementPollFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)

	id := uuid.NewString()
	artifactID1 := uuid.NewString()
	artifactID2 := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id

	workload1 := workloadFixture(id, artifactID1, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)

	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload1, nil)

	replacement := workloadReplacementFixture(id)
	mockService.EXPECT().StartWorkloadReplacement(gomock.Any(), id, gomock.Any()).Return(replacement, nil)
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), id, gomock.Any()).Return(nil, &client.ReplacementFailedError{
		Message: "candidate proton failed health checks",
	})

	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(nil, client.NewNotFoundError("workload"))

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID1, 1),
			},
			{
				Config:      workloadConfigWithReplicas(name, "", "low", artifactID2, 1),
				ExpectError: regexp.MustCompile("Workload replacement failed"),
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

	mockAPIKey(t)

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

	// Step 2: In-place replacement via settings endpoint (runtime-only)
	replacement := workloadReplacementFixture(id1)
	mockService.EXPECT().UpdateWorkloadSettings(gomock.Any(), id1, updateWorkloadSettingsReplicaMatcher(3)).Return(replacement, nil)
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), id1, gomock.Any()).Return(replacement, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload2, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id1).Return(workload2, nil)

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

	mockAPIKey(t)

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

	mockAPIKey(t)

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
					resource.TestCheckResourceAttr(resourceName, "runtime.container_groups.0.autoscaling.min_replica_count", "1"),
					captureAttr(resourceName, "id", &initialID),
					checkWorkloadExistsInAPI(name, true),
				),
			},
			{
				Config: workloadConfigWithAutoscaling(name, "", "low", artifactID, 2, 5, 70.0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.container_groups.0.autoscaling.min_replica_count", "2"),
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

	mockAPIKey(t)

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

	mockAPIKey(t)

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

func TestWorkloadCPUScalingRequiresNonZeroMinReplicas(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)

	artifactID := uuid.NewString()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// min_replica_count = 0 (scale to zero) is invalid with cpuAverageUtilization.
				Config:      workloadConfigWithAutoscaling("cpu-min-zero-test", "", "low", artifactID, 0, 3, 70),
				ExpectError: regexp.MustCompile("min_replica_count must be greater than 0"),
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

	mockAPIKey(t)

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

func checkWorkloadArtifactIDChanged(initialArtifactID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const rn = "datarobot_workload.test"
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource %s not found in state", rn)
		}
		newArtifactID := rs.Primary.Attributes["artifact_id"]
		if *initialArtifactID != "" && newArtifactID == *initialArtifactID {
			return fmt.Errorf("workload artifact_id unchanged after artifact spec update: still %q", newArtifactID)
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

type startReplacementWithRuntimeMatcher struct {
	artifactID   string
	replicaCount int64
}

func (m startReplacementWithRuntimeMatcher) Matches(x any) bool {
	req, ok := x.(*client.StartReplacementRequest)
	if !ok || req == nil {
		return false
	}
	if req.ArtifactID != m.artifactID || req.Strategy != client.ReplacementStrategyRolling {
		return false
	}
	if req.Runtime == nil || len(req.Runtime.ContainerGroups) == 0 {
		return false
	}
	replicaCount := req.Runtime.ContainerGroups[0].ReplicaCount
	return replicaCount != nil && *replicaCount == m.replicaCount
}

func (m startReplacementWithRuntimeMatcher) String() string {
	return fmt.Sprintf(
		"StartReplacementRequest{artifactId=%q strategy=%q replicaCount=%d}",
		m.artifactID, client.ReplacementStrategyRolling, m.replicaCount,
	)
}

type updateWorkloadSettingsReplicaMatcher int64

func (m updateWorkloadSettingsReplicaMatcher) Matches(x any) bool {
	req, ok := x.(*client.UpdateWorkloadSettingsRequest)
	if !ok || req == nil {
		return false
	}
	if len(req.Runtime.ContainerGroups) == 0 {
		return false
	}
	replicaCount := req.Runtime.ContainerGroups[0].ReplicaCount
	return replicaCount != nil && *replicaCount == int64(m)
}

func (m updateWorkloadSettingsReplicaMatcher) String() string {
	return fmt.Sprintf("UpdateWorkloadSettingsRequest with replica_count=%d", int64(m))
}

// ─── config helpers ────────────────────────────────────────────────────────────

func workloadMockConfig(cfg string) string {
	return testProviderConfigBlock() + "\n" + cfg
}

func workloadConfigWithReplicas(name, description, importance, artifactID string, replicaCount int64) string {
	desc := ""
	if description != "" {
		desc = fmt.Sprintf("description = %q", description)
	}
	return workloadMockConfig(fmt.Sprintf(`
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
`, name, importance, artifactID, desc, replicaCount))
}

func workloadConfigWithReplacementPolicy(name, artifactID string, warmupMinutes, keepOldVersionMinutes int64) string {
	return workloadMockConfig(fmt.Sprintf(`
resource "datarobot_workload" "test" {
  name        = %q
  importance  = "low"
  artifact_id = %q
  runtime = {
    container_groups = [
      {
        replica_count    = 1
        resource_bundles = ["cpu.small"]
      }
    ]
    replacement_policy = {
      warmup_minutes           = %d
      keep_old_version_minutes = %d
    }
  }
}
`, name, artifactID, warmupMinutes, keepOldVersionMinutes))
}

func workloadConfigWithReplicasAndResources(name, description, importance, artifactID string, replicaCount int64, resourceBundleID string) string {
	desc := ""
	if description != "" {
		desc = fmt.Sprintf("description = %q", description)
	}
	return workloadMockConfig(fmt.Sprintf(`
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
`, name, importance, artifactID, desc, replicaCount, resourceBundleID))
}

func workloadConfigWithAutoscaling(name, description, importance, artifactID string, minReplicaCount, maxReplicaCount int64, target float64) string {
	desc := ""
	if description != "" {
		desc = fmt.Sprintf("description = %q", description)
	}
	return workloadMockConfig(fmt.Sprintf(`
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
          enabled           = true
          min_replica_count = %d
          max_replica_count = %d
          policies = [
            {
              scaling_metric = "cpuAverageUtilization"
              target         = %g
            }
          ]
        }
      }
    ]
  }
}
`, name, importance, artifactID, desc, minReplicaCount, maxReplicaCount, target))
}

func workloadConfigConflictingRuntime(artifactID string) string {
	return workloadMockConfig(fmt.Sprintf(`
resource "datarobot_workload" "test" {
  name        = "conflict-test"
  artifact_id = %q
  runtime = {
    container_groups = [
      {
        replica_count = 2
        autoscaling = {
          enabled           = true
          min_replica_count = 1
          max_replica_count = 4
          policies = [
            {
              scaling_metric = "cpuAverageUtilization"
              target         = 50
            }
          ]
        }
      }
    ]
  }
}
`, artifactID))
}

func workloadAccConfig(name, description, importance string, replicaCount int64) string {
	return workloadAccConfigWithImage(name, description, importance, "containous/whoami:latest", replicaCount)
}

func workloadAccConfigWithImage(name, description, importance, imageURI string, replicaCount int64) string {
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
            image_uri = %q
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
`, artifactName, imageURI, name, importance, desc, replicaCount)
}

func workloadAccConfigWithReplacementPolicy(name, description, importance string, replicaCount, warmupMinutes, keepOldVersionMinutes int64) string {
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
            name       = "main"
            image_uri  = "containous/whoami:latest"
            port       = 8080
            primary    = true
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
    replacement_policy = {
      warmup_minutes           = %d
      keep_old_version_minutes = %d
    }
  }
}
`, artifactName, name, importance, desc, replicaCount, warmupMinutes, keepOldVersionMinutes)
}

// ─── fixture helpers ───────────────────────────────────────────────────────────

func workloadFixture(id, artifactID, name, description string, importance client.WorkloadImportance, replicaCount *int64, endpoint *string) *client.Workload {
	return &client.Workload{
		ID:          id,
		Name:        name,
		Description: description,
		Status:      client.ProtonStatusRunning,
		Importance:  importance,
		Type:        client.ArtifactTypeService,
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

func workloadFixtureWithAutoscaling(id, artifactID, name string, endpoint *string, minReplicaCount, maxReplicaCount int64, target float64) *client.Workload {
	enabled := true
	return &client.Workload{
		ID:         id,
		Name:       name,
		Status:     client.ProtonStatusRunning,
		Importance: client.WorkloadImportanceLow,
		Type:       client.ArtifactTypeService,
		ArtifactID: &artifactID,
		Endpoint:   endpoint,
		Runtime: client.WorkloadRuntime{
			ContainerGroups: []client.GroupRuntime{
				{
					Name:            "default",
					ResourceBundles: []string{"cpu.small"},
					Autoscaling: &client.AutoscalingProperties{
						Enabled:         &enabled,
						MinReplicaCount: minReplicaCount,
						MaxReplicaCount: maxReplicaCount,
						Policies: []client.AutoscalingPolicy{
							{
								ScalingMetric: "cpuAverageUtilization",
								Target:        target,
							},
						},
					},
				},
			},
		},
	}
}

// workloadConfigScalingUnspecified is a workload whose container group sets
// neither replica_count nor autoscaling (only resource_bundles) — the case where
// the backend supplies a cluster-dependent scaling default.
func workloadConfigScalingUnspecified(name, artifactID string) string {
	return workloadMockConfig(fmt.Sprintf(`
resource "datarobot_workload" "test" {
  name        = %q
  importance  = "low"
  artifact_id = %q
  runtime = {
    container_groups = [
      {
        resource_bundles = ["cpu.small"]
      }
    ]
  }
}
`, name, artifactID))
}

// TestIntegrationWorkloadNoDriftWhenScalingUnspecified guards the ModifyPlan
// drift fix: when the config specifies neither replica_count nor autoscaling and
// the backend fills in an autoscaling block, subsequent plans must be empty (no
// perpetual drift). The framework runs an automatic empty-plan check after the
// apply step; without ModifyPlan the backend-populated autoscaling would diff
// against the empty config and fail it.
func TestIntegrationWorkloadNoDriftWhenScalingUnspecified(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)

	id := uuid.NewString()
	artifactID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	endpoint := "https://workloads.example.com/" + id

	// The user configured neither replica_count nor autoscaling; the backend
	// responds with a scale-to-zero autoscaling block (min=0, max=1).
	workload := workloadFixtureWithAutoscaling(id, artifactID, name, &endpoint, 0, 1, 1000.0)

	deleted := false
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).DoAndReturn(
		func(_ context.Context, _ string) (*client.Workload, error) {
			if deleted {
				return nil, client.NewNotFoundError("workload")
			}
			return workload, nil
		}).AnyTimes()
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).DoAndReturn(
		func(_ context.Context, _ string) error {
			deleted = true
			return nil
		})

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigScalingUnspecified(name, artifactID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("datarobot_workload.test", "id"),
					// The backend-supplied scaling is kept out of state (sentinel), so
					// it matches the empty config. No-drift is asserted automatically
					// by the framework's post-apply empty-plan check.
					resource.TestCheckNoResourceAttr("datarobot_workload.test", "runtime.container_groups.0.autoscaling"),
					resource.TestCheckNoResourceAttr("datarobot_workload.test", "runtime.container_groups.0.replica_count"),
				),
			},
		},
	})
}

func TestWorkloadMissingResourceConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)

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
	return workloadMockConfig(fmt.Sprintf(`
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
`, artifactID))
}

func TestWorkloadEmptyContainers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)

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

	mockAPIKey(t)

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

	mockAPIKey(t)

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

	mockAPIKey(t)

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
	return workloadMockConfig(fmt.Sprintf(`
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
`, name, artifactID))
}

func workloadConfigWithStringMemory(name, artifactID string) string {
	return workloadMockConfig(fmt.Sprintf(`
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
`, name, artifactID))
}

func workloadConfigEmptyContainers(artifactID string) string {
	return workloadMockConfig(fmt.Sprintf(`
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
`, artifactID))
}

func workloadConfigWithMultipleGroups(artifactID string) string {
	return workloadMockConfig(fmt.Sprintf(`
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
`, artifactID))
}

func TestLoadWorkloadIntoModelType(t *testing.T) {
	t.Parallel()

	id := "wl-1"
	artifactID := "art-1"
	endpoint := "https://example.com/wl-1"
	workload := workloadFixture(id, artifactID, "agent-wl", "", client.WorkloadImportanceLow, nil, &endpoint)
	workload.Type = client.ArtifactTypeAgent

	var data WorkloadResourceModel
	loadWorkloadIntoModel(workload, &data)

	if data.Type.ValueString() != string(client.ArtifactTypeAgent) {
		t.Fatalf("Type = %q, want %q", data.Type.ValueString(), client.ArtifactTypeAgent)
	}

	workload.Type = ""
	loadWorkloadIntoModel(workload, &data)
	if !data.Type.IsNull() {
		t.Fatalf("empty Type = %v, want null", data.Type)
	}
}
