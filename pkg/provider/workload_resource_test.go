package provider

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
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
	testAccWorkloadResource(t)
}

func TestAccWorkloadFromBuiltArtifact(t *testing.T) {
	t.Parallel()
	testAccWorkloadFromBuiltArtifact(t, false)
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
	workload2.Type = client.ArtifactTypeAgent

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
					resource.TestCheckResourceAttr(resourceName, "type", "service"),
					captureAttr(resourceName, "id", &initialID),
					checkWorkloadExistsInAPI(name, true),
				),
			},
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID2, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "artifact_id", artifactID2),
					resource.TestCheckResourceAttr(resourceName, "type", "agent"),
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
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), id1, waitExpectsArtifact(artifactID2)).Return(replacement, nil)
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
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), id1, waitExpectsArtifact(artifactID)).Return(replacement, nil)
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
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), id, waitExpectsArtifact(artifactID2)).Return(replacement, nil)
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
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), id, waitExpectsArtifact(artifactID2)).Return(nil, &client.ReplacementFailedError{
		Message: "candidate proton failed health checks",
	})
	mockService.EXPECT().BaseURL().Return("https://app.datarobot.com").AnyTimes()

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
				Config: workloadConfigWithReplicas(name, "", "low", artifactID2, 1),
				// The diagnostic has to carry the reason and where to read the
				// rest of it; a bare "replacement failed" sends nobody anywhere.
				ExpectError: regexp.MustCompile(`(?s)Workload replacement failed.*candidate proton failed health checks.*` +
					`console-nextgen/workloads/` + id + `/activity-log/otel-logs`),
			},
		},
	})
}

// mockArtifactRollout sets up a create of workload id on artifactID1 and an
// apply that rolls it to artifactID2, where the wait is asked to wait for the
// new version to serve (or not, per serving), answers waitErr, and the workload
// then reads as afterRollout. GetWorkload follows the platform rather than
// counting calls, so the steps can refresh as often as Terraform likes.
func mockArtifactRollout(
	t *testing.T,
	mockService *mock_client.MockService,
	id, artifactID2 string,
	workload1, afterRollout *client.Workload,
	serving bool,
	waitErr error,
) {
	t.Helper()
	current := workload1
	deleted := false
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).DoAndReturn(func(context.Context, string) (*client.Workload, error) {
		if deleted {
			return nil, client.NewNotFoundError("workload")
		}
		return current, nil
	}).AnyTimes()
	replacement := workloadReplacementFixture(id)
	mockService.EXPECT().StartWorkloadReplacement(gomock.Any(), id, gomock.Any()).Return(replacement, nil)
	// The options are checked inside, with t.Errorf: a matcher that did not match
	// would leave the call unexpected, which hangs resource.Test instead of failing.
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), id, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, opts *client.WaitForWorkloadReplacementOptions) (*client.WorkloadReplacement, error) {
			want := waitExpectsServing{artifactID: artifactID2, serving: serving}
			if !want.Matches(opts) {
				t.Errorf("WaitForWorkloadReplacement got %+v, want %s", opts, want)
			}
			current = afterRollout
			return replacement, waitErr
		})
	mockService.EXPECT().BaseURL().Return("https://app.datarobot.com").AnyTimes()
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).DoAndReturn(func(context.Context, string) error {
		deleted = true
		return nil
	})
}

func TestIntegrationWorkloadInterruptedWaitAfterPromotionKeepsTheRollout(t *testing.T) {
	// The rollout landed; only the wait for the previous version to stop taking
	// requests was cut short. State must record the new artifact, or every later
	// apply asks for a rollout to the artifact the workload already runs.
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
	workload2 := workloadFixture(id, artifactID2, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)

	mockArtifactRollout(t, mockService, id, artifactID2, workload1, workload2, true, &client.ServingUnconfirmedError{
		WorkloadID: id,
		Err:        context.Canceled,
	})

	resourceName := "datarobot_workload.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID1, 1),
			},
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID2, 1),
				Check:  resource.TestCheckResourceAttr(resourceName, "artifact_id", artifactID2),
			},
		},
	})
}

func TestIntegrationWorkloadVersionThatStopsAfterPromotionIsRecordedAndReported(t *testing.T) {
	// The new version was promoted and then errored. The apply fails, and the
	// state still records the artifact the workload now runs, so the same
	// configuration plans no further rollout to it.
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
	errored := workloadFixture(id, artifactID2, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)
	errored.Status = client.ProtonStatusErrored

	mockArtifactRollout(t, mockService, id, artifactID2, workload1, errored, true, nil)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID1, 1),
			},
			{
				Config: workloadConfigWithReplicas(name, "", "low", artifactID2, 1),
				ExpectError: regexp.MustCompile(`(?s)Workload replacement failed.*promoted, but the Workload now reports status "errored".*` +
					`console-nextgen/workloads/` + id + `/activity-log/otel-logs`),
			},
			{
				Config:   workloadConfigWithReplicas(name, "", "low", artifactID2, 1),
				PlanOnly: true,
			},
		},
	})
}

func TestIntegrationWorkloadArtifactChangeOnEnclaveSkipsTheGatewayWait(t *testing.T) {
	// A workload that asks for an Enclave is not reached through the prediction
	// gateway, so a new artifact does not wait for the gateway's cached route,
	// even on a platform that does not report placements.
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
	useCaseID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount := int64(1)
	endpoint := "https://enclave.example.com/workloads/" + id
	workload1 := workloadFixture(id, artifactID1, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)
	workload2 := workloadFixture(id, artifactID2, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)

	mockArtifactRollout(t, mockService, id, artifactID2, workload1, workload2, false, nil)

	resourceName := "datarobot_workload.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithPlacement(name, artifactID1, useCaseID, "availability", ""),
			},
			{
				Config: workloadConfigWithPlacement(name, artifactID2, useCaseID, "availability", ""),
				Check:  resource.TestCheckResourceAttr(resourceName, "artifact_id", artifactID2),
			},
		},
	})
}

func TestWorkloadTargetsEnclave(t *testing.T) {
	enclaves := func(names ...string) types.List {
		values := make([]attr.Value, 0, len(names))
		for _, name := range names {
			values = append(values, types.StringValue(name))
		}
		return types.ListValueMust(types.StringType, values)
	}
	cases := []struct {
		name    string
		runtime WorkloadRuntimeModel
		want    bool
	}{
		{name: "no placement", runtime: WorkloadRuntimeModel{EnclaveSelectionPolicy: types.StringNull(), Enclaves: types.ListNull(types.StringType)}, want: false},
		{name: "empty pin", runtime: WorkloadRuntimeModel{EnclaveSelectionPolicy: types.StringNull(), Enclaves: enclaves()}, want: false},
		{name: "policy", runtime: WorkloadRuntimeModel{EnclaveSelectionPolicy: types.StringValue("availability"), Enclaves: types.ListNull(types.StringType)}, want: true},
		{name: "pin implies manual", runtime: WorkloadRuntimeModel{EnclaveSelectionPolicy: types.StringNull(), Enclaves: enclaves("finance")}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := workloadTargetsEnclave(tc.runtime); got != tc.want {
				t.Fatalf("workloadTargetsEnclave() = %v, want %v", got, tc.want)
			}
		})
	}
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
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), id1, waitExpectsServing{artifactID: artifactID, serving: false}).Return(replacement, nil)
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
	mockService.EXPECT().ListUseCasesForEntity(gomock.Any(), "workload", id).Return(nil, false, nil)
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

// An imported Workload on an Enclave has to come with its Use Case: a configuration with the
// placement and no `use_case_id` is refused, and adding the real one must not plan a change.
func TestIntegrationWorkloadImportReadsUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service { return mockService })()
	mockAPIKey(t)

	id, artifactID, useCaseID, name := uuid.NewString(), uuid.NewString(), uuid.NewString(), "workload-"+uuid.NewString()[:8]
	replicaCount := int64(1)
	workload := workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, nil)
	workload.Runtime.EnclaveSelectionPolicy = enclavePolicyPtr(client.EnclaveSelectionPolicyAvailability)
	expectWorkloadsCreatedAndDeleted(mockService, workload)
	mockService.EXPECT().ListUseCasesForEntity(gomock.Any(), "workload", id).
		Return([]client.UseCaseResponse{{ID: useCaseID}}, false, nil)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: workloadConfigWithPlacement(name, artifactID, useCaseID, "availability", "")},
			{
				ResourceName:      "datarobot_workload.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// Imported while linked to several Use Cases, a placed Workload has no use_case_id in state.
// Writing one down then names a link it already has: it is recorded without linking again, and
// with a placement change it rolls out in place instead of replacing the Workload. Removing the
// placement is refused until use_case_id is in state.
func TestIntegrationWorkloadAdoptsUseCaseAfterImport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service { return mockService })()
	mockAPIKey(t)

	id, artifactID, name := uuid.NewString(), uuid.NewString(), "workload-"+uuid.NewString()[:8]
	firstUseCase, secondUseCase := uuid.NewString(), uuid.NewString()
	replicaCount := int64(1)
	workload := workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, nil)
	workload.Runtime.EnclaveSelectionPolicy = enclavePolicyPtr(client.EnclaveSelectionPolicyAvailability)

	// Created outside Terraform: served until deleted, rolled out once with the pin.
	deleted := false
	mockService.EXPECT().GetWorkload(gomock.Any(), id).DoAndReturn(
		func(context.Context, string) (*client.Workload, error) {
			if deleted {
				return nil, client.NewNotFoundError("workload")
			}
			return workload, nil
		}).AnyTimes()
	replacement := workloadReplacementFixture(id)
	mockService.EXPECT().UpdateWorkloadSettings(gomock.Any(), id, settingsPlacementMatcher{placementMatcher{
		policy: enclavePolicyPtr(client.EnclaveSelectionPolicyManual), enclaves: []string{"finance"}, explicit: true,
	}}).Return(replacement, nil)
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), id, waitExpectsArtifact(artifactID)).Return(replacement, nil)
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).DoAndReturn(
		func(context.Context, string) error {
			deleted = true
			return nil
		})
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).DoAndReturn(
		func(context.Context, *client.CreateWorkloadRequest) (*client.Workload, error) {
			t.Errorf("Workload replaced instead of updated in place")
			return nil, fmt.Errorf("replaced")
		}).AnyTimes()
	mockService.EXPECT().ListUseCasesForEntity(gomock.Any(), "workload", id).
		Return([]client.UseCaseResponse{{ID: firstUseCase}, {ID: secondUseCase}}, false, nil).AnyTimes()
	mockService.EXPECT().AddEntityToUseCase(gomock.Any(), gomock.Any(), "workload", id).DoAndReturn(
		func(_ context.Context, useCaseID, _, _ string) error {
			t.Errorf("linked again to Use Case %s, which the Workload is already linked to", useCaseID)
			return nil
		}).AnyTimes()
	mockService.EXPECT().RemoveEntityFromUseCase(gomock.Any(), gomock.Any(), "workload", id).DoAndReturn(
		func(_ context.Context, useCaseID, _, _ string) error {
			t.Errorf("unlinked from Use Case %s", useCaseID)
			return nil
		}).AnyTimes()

	resourceName := "datarobot_workload.test"
	notRecorded := regexp.MustCompile("its Use Case is not in state")
	initialID := id

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             workloadConfigWithPlacement(name, artifactID, firstUseCase, "availability", ""),
				ResourceName:       resourceName,
				ImportState:        true,
				ImportStateId:      id,
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if got := states[0].Attributes["use_case_id"]; got != "" {
						return fmt.Errorf("use_case_id = %q after import, want it empty", got)
					}
					return nil
				},
			},
			{
				Config:      workloadConfigWithPlacement(name, artifactID, "", "", ""),
				ExpectError: notRecorded,
			},
			{
				Config:      workloadConfigWithPlacement(name, artifactID, firstUseCase, "", ""),
				ExpectError: notRecorded,
			},
			{
				Config:           workloadConfigWithPlacement(name, artifactID, firstUseCase, "", "finance"),
				ConfigPlanChecks: expectInPlacePlacementChange(resourceName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "use_case_id", firstUseCase),
					checkWorkloadIDPreserved(&initialID),
				),
			},
		},
	})
}

func TestIntegrationWorkloadImportUseCaseID(t *testing.T) {
	first, second := uuid.NewString(), uuid.NewString()
	tests := []struct {
		name     string
		useCases []client.UseCaseResponse
		more     bool
		err      error
		want     types.String
		warnings []string
		detail   []string
	}{
		{name: "not linked", want: types.StringNull()},
		{name: "one Use Case", useCases: []client.UseCaseResponse{{ID: first}}, want: types.StringValue(first)},
		{
			name:     "several Use Cases",
			useCases: []client.UseCaseResponse{{ID: first}, {ID: second}},
			want:     types.StringNull(),
			warnings: []string{"Workload is linked to several Use Cases"},
			detail:   []string{first, second},
		},
		{
			name:     "more than one page",
			useCases: []client.UseCaseResponse{{ID: first}},
			more:     true,
			want:     types.StringNull(),
			warnings: []string{"Workload is linked to several Use Cases"},
			detail:   []string{first, "more Use Cases than the first 1 the API lists"},
		},
		{
			name:     "lookup refused",
			err:      fmt.Errorf("422 Unprocessable Entity"),
			want:     types.StringNull(),
			warnings: []string{"Could not read the Use Case of the Workload"},
			detail:   []string{"422 Unprocessable Entity"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockService := mock_client.NewMockService(ctrl)
			mockService.EXPECT().ListUseCasesForEntity(gomock.Any(), "workload", "wl").Return(tt.useCases, tt.more, tt.err)

			var diags diag.Diagnostics
			got := importWorkloadUseCaseID(context.Background(), mockService, "wl", &diags)

			if !got.Equal(tt.want) {
				t.Errorf("use_case_id = %s, want %s", got, tt.want)
			}
			if diags.HasError() {
				t.Fatalf("unexpected error: %v", diags)
			}
			if warnings := diagWarningSummaries(diags); !slices.Equal(warnings, tt.warnings) {
				t.Fatalf("warnings = %v, want %v", warnings, tt.warnings)
			}
			for _, want := range tt.detail {
				if detail := diags.Warnings()[0].Detail(); !strings.Contains(detail, want) {
					t.Errorf("warning detail lacks %q: %s", want, detail)
				}
			}
		})
	}
}

func TestIntegrationWorkloadRelinkUseCase(t *testing.T) {
	a, b := uuid.NewString(), uuid.NewString()
	placed := WorkloadRuntimeModel{EnclaveSelectionPolicy: types.StringValue("availability")}
	listed := func(ids ...string) []client.UseCaseResponse {
		out := make([]client.UseCaseResponse, len(ids))
		for i, id := range ids {
			out[i] = client.UseCaseResponse{ID: id}
		}
		return out
	}
	tests := []struct {
		name     string
		state    types.String
		runtime  WorkloadRuntimeModel
		planned  types.String
		lookup   bool // whether the lookup is called
		listed   []client.UseCaseResponse
		more     bool
		err      error
		links    []string
		unlinks  []string
		warnings []string
	}{
		{name: "to a new Use Case", state: types.StringValue(a), planned: types.StringValue(b), lookup: true,
			listed: listed(a), links: []string{b}, unlinks: []string{a}},
		{name: "to a Use Case already linked", state: types.StringValue(a), planned: types.StringValue(b), lookup: true,
			listed: listed(a, b), unlinks: []string{a}},
		{name: "unlinked", state: types.StringValue(a), planned: types.StringNull(), unlinks: []string{a}},
		{name: "first link of a Hub Workload", state: types.StringNull(), planned: types.StringValue(b), lookup: true,
			links: []string{b}},
		{name: "adopted after import", state: types.StringNull(), runtime: placed, planned: types.StringValue(b), lookup: true,
			listed: listed(a, b)},
		{name: "adopted but not linked", state: types.StringNull(), runtime: placed, planned: types.StringValue(b), lookup: true,
			listed: listed(a), links: []string{b}, warnings: []string{"Workload linked to another Use Case"}},
		{name: "adopted, lookup failed", state: types.StringNull(), runtime: placed, planned: types.StringValue(b), lookup: true,
			err: fmt.Errorf("502 Bad Gateway"), links: []string{b}},
		{name: "adopted, not on the first page", state: types.StringNull(), runtime: placed, planned: types.StringValue(b), lookup: true,
			listed: listed(a), more: true, links: []string{b}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockService := mock_client.NewMockService(ctrl)
			if tt.lookup {
				mockService.EXPECT().ListUseCasesForEntity(gomock.Any(), "workload", "wl").Return(tt.listed, tt.more, tt.err)
			}
			var links, unlinks []string
			mockService.EXPECT().AddEntityToUseCase(gomock.Any(), gomock.Any(), "workload", "wl").DoAndReturn(
				func(_ context.Context, useCaseID, _, _ string) error {
					links = append(links, useCaseID)
					return nil
				}).AnyTimes()
			mockService.EXPECT().RemoveEntityFromUseCase(gomock.Any(), gomock.Any(), "workload", "wl").DoAndReturn(
				func(_ context.Context, useCaseID, _, _ string) error {
					unlinks = append(unlinks, useCaseID)
					return nil
				}).AnyTimes()

			var diags diag.Diagnostics
			state := WorkloadResourceModel{UseCaseID: tt.state, Runtime: tt.runtime}
			if err := relinkWorkloadUseCase(context.Background(), mockService, "wl", state, tt.planned, &diags); err != nil {
				t.Fatalf("relinkWorkloadUseCase: %v", err)
			}
			if !slices.Equal(links, tt.links) || !slices.Equal(unlinks, tt.unlinks) {
				t.Errorf("linked %v and unlinked %v, want %v and %v", links, unlinks, tt.links, tt.unlinks)
			}
			if warnings := diagWarningSummaries(diags); !slices.Equal(warnings, tt.warnings) {
				t.Errorf("warnings = %v, want %v", warnings, tt.warnings)
			}
		})
	}
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
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), workloadID,
		waitExpectsArtifact(*updatedWorkload.ArtifactID)).Return(replacement, nil)
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

// waitExpectsArtifact asserts the wait is told which artifact the rollout was
// asked to promote. Without it the wait cannot tell a promoted rollout from one
// the platform abandoned, so the argument being passed is part of the contract.
type waitExpectsArtifact string

func (m waitExpectsArtifact) Matches(x any) bool {
	opts, ok := x.(*client.WaitForWorkloadReplacementOptions)
	return ok && opts != nil && opts.ExpectedArtifactID == string(m)
}

func (m waitExpectsArtifact) String() string {
	return fmt.Sprintf("WaitForWorkloadReplacementOptions{expectedArtifactId=%q}", string(m))
}

// waitExpectsServing is waitExpectsArtifact plus whether the wait goes on past
// promotion until the new version takes the workload's requests, which only a
// rollout to a new artifact asks for.
type waitExpectsServing struct {
	artifactID string
	serving    bool
}

func (m waitExpectsServing) Matches(x any) bool {
	opts, ok := x.(*client.WaitForWorkloadReplacementOptions)
	return ok && opts != nil && opts.ExpectedArtifactID == m.artifactID && opts.WaitUntilServing == m.serving
}

func (m waitExpectsServing) String() string {
	return fmt.Sprintf("WaitForWorkloadReplacementOptions{expectedArtifactId=%q, waitUntilServing=%t}", m.artifactID, m.serving)
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

func testAccWorkloadResource(t *testing.T) {
	t.Helper()

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

func testAccWorkloadFromBuiltArtifact(t *testing.T, isMock bool) {
	t.Helper()

	sourceDir := writeWorkloadACCSourceTree(t)

	const (
		artifactResourceName = "datarobot_artifact.app"
		workloadResourceName = "datarobot_workload.test"
	)
	artifactName := "acc-built-artifact-" + nameSalt
	workloadName := "acc-built-workload-" + nameSalt
	var lastArtifactID, lastWorkloadID string

	preCheck := func() { testAccPreCheck(t) }
	if !isMock {
		preCheck = func() { testAccArtifactBuildPreCheck(t) }
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               isMock,
		PreCheck:                 preCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             checkWorkloadFromBuiltArtifactDestroyed(&lastWorkloadID, &lastArtifactID, isMock),
		Steps: []resource.TestStep{
			{
				Config: workloadFromBuiltArtifactAccConfig(artifactName, workloadName, sourceDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					append([]resource.TestCheckFunc{
						resource.TestCheckResourceAttr(artifactResourceName, "status", "locked"),
						resource.TestCheckResourceAttrSet(artifactResourceName, "artifact_id"),
						resource.TestCheckResourceAttrSet(workloadResourceName, "id"),
						resource.TestCheckResourceAttrSet(workloadResourceName, "endpoint"),
						resource.TestCheckResourceAttr(workloadResourceName, "name", workloadName),
						resource.TestCheckResourceAttr(workloadResourceName, "status", "running"),
						resource.TestCheckResourceAttrSet(workloadResourceName, "artifact_id"),
						checkWorkloadArtifactIDMatches(artifactResourceName, workloadResourceName, isMock),
						checkWorkloadRunningInAPI(workloadResourceName, isMock),
						checkWorkloadExistsInAPI(workloadName, isMock),
						captureAttr(workloadResourceName, "id", &lastWorkloadID),
						captureAttr(artifactResourceName, "artifact_id", &lastArtifactID),
					}, artifactBuildCheckFuncs(artifactResourceName, isMock, true)...)...,
				),
			},
		},
	})
}

func writeWorkloadACCSourceTree(t *testing.T) string {
	t.Helper()

	return writeArtifactSourceTree(t, map[string]string{
		"app.py": `from fastapi import FastAPI

app = FastAPI(title="Workload ACC Test")


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok"}
`,
		"requirements.txt": "fastapi>=0.115.0\nuvicorn>=0.32.0\n",
		"Dockerfile": `FROM python:3.12-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app.py .

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8080"]
`,
	})
}

func checkWorkloadRunningInAPI(workloadResourceName string, isMock bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[workloadResourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", workloadResourceName)
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
		if workload.Status != client.ProtonStatusRunning {
			return fmt.Errorf("expected workload status running in API, got %q", workload.Status)
		}

		return nil
	}
}

func checkWorkloadDestroyedFromAPI(lastWorkloadID *string, isMock bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if isMock || *lastWorkloadID == "" {
			return nil
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("provider not found")
		}
		p.service = NewService(cl)

		_, err := p.service.GetWorkload(context.Background(), *lastWorkloadID)
		if err == nil {
			return fmt.Errorf("workload %s still exists after destroy", *lastWorkloadID)
		}
		if _, ok := err.(*client.NotFoundError); !ok {
			return fmt.Errorf("unexpected error checking workload %s after destroy: %w", *lastWorkloadID, err)
		}

		return nil
	}
}

func checkWorkloadFromBuiltArtifactDestroyed(lastWorkloadID, lastArtifactID *string, isMock bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if err := checkWorkloadDestroyedFromAPI(lastWorkloadID, isMock)(s); err != nil {
			return err
		}
		return checkArtifactRepoDestroyedFromAPI(lastArtifactID, isMock)(s)
	}
}

func workloadFromBuiltArtifactAccConfig(artifactName, workloadName, sourceDir string) string {
	return fmt.Sprintf(`
resource "datarobot_artifact" "app" {
  name   = %q
  status = "locked"

  source = {
    dir = %q
  }

  spec = {
    container_groups = [{
      containers = [{
        name    = "main"
        primary = true
        port    = 8080

        image_build_config = {
          dockerfile = {
            source = "provided"
          }
        }
      }]
    }]
  }
}

resource "datarobot_workload" "test" {
  name        = %q
  importance  = "low"
  artifact_id = datarobot_artifact.app.artifact_id

  runtime = {
    container_groups = [{
      replica_count    = 1
      resource_bundles = ["cpu.small"]
    }]
  }

  depends_on = [datarobot_artifact.app]
}
`, artifactName, sourceDir, workloadName)
}

func checkWorkloadArtifactIDMatches(artifactResourceName, workloadResourceName string, isMock bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		artifactRS, ok := s.RootModule().Resources[artifactResourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", artifactResourceName)
		}
		workloadRS, ok := s.RootModule().Resources[workloadResourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", workloadResourceName)
		}

		artifactID := artifactRS.Primary.Attributes["artifact_id"]
		workloadArtifactID := workloadRS.Primary.Attributes["artifact_id"]
		if artifactID == "" {
			return fmt.Errorf("artifact_id is not set on %s", artifactResourceName)
		}
		if workloadArtifactID != artifactID {
			return fmt.Errorf("workload artifact_id %q does not match artifact %q", workloadArtifactID, artifactID)
		}

		if isMock {
			return nil
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("provider not found")
		}
		p.service = NewService(cl)

		workload, err := p.service.GetWorkload(context.Background(), workloadRS.Primary.ID)
		if err != nil {
			return fmt.Errorf("GetWorkload(%s): %w", workloadRS.Primary.ID, err)
		}
		if workload.ArtifactID == nil || *workload.ArtifactID != artifactID {
			got := ""
			if workload.ArtifactID != nil {
				got = *workload.ArtifactID
			}
			return fmt.Errorf("workload API artifact_id %q does not match %q", got, artifactID)
		}

		return nil
	}
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

	workload.Type = client.ArtifactTypeMCP
	loadWorkloadIntoModel(workload, &data)
	if data.Type.ValueString() != string(client.ArtifactTypeMCP) {
		t.Fatalf("Type = %q, want %q", data.Type.ValueString(), client.ArtifactTypeMCP)
	}

	workload.Type = ""
	loadWorkloadIntoModel(workload, &data)
	if !data.Type.IsNull() {
		t.Fatalf("empty Type = %v, want null", data.Type)
	}
}

// ---------------------------------------------------------------------------
// Enclave placement (use_case_id, runtime.enclave_selection_policy, runtime.enclaves)
// ---------------------------------------------------------------------------

func enclaveListForTest(names ...string) types.List {
	elements := make([]attr.Value, len(names))
	for i, n := range names {
		elements[i] = types.StringValue(n)
	}
	return types.ListValueMust(types.StringType, elements)
}

func workloadPlacementModel(useCaseID types.String, policy types.String, enclaves types.List) WorkloadResourceModel {
	return WorkloadResourceModel{
		Name:        types.StringValue("placement-test"),
		ArtifactID:  types.StringValue("artifact-1"),
		Importance:  types.StringValue("low"),
		Description: types.StringNull(),
		UseCaseID:   useCaseID,
		Runtime: WorkloadRuntimeModel{
			EnclaveSelectionPolicy: policy,
			Enclaves:               enclaves,
		},
	}
}

func TestValidateWorkloadEnclavePlacement(t *testing.T) {
	t.Parallel()

	nullList := types.ListNull(types.StringType)

	testCases := map[string]struct {
		useCaseID types.String
		policy    types.String
		enclaves  types.List
		wantErr   string
	}{
		"no placement at all": {
			useCaseID: types.StringNull(),
			policy:    types.StringNull(),
			enclaves:  nullList,
		},
		"use case alone is a placement": {
			useCaseID: types.StringValue("uc-1"),
			policy:    types.StringNull(),
			enclaves:  nullList,
		},
		"named enclave with a use case": {
			useCaseID: types.StringValue("uc-1"),
			policy:    types.StringValue("manual"),
			enclaves:  enclaveListForTest("finance"),
		},
		"availability with a use case": {
			useCaseID: types.StringValue("uc-1"),
			policy:    types.StringValue("availability"),
			enclaves:  nullList,
		},
		"named enclave without a use case": {
			useCaseID: types.StringNull(),
			policy:    types.StringNull(),
			enclaves:  enclaveListForTest("finance"),
			wantErr:   "Missing use_case_id for Enclave placement",
		},
		"policy without a use case": {
			useCaseID: types.StringNull(),
			policy:    types.StringValue("availability"),
			enclaves:  nullList,
			wantErr:   "Missing use_case_id for Enclave placement",
		},
		"named enclave with availability policy": {
			useCaseID: types.StringValue("uc-1"),
			policy:    types.StringValue("availability"),
			enclaves:  enclaveListForTest("finance"),
			wantErr:   "Conflicting Enclave placement",
		},
		"manual policy without a named enclave": {
			useCaseID: types.StringValue("uc-1"),
			policy:    types.StringValue("manual"),
			enclaves:  nullList,
			wantErr:   "Missing Enclave for manual placement",
		},
		"manual policy with an empty enclave list": {
			useCaseID: types.StringValue("uc-1"),
			policy:    types.StringValue("manual"),
			enclaves:  types.ListValueMust(types.StringType, []attr.Value{}),
			wantErr:   "Missing Enclave for manual placement",
		},
		// An unresolved value is "set, not resolved yet". Terraform validates the
		// configuration again during the plan walk, where these have resolved.
		"unresolved use case is not a missing use case": {
			useCaseID: types.StringUnknown(),
			policy:    types.StringNull(),
			enclaves:  enclaveListForTest("finance"),
		},
		"unresolved policy is not judged against the enclave list": {
			useCaseID: types.StringValue("uc-1"),
			policy:    types.StringUnknown(),
			enclaves:  enclaveListForTest("finance"),
		},
		"unresolved enclave list satisfies a manual policy": {
			useCaseID: types.StringValue("uc-1"),
			policy:    types.StringValue("manual"),
			enclaves:  types.ListUnknown(types.StringType),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := &fwresource.ValidateConfigResponse{}
			validateWorkloadEnclavePlacement(workloadPlacementModel(tc.useCaseID, tc.policy, tc.enclaves), resp)

			if tc.wantErr == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected error: %v", resp.Diagnostics.Errors())
				}
				return
			}

			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected error %q, got none", tc.wantErr)
			}
			found := false
			for _, d := range resp.Diagnostics.Errors() {
				if d.Summary() == tc.wantErr {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected error %q, got %v", tc.wantErr, resp.Diagnostics.Errors())
			}
		})
	}
}

func TestWorkloadCreateRequestEnclavePlacement(t *testing.T) {
	t.Parallel()

	nullList := types.ListNull(types.StringType)

	testCases := map[string]struct {
		useCaseID  types.String
		policy     types.String
		enclaves   types.List
		wantUC     *string
		wantPolicy *client.EnclaveSelectionPolicy
		wantEncl   []string
	}{
		"no placement sends no placement": {
			useCaseID: types.StringNull(),
			policy:    types.StringNull(),
			enclaves:  nullList,
		},
		// A use case is an organizational link and says nothing about placement, so
		// it is sent on its own and the workload stays off the Enclave path. Implying
		// a policy here is what made use_case_id unusable without the Enclave feature.
		"use case alone requests no Enclave": {
			useCaseID: types.StringValue("uc-1"),
			policy:    types.StringNull(),
			enclaves:  nullList,
			wantUC:    strPtr("uc-1"),
		},
		// And the API rejects `enclaves` under any policy but manual, so naming
		// one has to imply the pin.
		"named enclave implies manual": {
			useCaseID:  types.StringValue("uc-1"),
			policy:     types.StringNull(),
			enclaves:   enclaveListForTest("finance"),
			wantUC:     strPtr("uc-1"),
			wantPolicy: enclavePolicyPtr(client.EnclaveSelectionPolicyManual),
			wantEncl:   []string{"finance"},
		},
		"an explicit policy is honoured": {
			useCaseID:  types.StringValue("uc-1"),
			policy:     types.StringValue("availability"),
			enclaves:   nullList,
			wantUC:     strPtr("uc-1"),
			wantPolicy: enclavePolicyPtr(client.EnclaveSelectionPolicyAvailability),
		},
		"an unresolved use case is not sent and implies nothing": {
			useCaseID: types.StringUnknown(),
			policy:    types.StringNull(),
			enclaves:  nullList,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := workloadCreateRequest(workloadPlacementModel(tc.useCaseID, tc.policy, tc.enclaves))

			switch {
			case tc.wantUC == nil && req.UseCaseID != nil:
				t.Fatalf("UseCaseID = %q, want unset", *req.UseCaseID)
			case tc.wantUC != nil && req.UseCaseID == nil:
				t.Fatalf("UseCaseID unset, want %q", *tc.wantUC)
			case tc.wantUC != nil && *req.UseCaseID != *tc.wantUC:
				t.Fatalf("UseCaseID = %q, want %q", *req.UseCaseID, *tc.wantUC)
			}

			got := req.Runtime.EnclaveSelectionPolicy
			switch {
			case tc.wantPolicy == nil && got != nil:
				t.Fatalf("EnclaveSelectionPolicy = %q, want unset", *got)
			case tc.wantPolicy != nil && got == nil:
				t.Fatalf("EnclaveSelectionPolicy unset, want %q", *tc.wantPolicy)
			case tc.wantPolicy != nil && *got != *tc.wantPolicy:
				t.Fatalf("EnclaveSelectionPolicy = %q, want %q", *got, *tc.wantPolicy)
			}

			if len(req.Runtime.Enclaves) != len(tc.wantEncl) {
				t.Fatalf("Enclaves = %v, want %v", req.Runtime.Enclaves, tc.wantEncl)
			}
			for i, want := range tc.wantEncl {
				if req.Runtime.Enclaves[i] != want {
					t.Fatalf("Enclaves[%d] = %q, want %q", i, req.Runtime.Enclaves[i], want)
				}
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func enclavePolicyPtr(p client.EnclaveSelectionPolicy) *client.EnclaveSelectionPolicy { return &p }

func TestLoadWorkloadRuntimeEnclaves(t *testing.T) {
	t.Parallel()

	policy := client.EnclaveSelectionPolicyManual
	model := loadWorkloadRuntimeFromAPI(client.WorkloadRuntime{
		EnclaveSelectionPolicy: &policy,
		Enclaves:               []string{"finance"},
	})

	if model.EnclaveSelectionPolicy.ValueString() != "manual" {
		t.Fatalf("EnclaveSelectionPolicy = %v, want manual", model.EnclaveSelectionPolicy)
	}
	got := model.Enclaves.Elements()
	if len(got) != 1 {
		t.Fatalf("Enclaves = %v, want [finance]", model.Enclaves)
	}
	enclave, ok := got[0].(types.String)
	if !ok || enclave.ValueString() != "finance" {
		t.Fatalf("Enclaves = %v, want [finance]", model.Enclaves)
	}

	// A cluster without the Enclave entitlement omits both fields. The list still
	// needs its element type, or writing the model to state fails.
	absent := loadWorkloadRuntimeFromAPI(client.WorkloadRuntime{})
	if !absent.EnclaveSelectionPolicy.IsNull() {
		t.Fatalf("EnclaveSelectionPolicy = %v, want null", absent.EnclaveSelectionPolicy)
	}
	if !absent.Enclaves.IsNull() {
		t.Fatalf("Enclaves = %v, want null", absent.Enclaves)
	}
	if absent.Enclaves.ElementType(context.Background()) != types.StringType {
		t.Fatalf("Enclaves element type = %v, want string", absent.Enclaves.ElementType(context.Background()))
	}
}

func TestPreserveWorkloadEnclavePlacementOverAPIOmission(t *testing.T) {
	t.Parallel()

	configured := workloadPlacementModel(
		types.StringValue("uc-1"),
		types.StringValue("manual"),
		enclaveListForTest("finance"),
	)

	// What a cluster without the Enclave entitlement reports back.
	fromAPI := configured
	fromAPI.Runtime = loadWorkloadRuntimeFromAPI(client.WorkloadRuntime{})

	preserveWorkloadEnclavePlacement(configured, &fromAPI)

	if fromAPI.Runtime.EnclaveSelectionPolicy.ValueString() != "manual" {
		t.Fatalf("EnclaveSelectionPolicy = %v, want the configured manual", fromAPI.Runtime.EnclaveSelectionPolicy)
	}
	if got := fromAPI.Runtime.Enclaves.Elements(); len(got) != 1 {
		t.Fatalf("Enclaves = %v, want the configured [finance]", fromAPI.Runtime.Enclaves)
	}
}

func TestWorkloadEnclavesRequireUseCase(t *testing.T) {
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
				Config:      workloadConfigWithPlacement("enclave-no-use-case", artifactID, "", "", "finance"),
				ExpectError: regexp.MustCompile("Missing use_case_id for Enclave placement"),
			},
		},
	})
}

func TestWorkloadManualPolicyRequiresEnclave(t *testing.T) {
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
				Config:      workloadConfigWithPlacement("manual-no-enclave", artifactID, uuid.NewString(), "manual", ""),
				ExpectError: regexp.MustCompile("Missing Enclave for manual placement"),
			},
		},
	})
}

func TestWorkloadRejectsUnsupportedEnclaveSelectionPolicy(t *testing.T) {
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
				Config:      workloadConfigWithPlacement("bad-policy", artifactID, uuid.NewString(), "whatever", ""),
				ExpectError: regexp.MustCompile("Invalid Attribute Value Match"),
			},
		},
	})
}

func TestWorkloadRejectsMultipleEnclaves(t *testing.T) {
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
				Config:      workloadConfigWithPlacement("two-enclaves", artifactID, uuid.NewString(), "manual", `"finance", "research"`),
				ExpectError: regexp.MustCompile("Invalid Attribute Value"),
			},
		},
	})
}

func TestIntegrationWorkloadEnclavePlacement(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)

	id := uuid.NewString()
	artifactID := uuid.NewString()
	useCaseID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id

	// The fixture deliberately carries no Enclave fields: that is what a cluster
	// without the Enclave entitlement returns, and state must still hold what the
	// configuration asked for.
	workload := workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)

	var created *client.CreateWorkloadRequest
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *client.CreateWorkloadRequest) (*client.Workload, error) {
			created = req
			return workload, nil
		})
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload, nil) // waitForRunning
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload, nil) // post-create Read

	// Destroy
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(workload, nil)
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).Return(nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).Return(nil, client.NewNotFoundError("workload"))

	resourceName := "datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithPlacement(name, artifactID, useCaseID, "", "finance"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "use_case_id", useCaseID),
					resource.TestCheckResourceAttr(resourceName, "runtime.enclaves.0", "finance"),
					// Derived, not configured: it stays out of state so the
					// configuration and the state agree.
					resource.TestCheckNoResourceAttr(resourceName, "runtime.enclave_selection_policy"),
					func(*terraform.State) error {
						if created == nil {
							return fmt.Errorf("CreateWorkload was never called")
						}
						if created.UseCaseID == nil || *created.UseCaseID != useCaseID {
							return fmt.Errorf("useCaseId sent = %v, want %q", created.UseCaseID, useCaseID)
						}
						policy := created.Runtime.EnclaveSelectionPolicy
						if policy == nil || *policy != client.EnclaveSelectionPolicyManual {
							return fmt.Errorf("enclaveSelectionPolicy sent = %v, want manual", policy)
						}
						if len(created.Runtime.Enclaves) != 1 || created.Runtime.Enclaves[0] != "finance" {
							return fmt.Errorf("enclaves sent = %v, want [finance]", created.Runtime.Enclaves)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestIntegrationWorkloadRepinsEnclaveInPlace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)

	id := uuid.NewString()
	artifactID := uuid.NewString()
	useCaseID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id
	workload := workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)

	manual := client.EnclaveSelectionPolicyManual
	expectWorkloadPlacementUpdate(mockService, workload, placementMatcher{policy: &manual, enclaves: []string{"research"}, explicit: true})

	var initialID string
	resourceName := "datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithPlacement(name, artifactID, useCaseID, "manual", "finance"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.enclaves.0", "finance"),
					captureAttr(resourceName, "id", &initialID),
				),
			},
			{
				Config:           workloadConfigWithPlacement(name, artifactID, useCaseID, "manual", "research"),
				ConfigPlanChecks: expectInPlacePlacementChange(resourceName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.enclaves.0", "research"),
					resource.TestCheckResourceAttr(resourceName, "endpoint", endpoint),
					checkWorkloadIDPreserved(&initialID),
				),
			},
		},
	})
}

// Removing the placement while use_case_id stays fails the plan; nothing reaches the API.
func TestIntegrationWorkloadRefusesLeavingEnclaveInPlace(t *testing.T) {
	for name, tc := range map[string]struct{ policy, enclaves string }{
		"policy removed": {policy: "availability"},
		"pin removed":    {enclaves: "finance"},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockService := mock_client.NewMockService(ctrl)
			defer HookGlobal(&NewService, func(c *client.Client) client.Service { return mockService })()
			mockAPIKey(t)

			artifactID, useCaseID, name := uuid.NewString(), uuid.NewString(), "workload-"+uuid.NewString()[:8]
			replicaCount := int64(1)
			expectWorkloadsCreatedAndDeleted(mockService,
				workloadFixture(uuid.NewString(), artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, nil))

			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: workloadConfigWithPlacement(name, artifactID, useCaseID, tc.policy, tc.enclaves)},
					{
						Config:      workloadConfigWithPlacement(name, artifactID, useCaseID, "", ""),
						ExpectError: regexp.MustCompile("Cannot move a Workload off its Enclave in place"),
					},
				},
			})
		})
	}
}

// Removing use_case_id along with the placement replaces the Workload instead.
func TestIntegrationWorkloadLeavesEnclaveByReplacement(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service { return mockService })()
	mockAPIKey(t)

	artifactID, name := uuid.NewString(), "workload-"+uuid.NewString()[:8]
	replicaCount := int64(1)
	expectWorkloadsCreatedAndDeleted(mockService,
		workloadFixture(uuid.NewString(), artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, nil),
		workloadFixture(uuid.NewString(), artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, nil))

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: workloadConfigWithPlacement(name, artifactID, uuid.NewString(), "availability", "")},
			{
				Config: workloadConfigWithPlacement(name, artifactID, "", "", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("datarobot_workload.test", plancheck.ResourceActionReplace),
				}},
			},
		},
	})
}

// expectWorkloadsCreatedAndDeleted serves each Workload, in create order, until it is deleted.
func expectWorkloadsCreatedAndDeleted(mockService *mock_client.MockService, workloads ...*client.Workload) {
	live := map[string]*client.Workload{}
	created := 0
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).DoAndReturn(
		func(context.Context, *client.CreateWorkloadRequest) (*client.Workload, error) {
			w := workloads[created]
			created++
			live[w.ID] = w
			return w, nil
		}).Times(len(workloads))
	mockService.EXPECT().GetWorkload(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, id string) (*client.Workload, error) {
			if w, ok := live[id]; ok {
				return w, nil
			}
			return nil, client.NewNotFoundError("workload")
		}).AnyTimes()
	mockService.EXPECT().DeleteWorkload(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, id string) error {
			delete(live, id)
			return nil
		}).Times(len(workloads))
}

// A use_case_id change on its own moves the link in place, new link first, and keeps the
// Workload. The new Use Case is looked up first; a link the lookup does not show but the
// platform already has is accepted.
func TestIntegrationWorkloadChangesUseCaseInPlace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service { return mockService })()
	mockAPIKey(t)

	id, artifactID, name := uuid.NewString(), uuid.NewString(), "workload-"+uuid.NewString()[:8]
	firstUseCase, secondUseCase := uuid.NewString(), uuid.NewString()
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id
	expectWorkloadsCreatedAndDeleted(mockService,
		workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint))
	gomock.InOrder(
		mockService.EXPECT().ListUseCasesForEntity(gomock.Any(), "workload", id).
			Return([]client.UseCaseResponse{{ID: firstUseCase}}, false, nil),
		mockService.EXPECT().AddEntityToUseCase(gomock.Any(), secondUseCase, "workload", id).Return(nil),
		mockService.EXPECT().RemoveEntityFromUseCase(gomock.Any(), firstUseCase, "workload", id).Return(nil),
		mockService.EXPECT().RemoveEntityFromUseCase(gomock.Any(), secondUseCase, "workload", id).Return(nil),
		mockService.EXPECT().ListUseCasesForEntity(gomock.Any(), "workload", id).Return(nil, false, nil),
		mockService.EXPECT().AddEntityToUseCase(gomock.Any(), firstUseCase, "workload", id).
			Return(fmt.Errorf("409 Conflict: This workload is already linked to this Use Case.")),
	)

	resourceName := "datarobot_workload.test"
	inPlace := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
		plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
	}}
	var initialID string

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithPlacement(name, artifactID, firstUseCase, "", ""),
				Check:  captureAttr(resourceName, "id", &initialID),
			},
			{
				Config:           workloadConfigWithPlacement(name, artifactID, secondUseCase, "", ""),
				ConfigPlanChecks: inPlace,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "use_case_id", secondUseCase),
					resource.TestCheckResourceAttr(resourceName, "endpoint", endpoint),
					checkWorkloadIDPreserved(&initialID),
				),
			},
			{
				Config:           workloadConfigWithPlacement(name, artifactID, "", "", ""),
				ConfigPlanChecks: inPlace,
				Check:            resource.TestCheckNoResourceAttr(resourceName, "use_case_id"),
			},
			{
				Config:           workloadConfigWithPlacement(name, artifactID, firstUseCase, "", ""),
				ConfigPlanChecks: inPlace,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "use_case_id", firstUseCase),
					checkWorkloadIDPreserved(&initialID),
				),
			},
		},
	})
}

// The QA repro: a Workload on an Enclave repointed at another Use Case with its placement
// kept. It used to be destroyed first; now only the link moves and nothing is rolled out.
func TestIntegrationWorkloadChangesUseCaseInPlaceOnItsEnclave(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service { return mockService })()
	mockAPIKey(t)

	id, artifactID, name := uuid.NewString(), uuid.NewString(), "workload-"+uuid.NewString()[:8]
	firstUseCase, secondUseCase := uuid.NewString(), uuid.NewString()
	replicaCount := int64(1)
	expectWorkloadsCreatedAndDeleted(mockService,
		workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, nil))
	gomock.InOrder(
		mockService.EXPECT().ListUseCasesForEntity(gomock.Any(), "workload", id).
			Return([]client.UseCaseResponse{{ID: firstUseCase}}, false, nil),
		mockService.EXPECT().AddEntityToUseCase(gomock.Any(), secondUseCase, "workload", id).Return(nil),
		mockService.EXPECT().RemoveEntityFromUseCase(gomock.Any(), firstUseCase, "workload", id).Return(nil),
	)

	resourceName := "datarobot_workload.test"
	var initialID string

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithPlacement(name, artifactID, firstUseCase, "availability", ""),
				Check:  captureAttr(resourceName, "id", &initialID),
			},
			{
				Config: workloadConfigWithPlacement(name, artifactID, secondUseCase, "availability", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
				}},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "use_case_id", secondUseCase),
					checkWorkloadIDPreserved(&initialID),
				),
			},
		},
	})
}

// Changed together with the placement, use_case_id still replaces the Workload, also when it
// had none: without a placement in state, a missing use_case_id is not one import left out.
func TestIntegrationWorkloadReplacesWhenUseCaseAndPlacementChange(t *testing.T) {
	for name, initialUseCase := range map[string]string{"from another Use Case": uuid.NewString(), "from no Use Case": ""} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockService := mock_client.NewMockService(ctrl)
			defer HookGlobal(&NewService, func(c *client.Client) client.Service { return mockService })()
			mockAPIKey(t)

			artifactID, name := uuid.NewString(), "workload-"+uuid.NewString()[:8]
			replicaCount := int64(1)
			expectWorkloadsCreatedAndDeleted(mockService,
				workloadFixture(uuid.NewString(), artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, nil),
				workloadFixture(uuid.NewString(), artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, nil))

			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: workloadConfigWithPlacement(name, artifactID, initialUseCase, "", "")},
					{
						Config: workloadConfigWithPlacement(name, artifactID, uuid.NewString(), "availability", ""),
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("datarobot_workload.test", plancheck.ResourceActionReplace),
						}},
					},
				},
			})
		})
	}
}

// A failed link call leaves the old link and the old use_case_id in state, and the next apply
// retries the move.
func TestIntegrationWorkloadKeepsUseCaseWhenLinkFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service { return mockService })()
	mockAPIKey(t)

	id, artifactID, name := uuid.NewString(), uuid.NewString(), "workload-"+uuid.NewString()[:8]
	firstUseCase, secondUseCase := uuid.NewString(), uuid.NewString()
	replicaCount := int64(1)
	expectWorkloadsCreatedAndDeleted(mockService,
		workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, nil))
	// Counted rather than strictly ordered: an unexpected call inside resource.Test hangs
	// instead of failing, so ordering is checked with t.Errorf.
	var links, unlinks int
	mockService.EXPECT().ListUseCasesForEntity(gomock.Any(), "workload", id).
		Return([]client.UseCaseResponse{{ID: firstUseCase}}, false, nil).AnyTimes()
	mockService.EXPECT().AddEntityToUseCase(gomock.Any(), secondUseCase, "workload", id).DoAndReturn(
		func(context.Context, string, string, string) error {
			links++
			if links == 1 {
				return fmt.Errorf("403 Forbidden: cannot link to this Use Case")
			}
			return nil
		}).AnyTimes()
	mockService.EXPECT().RemoveEntityFromUseCase(gomock.Any(), firstUseCase, "workload", id).DoAndReturn(
		func(context.Context, string, string, string) error {
			unlinks++
			if links < 2 {
				t.Errorf("old Use Case unlinked before the new link succeeded")
			}
			return nil
		}).AnyTimes()

	resourceName := "datarobot_workload.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: workloadConfigWithPlacement(name, artifactID, firstUseCase, "", "")},
			{
				Config:      workloadConfigWithPlacement(name, artifactID, secondUseCase, "", ""),
				ExpectError: regexp.MustCompile("Error changing the Use Case of the Workload"),
			},
			{
				// The old value stayed in state.
				Config:   workloadConfigWithPlacement(name, artifactID, firstUseCase, "", ""),
				PlanOnly: true,
			},
			{
				// The next apply retries the move.
				Config: workloadConfigWithPlacement(name, artifactID, secondUseCase, "", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "use_case_id", secondUseCase),
					func(*terraform.State) error {
						if links != 2 || unlinks != 1 {
							return fmt.Errorf("link calls = %d, unlink calls = %d, want 2 and 1", links, unlinks)
						}
						return nil
					},
				),
			},
		},
	})
}

// The new use_case_id reaches state as soon as the link moves, so a rollout that fails
// afterwards leaves state matching the links and the next plan does not relink.
func TestIntegrationWorkloadRecordsUseCaseWhenRolloutFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service { return mockService })()
	mockAPIKey(t)

	id, artifact1, artifact2, name := uuid.NewString(), uuid.NewString(), uuid.NewString(), "workload-"+uuid.NewString()[:8]
	firstUseCase, secondUseCase := uuid.NewString(), uuid.NewString()
	replicaCount := int64(1)
	expectWorkloadsCreatedAndDeleted(mockService,
		workloadFixture(id, artifact1, name, "", client.WorkloadImportanceLow, &replicaCount, nil))
	gomock.InOrder(
		mockService.EXPECT().ListUseCasesForEntity(gomock.Any(), "workload", id).
			Return([]client.UseCaseResponse{{ID: firstUseCase}}, false, nil),
		mockService.EXPECT().AddEntityToUseCase(gomock.Any(), secondUseCase, "workload", id).Return(nil),
		mockService.EXPECT().RemoveEntityFromUseCase(gomock.Any(), firstUseCase, "workload", id).Return(nil),
		mockService.EXPECT().StartWorkloadReplacement(gomock.Any(), id, gomock.Any()).
			Return(nil, fmt.Errorf("422 ENCLAVE_NOT_IN_USE_CASE")),
	)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: workloadConfigWithPlacement(name, artifact1, firstUseCase, "", "")},
			{
				Config:      workloadConfigWithPlacement(name, artifact2, secondUseCase, "", ""),
				ExpectError: regexp.MustCompile("ENCLAVE_NOT_IN_USE_CASE"),
			},
			{
				Config:   workloadConfigWithPlacement(name, artifact1, secondUseCase, "", ""),
				PlanOnly: true,
			},
		},
	})
}

func TestIntegrationWorkloadPlacementChangeRidesArtifactReplacement(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)

	id := uuid.NewString()
	artifact1 := uuid.NewString()
	artifact2 := uuid.NewString()
	useCaseID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id
	workload1 := workloadFixture(id, artifact1, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)
	workload2 := workloadFixture(id, artifact2, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)

	current := workload1
	deleted := false
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload1, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).DoAndReturn(
		func(context.Context, string) (*client.Workload, error) {
			if deleted {
				return nil, client.NewNotFoundError("workload")
			}
			return current, nil
		}).AnyTimes()

	manual := client.EnclaveSelectionPolicyManual
	replacement := workloadReplacementFixture(id)
	mockService.EXPECT().StartWorkloadReplacement(gomock.Any(), id,
		replacementPlacementMatcher{artifactID: artifact2, placement: placementMatcher{policy: &manual, enclaves: []string{"finance"}, explicit: true}}).
		DoAndReturn(func(context.Context, string, *client.StartReplacementRequest) (*client.WorkloadReplacement, error) {
			current = workload2
			return replacement, nil
		})
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), id, waitExpectsArtifact(artifact2)).Return(replacement, nil)
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).DoAndReturn(
		func(context.Context, string) error {
			deleted = true
			return nil
		})

	var initialID string
	resourceName := "datarobot_workload.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: workloadConfigWithPlacement(name, artifact1, useCaseID, "availability", ""),
				Check:  captureAttr(resourceName, "id", &initialID),
			},
			{
				Config:           workloadConfigWithPlacement(name, artifact2, useCaseID, "", "finance"),
				ConfigPlanChecks: expectInPlacePlacementChange(resourceName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "artifact_id", artifact2),
					resource.TestCheckResourceAttr(resourceName, "runtime.enclaves.0", "finance"),
					checkWorkloadIDPreserved(&initialID),
				),
			},
		},
	})
}

// expectInPlacePlacementChange: an update, not a replacement, with endpoint and status unknown.
func expectInPlacePlacementChange(resourceName string) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{
		PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
			plancheck.ExpectUnknownValue(resourceName, tfjsonpath.New("endpoint")),
			plancheck.ExpectUnknownValue(resourceName, tfjsonpath.New("status")),
		},
	}
}

func expectWorkloadPlacementUpdate(mockService *mock_client.MockService, workload *client.Workload, want placementMatcher) {
	id := workload.ID
	deleted := false
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).DoAndReturn(
		func(context.Context, string) (*client.Workload, error) {
			if deleted {
				return nil, client.NewNotFoundError("workload")
			}
			return workload, nil
		}).AnyTimes()

	replacement := workloadReplacementFixture(id)
	mockService.EXPECT().UpdateWorkloadSettings(gomock.Any(), id, settingsPlacementMatcher{want}).Return(replacement, nil)
	mockService.EXPECT().WaitForWorkloadReplacement(gomock.Any(), id, waitExpectsArtifact(*workload.ArtifactID)).Return(replacement, nil)
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).DoAndReturn(
		func(context.Context, string) error {
			deleted = true
			return nil
		})
}

type placementMatcher struct {
	policy   *client.EnclaveSelectionPolicy
	enclaves []string
	explicit bool
}

func (m placementMatcher) matches(runtime client.WorkloadRuntime) bool {
	if runtime.ExplicitEnclavePlacement != m.explicit {
		return false
	}
	if (runtime.EnclaveSelectionPolicy == nil) != (m.policy == nil) {
		return false
	}
	if m.policy != nil && *runtime.EnclaveSelectionPolicy != *m.policy {
		return false
	}
	return slices.Equal(runtime.Enclaves, m.enclaves)
}

func (m placementMatcher) String() string {
	policy := "<none>"
	if m.policy != nil {
		policy = string(*m.policy)
	}
	return fmt.Sprintf("placement policy=%s enclaves=%v explicit=%v", policy, m.enclaves, m.explicit)
}

type settingsPlacementMatcher struct{ placementMatcher }

func (m settingsPlacementMatcher) Matches(x any) bool {
	req, ok := x.(*client.UpdateWorkloadSettingsRequest)
	return ok && req != nil && m.matches(req.Runtime)
}

func (m settingsPlacementMatcher) String() string {
	return "settings request with " + m.placementMatcher.String()
}

type replacementPlacementMatcher struct {
	artifactID string
	placement  placementMatcher
}

func (m replacementPlacementMatcher) Matches(x any) bool {
	req, ok := x.(*client.StartReplacementRequest)
	if !ok || req == nil || req.Runtime == nil {
		return false
	}
	return req.ArtifactID == m.artifactID && req.Strategy == client.ReplacementStrategyRolling && m.placement.matches(*req.Runtime)
}

func (m replacementPlacementMatcher) String() string {
	return fmt.Sprintf("replacement to artifact %s carrying %s", m.artifactID, m.placement.String())
}

func TestWorkloadPlacementChanged(t *testing.T) {
	nullList := types.ListNull(types.StringType)
	emptyList := types.ListValueMust(types.StringType, []attr.Value{})
	none := types.StringNull()
	availability := types.StringValue(string(client.EnclaveSelectionPolicyAvailability))
	manual := types.StringValue(string(client.EnclaveSelectionPolicyManual))
	runtime := func(policy types.String, enclaves types.List) WorkloadRuntimeModel {
		return workloadPlacementModel(types.StringNull(), policy, enclaves).Runtime
	}

	cases := map[string]struct {
		plan, state WorkloadRuntimeModel
		want        bool
	}{
		"no placement on either side":     {runtime(none, nullList), runtime(none, nullList), false},
		"same explicit policy":            {runtime(availability, nullList), runtime(availability, nullList), false},
		"policy removed":                  {runtime(none, nullList), runtime(availability, nullList), true},
		"policy added":                    {runtime(availability, nullList), runtime(none, nullList), true},
		"scheduler choice to a pin":       {runtime(none, enclaveListForTest("finance")), runtime(availability, nullList), true},
		"pin moved to another Enclave":    {runtime(none, enclaveListForTest("research")), runtime(none, enclaveListForTest("finance")), true},
		"pin removed":                     {runtime(none, nullList), runtime(none, enclaveListForTest("finance")), true},
		"explicit manual equals derived":  {runtime(manual, enclaveListForTest("finance")), runtime(none, enclaveListForTest("finance")), false},
		"empty list equals omitted list":  {runtime(none, emptyList), runtime(none, nullList), false},
		"unresolved policy counts as set": {runtime(types.StringUnknown(), nullList), runtime(none, nullList), true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := workloadPlacementChanged(tc.plan, tc.state); got != tc.want {
				t.Fatalf("workloadPlacementChanged = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWorkloadPlacementCleared(t *testing.T) {
	nullList := types.ListNull(types.StringType)
	none := types.StringNull()
	availability := types.StringValue(string(client.EnclaveSelectionPolicyAvailability))
	pin := enclaveListForTest("finance")
	runtime := func(policy types.String, enclaves types.List) WorkloadRuntimeModel {
		return workloadPlacementModel(types.StringNull(), policy, enclaves).Runtime
	}

	cases := map[string]struct {
		plan, state WorkloadRuntimeModel
		want        bool
	}{
		"policy removed":                 {runtime(none, nullList), runtime(availability, nullList), true},
		"pin removed":                    {runtime(none, nullList), runtime(none, pin), true},
		"pin emptied":                    {runtime(none, types.ListValueMust(types.StringType, nil)), runtime(none, pin), true},
		"pin dropped for availability":   {runtime(availability, nullList), runtime(none, pin), false},
		"never placed":                   {runtime(none, nullList), runtime(none, nullList), false},
		"unresolved policy":              {runtime(types.StringUnknown(), nullList), runtime(availability, nullList), false},
		"unresolved Enclave list":        {runtime(none, types.ListUnknown(types.StringType)), runtime(availability, nullList), false},
		"unresolved Enclave in the list": {runtime(none, types.ListValueMust(types.StringType, []attr.Value{types.StringUnknown()})), runtime(none, pin), false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := workloadPlacementCleared(tc.plan, tc.state); got != tc.want {
				t.Fatalf("workloadPlacementCleared = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWorkloadRuntimeUpdateWritesPlacementOnlyWhenChanged(t *testing.T) {
	nullList := types.ListNull(types.StringType)
	none := types.StringNull()
	availability := types.StringValue(string(client.EnclaveSelectionPolicyAvailability))
	manual := types.StringValue(string(client.EnclaveSelectionPolicyManual))
	runtime := func(policy types.String, enclaves types.List) WorkloadRuntimeModel {
		return workloadPlacementModel(types.StringNull(), policy, enclaves).Runtime
	}
	policyOf := func(r client.WorkloadRuntime) string {
		if r.EnclaveSelectionPolicy == nil {
			return "<nil>"
		}
		return string(*r.EnclaveSelectionPolicy)
	}

	cases := map[string]struct {
		plan, state  WorkloadRuntimeModel
		wantExplicit bool
		wantPolicy   string
		wantEnclaves []string
	}{
		"policy dropped":                {runtime(none, nullList), runtime(availability, nullList), true, "<nil>", nil},
		"never had a placement":         {runtime(none, nullList), runtime(none, nullList), false, "<nil>", nil},
		"unchanged placement":           {runtime(availability, nullList), runtime(availability, nullList), false, "availability", nil},
		"scheduler choice to a pin":     {runtime(none, enclaveListForTest("finance")), runtime(availability, nullList), true, "manual", []string{"finance"}},
		"pin removed, policy kept":      {runtime(availability, nullList), runtime(none, enclaveListForTest("finance")), true, "availability", nil},
		"pin removed, manual kept":      {runtime(manual, nullList), runtime(none, enclaveListForTest("finance")), true, "manual", nil},
		"pin moved to another Enclave":  {runtime(none, enclaveListForTest("research")), runtime(none, enclaveListForTest("finance")), true, "manual", []string{"research"}},
		"replica-only style, pin stays": {runtime(none, enclaveListForTest("finance")), runtime(none, enclaveListForTest("finance")), false, "manual", []string{"finance"}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := workloadRuntimeUpdate(tc.plan, tc.state)
			if got.ExplicitEnclavePlacement != tc.wantExplicit {
				t.Fatalf("ExplicitEnclavePlacement = %v, want %v", got.ExplicitEnclavePlacement, tc.wantExplicit)
			}
			if policyOf(got) != tc.wantPolicy {
				t.Fatalf("policy = %s, want %s", policyOf(got), tc.wantPolicy)
			}
			if !slices.Equal(got.Enclaves, tc.wantEnclaves) {
				t.Fatalf("enclaves = %v, want %v", got.Enclaves, tc.wantEnclaves)
			}
		})
	}
}

// workloadConfigWithPlacement renders a workload with any combination of the Enclave
// placement attributes, leaving out the ones given as "".
func workloadConfigWithPlacement(name, artifactID, useCaseID, policy, enclaves string) string {
	useCase := ""
	if useCaseID != "" {
		useCase = fmt.Sprintf("use_case_id = %q", useCaseID)
	}
	policyLine := ""
	if policy != "" {
		policyLine = fmt.Sprintf("enclave_selection_policy = %q", policy)
	}
	enclavesLine := ""
	if enclaves != "" {
		if !strings.Contains(enclaves, `"`) {
			enclaves = fmt.Sprintf("%q", enclaves)
		}
		enclavesLine = fmt.Sprintf("enclaves = [%s]", enclaves)
	}
	return workloadMockConfig(fmt.Sprintf(`
resource "datarobot_workload" "test" {
  name        = %q
  importance  = "low"
  artifact_id = %q
  %s
  runtime = {
    %s
    %s
    container_groups = [
      {
        replica_count    = 1
        resource_bundles = ["cpu.small"]
      }
    ]
  }
}
`, name, artifactID, useCase, policyLine, enclavesLine))
}

// A policy that is unknown at plan time (here: the output of a terraform_data that is
// being replaced) plans as an update with endpoint unknown. When it resolves to the value
// already in state, Terraform's apply-time re-plan finds no diff: no rollout is triggered,
// the workload keeps its ID, and state stays consistent.
func TestIntegrationWorkloadUnknownPlacementResolvingToSameValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)

	id := uuid.NewString()
	artifactID := uuid.NewString()
	useCaseID := uuid.NewString()
	name := "workload-" + uuid.NewString()[:8]
	replicaCount := int64(1)
	endpoint := "https://workloads.example.com/" + id
	workload := workloadFixture(id, artifactID, name, "", client.WorkloadImportanceLow, &replicaCount, &endpoint)

	deleted := false
	mockService.EXPECT().CreateWorkload(gomock.Any(), gomock.Any()).Return(workload, nil)
	mockService.EXPECT().GetWorkload(gomock.Any(), id).DoAndReturn(
		func(context.Context, string) (*client.Workload, error) {
			if deleted {
				return nil, client.NewNotFoundError("workload")
			}
			return workload, nil
		}).AnyTimes()
	// No UpdateWorkloadSettings or StartWorkloadReplacement expectation: the placement
	// did not really change, so no rollout may be triggered.
	mockService.EXPECT().DeleteWorkload(gomock.Any(), id).DoAndReturn(
		func(context.Context, string) error {
			deleted = true
			return nil
		})

	config := func(trigger string) string {
		return workloadMockConfig(fmt.Sprintf(`
resource "terraform_data" "policy" {
  input            = "availability"
  triggers_replace = [%q]
}

resource "datarobot_workload" "test" {
  name        = %q
  importance  = "low"
  artifact_id = %q
  use_case_id = %q
  runtime = {
    enclave_selection_policy = terraform_data.policy.output
    container_groups = [
      {
        replica_count    = 1
        resource_bundles = ["cpu.small"]
      }
    ]
  }
}
`, trigger, name, artifactID, useCaseID))
	}

	resourceName := "datarobot_workload.test"
	var initialID string

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime.enclave_selection_policy", "availability"),
					captureAttr(resourceName, "id", &initialID),
				),
			},
			{
				// Replacing terraform_data makes its output unknown at plan time.
				Config: config("two"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(resourceName, tfjsonpath.New("endpoint")),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "endpoint", endpoint),
					resource.TestCheckResourceAttr(resourceName, "status", "running"),
					resource.TestCheckResourceAttr(resourceName, "runtime.enclave_selection_policy", "availability"),
					checkWorkloadIDPreserved(&initialID),
				),
			},
		},
	})
}
