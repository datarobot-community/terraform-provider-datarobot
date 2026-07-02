package provider

import (
	"context"
	"fmt"
	"slices"
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

	// Step 2: Update — add pandas. The update body must be a complete
	// redefinition: both the pre-existing and the newly-added package, plus
	// the required "name" field (regression coverage for a bug where only
	// the diff was sent and "name" was omitted, silently dropping packages
	// and failing API validation).
	mockService.EXPECT().GetPipelineImage(gomock.Any(), id).Return(image1, nil) // pre-step2 plan
	mockService.EXPECT().UpdatePipelineImage(gomock.Any(), id, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, req *client.PipelineImageUpdateRequest) (*client.PipelineImage, error) {
			if req.Name != name {
				t.Errorf("UpdatePipelineImage: expected Name %q, got %q", name, req.Name)
			}
			if !slices.Equal(req.Packages, pkgs2) {
				t.Errorf("UpdatePipelineImage: expected full package list %v, got %v", pkgs2, req.Packages)
			}
			return image2, nil
		},
	)

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

func TestIntegrationPipelineImageResource_PythonBaseImageAndImageURI(t *testing.T) {
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
	pkgs := []string{"numpy==1.26.0"}
	baseImage := "covalent-runtime-image:latest"
	imageURI := "123456789.dkr.ecr.us-east-1.amazonaws.com/pipelines/images/" + id + ":v1"

	image := pipelineImageFixtureFull(id, name, nil, pkgs, &baseImage, &imageURI, 1)

	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "PIPELINES_API_ENABLED").Return(true, nil)
	mockService.EXPECT().CreatePipelineImage(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *client.PipelineImageCreateRequest) (*client.PipelineImage, error) {
			if req.PythonBaseImage == nil || *req.PythonBaseImage != baseImage {
				t.Errorf("CreatePipelineImage: expected PythonBaseImage %q, got %v", baseImage, req.PythonBaseImage)
			}
			return image, nil
		},
	)
	mockService.EXPECT().GetPipelineImage(gomock.Any(), id).Return(image, nil).AnyTimes() // post-create + pre-destroy reads
	mockService.EXPECT().DeletePipelineImage(gomock.Any(), id).Return(nil)

	const rn = "datarobot_pipeline_image.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: pipelineImageConfigWithBaseImage(name, nil, pkgs, &baseImage),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "python_base_image", baseImage),
					resource.TestCheckResourceAttr(rn, "latest_image_uri", imageURI),
				),
			},
		},
	})
}

func TestIntegrationPipelineImageResource_UpdateTriggeredByBaseImageChangeAlone(t *testing.T) {
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
	pkgs := []string{"numpy==1.26.0"}
	baseImage1 := "python:3.11"
	baseImage2 := "python:3.12"

	image1 := pipelineImageFixtureWithBaseImage(id, name, nil, pkgs, &baseImage1, 1)
	image2 := pipelineImageFixtureWithBaseImage(id, name, nil, pkgs, &baseImage2, 2)

	// Step 1: Create
	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "PIPELINES_API_ENABLED").Return(true, nil)
	mockService.EXPECT().CreatePipelineImage(gomock.Any(), gomock.Any()).Return(image1, nil)
	mockService.EXPECT().GetPipelineImage(gomock.Any(), id).Return(image1, nil) // post-create Read

	// Step 2: only python_base_image changes (packages untouched) — must still
	// call UpdatePipelineImage. Regression coverage for a bug where the
	// resource only checked for newly-added packages before deciding whether
	// to call the API, silently ignoring a base-image-only change.
	mockService.EXPECT().GetPipelineImage(gomock.Any(), id).Return(image1, nil) // pre-step2 plan
	mockService.EXPECT().UpdatePipelineImage(gomock.Any(), id, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, req *client.PipelineImageUpdateRequest) (*client.PipelineImage, error) {
			if req.PythonBaseImage == nil || *req.PythonBaseImage != baseImage2 {
				t.Errorf("UpdatePipelineImage: expected PythonBaseImage %q, got %v", baseImage2, req.PythonBaseImage)
			}
			if !slices.Equal(req.Packages, pkgs) {
				t.Errorf("UpdatePipelineImage: expected unchanged package list %v, got %v", pkgs, req.Packages)
			}
			return image2, nil
		},
	)

	// Destroy
	mockService.EXPECT().GetPipelineImage(gomock.Any(), id).Return(image2, nil) // pre-destroy plan
	mockService.EXPECT().DeletePipelineImage(gomock.Any(), id).Return(nil)

	const rn = "datarobot_pipeline_image.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: pipelineImageConfigWithBaseImage(name, nil, pkgs, &baseImage1),
				Check:  resource.TestCheckResourceAttr(rn, "python_base_image", baseImage1),
			},
			{
				Config: pipelineImageConfigWithBaseImage(name, nil, pkgs, &baseImage2),
				Check:  resource.TestCheckResourceAttr(rn, "python_base_image", baseImage2),
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
	return pipelineImageFixtureFull(id, name, desc, pkgs, nil, nil, version)
}

func pipelineImageFixtureWithBaseImage(id, name string, desc *string, pkgs []string, baseImage *string, version int) *client.PipelineImage {
	return pipelineImageFixtureFull(id, name, desc, pkgs, baseImage, nil, version)
}

func pipelineImageFixtureFull(id, name string, desc *string, pkgs []string, baseImage *string, imageURI *string, version int) *client.PipelineImage {
	return &client.PipelineImage{
		ImageID:       id,
		Name:          name,
		Description:   desc,
		LatestVersion: version,
		Versions: []client.PipelineImageVersion{
			{
				Version: version,
				Definition: client.PipelineImageDefinition{
					Name:            name,
					Packages:        pkgs,
					PythonBaseImage: baseImage,
				},
				Status:   client.PipelineImageStatusReady,
				ImageURI: imageURI,
			},
		},
		CreatedAt: "2025-01-01T00:00:00Z",
		UpdatedAt: "2025-01-01T00:00:00Z",
	}
}

// ─── config helpers ────────────────────────────────────────────────────────────

func pipelineImageConfig(name string, desc *string, pkgs []string) string {
	return pipelineImageConfigWithBaseImage(name, desc, pkgs, nil)
}

func pipelineImageConfigWithBaseImage(name string, desc *string, pkgs []string, baseImage *string) string {
	descAttr := ""
	if desc != nil {
		descAttr = fmt.Sprintf("description = %q", *desc)
	}
	baseImageAttr := ""
	if baseImage != nil {
		baseImageAttr = fmt.Sprintf("python_base_image = %q", *baseImage)
	}
	pkgList := ""
	for _, p := range pkgs {
		pkgList += fmt.Sprintf("    %q,\n", p)
	}
	return fmt.Sprintf(`
resource "datarobot_pipeline_image" "test" {
  name     = %q
  %s
  %s
  packages = [
%s  ]
}
`, name, descAttr, baseImageAttr, pkgList)
}
