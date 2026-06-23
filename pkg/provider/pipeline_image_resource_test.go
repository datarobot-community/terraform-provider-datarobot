package provider

import (
	"fmt"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestIntegrationPipelineImageResource(t *testing.T) {
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
	name := "image-" + uuid.NewString()[:8]
	desc := "test pipeline image"
	pkgs1 := []string{"numpy==1.26.0"}
	pkgs2 := []string{"numpy==1.26.0", "pandas>=2.0"}

	image1 := pipelineImageFixture(id, name, &desc, pkgs1, 1)
	image2 := pipelineImageFixture(id, name, &desc, pkgs2, 2)

	// Step 1: Create
	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "PIPELINES_API_ENABLED").Return(true, nil)
	mockService.EXPECT().CreatePipelineImage(gomock.Any(), gomock.Any()).Return(image1, nil)
	mockService.EXPECT().GetPipelineImage(gomock.Any(), id).Return(image1, nil) // post-create Read

	// Step 2: Update — add pandas
	mockService.EXPECT().GetPipelineImage(gomock.Any(), id).Return(image1, nil) // pre-step2 plan
	mockService.EXPECT().UpdatePipelineImage(gomock.Any(), id, gomock.Any()).Return(image2, nil)

	// Destroy
	mockService.EXPECT().GetPipelineImage(gomock.Any(), id).Return(image2, nil) // pre-destroy plan
	mockService.EXPECT().DeletePipelineImage(gomock.Any(), id).Return(nil)

	const rn = "datarobot_pipeline_image.test"
	var initialID string

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: pipelineImageConfig(name, &desc, pkgs1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "name", name),
					resource.TestCheckResourceAttr(rn, "description", desc),
					resource.TestCheckResourceAttr(rn, "packages.#", "1"),
					resource.TestCheckResourceAttr(rn, "packages.0", "numpy==1.26.0"),
					resource.TestCheckResourceAttrSet(rn, "latest_version"),
					resource.TestCheckResourceAttrSet(rn, "latest_status"),
					captureAttr(rn, "id", &initialID),
				),
			},
			{
				Config: pipelineImageConfig(name, &desc, pkgs2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "packages.#", "2"),
					resource.TestCheckResourceAttr(rn, "packages.1", "pandas>=2.0"),
					checkWorkloadIDPreserved(rn, &initialID),
				),
			},
		},
	})
}

func TestIntegrationPipelineImageReplaceOnNameChange(t *testing.T) {
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
	name1 := "image-" + uuid.NewString()[:8]
	name2 := "image-updated-" + uuid.NewString()[:8]
	pkgs := []string{"numpy==1.26.0"}

	image1 := pipelineImageFixture(id1, name1, nil, pkgs, 1)
	image2 := pipelineImageFixture(id2, name2, nil, pkgs, 1)

	// Step 1: Create
	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "PIPELINES_API_ENABLED").Return(true, nil)
	mockService.EXPECT().CreatePipelineImage(gomock.Any(), gomock.Any()).Return(image1, nil)
	mockService.EXPECT().GetPipelineImage(gomock.Any(), id1).Return(image1, nil) // post-create Read

	// Step 2: Replace (name change triggers RequiresReplace)
	mockService.EXPECT().GetPipelineImage(gomock.Any(), id1).Return(image1, nil) // pre-replace plan
	mockService.EXPECT().DeletePipelineImage(gomock.Any(), id1).Return(nil)
	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "PIPELINES_API_ENABLED").Return(true, nil)
	mockService.EXPECT().CreatePipelineImage(gomock.Any(), gomock.Any()).Return(image2, nil)
	mockService.EXPECT().GetPipelineImage(gomock.Any(), id2).Return(image2, nil) // post-replace / pre-destroy combined

	// Destroy
	mockService.EXPECT().DeletePipelineImage(gomock.Any(), id2).Return(nil)

	const rn = "datarobot_pipeline_image.test"
	var initialID string

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: pipelineImageConfig(name1, nil, pkgs),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "name", name1),
					captureAttr(rn, "id", &initialID),
				),
			},
			{
				Config: pipelineImageConfig(name2, nil, pkgs),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "name", name2),
					checkIDChanged(rn, &initialID),
				),
			},
		},
	})
}

func TestIntegrationPipelineImageReplaceOnPackageRemoval(t *testing.T) {
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
	name := "image-" + uuid.NewString()[:8]
	pkgs2 := []string{"numpy==1.26.0", "pandas>=2.0"}
	pkgs1 := []string{"numpy==1.26.0"}

	image2pkgs := pipelineImageFixture(id1, name, nil, pkgs2, 1)
	image1pkg := pipelineImageFixture(id2, name, nil, pkgs1, 1)

	// Step 1: Create with 2 packages
	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "PIPELINES_API_ENABLED").Return(true, nil)
	mockService.EXPECT().CreatePipelineImage(gomock.Any(), gomock.Any()).Return(image2pkgs, nil)
	mockService.EXPECT().GetPipelineImage(gomock.Any(), id1).Return(image2pkgs, nil) // post-create Read

	// Step 2: Remove pandas → ModifyPlan forces RequiresReplace
	mockService.EXPECT().GetPipelineImage(gomock.Any(), id1).Return(image2pkgs, nil) // pre-replace plan
	mockService.EXPECT().DeletePipelineImage(gomock.Any(), id1).Return(nil)
	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "PIPELINES_API_ENABLED").Return(true, nil)
	mockService.EXPECT().CreatePipelineImage(gomock.Any(), gomock.Any()).Return(image1pkg, nil)
	mockService.EXPECT().GetPipelineImage(gomock.Any(), id2).Return(image1pkg, nil) // post-replace / pre-destroy combined

	// Destroy
	mockService.EXPECT().DeletePipelineImage(gomock.Any(), id2).Return(nil)

	const rn = "datarobot_pipeline_image.test"
	var initialID string

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: pipelineImageConfig(name, nil, pkgs2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "packages.#", "2"),
					captureAttr(rn, "id", &initialID),
				),
			},
			{
				Config: pipelineImageConfig(name, nil, pkgs1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "packages.#", "1"),
					checkIDChanged(rn, &initialID),
				),
			},
		},
	})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func checkIDChanged(resourceName string, initialID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		if *initialID != "" && rs.Primary.ID == *initialID {
			return fmt.Errorf("expected resource ID to change after replacement, but it stayed %q", *initialID)
		}
		return nil
	}
}

// ─── fixtures ─────────────────────────────────────────────────────────────────

func pipelineImageFixture(id, name string, desc *string, pkgs []string, version int) *client.PipelineImage {
	return &client.PipelineImage{
		ImageID:       id,
		Name:          name,
		Description:   desc,
		LatestVersion: version,
		Versions: []client.PipelineImageVersion{
			{
				Version:  version,
				Packages: pkgs,
				Status:   client.PipelineImageStatusReady,
			},
		},
		CreatedAt: "2025-01-01T00:00:00Z",
		UpdatedAt: "2025-01-01T00:00:00Z",
	}
}

// ─── config helpers ────────────────────────────────────────────────────────────

func pipelineImageConfig(name string, desc *string, pkgs []string) string {
	descAttr := ""
	if desc != nil {
		descAttr = fmt.Sprintf("description = %q", *desc)
	}
	pkgList := ""
	for _, p := range pkgs {
		pkgList += fmt.Sprintf("    %q,\n", p)
	}
	return fmt.Sprintf(`
resource "datarobot_pipeline_image" "test" {
  name     = %q
  %s
  packages = [
%s  ]
}
`, name, descAttr, pkgList)
}
