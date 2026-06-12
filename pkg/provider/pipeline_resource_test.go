package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// pipelinePySrc is minimal valid pipeline source content for test temp files.
const pipelinePySrc = `
def task1():
    return 1

def pipeline():
    return task1()
`

func TestIntegrationPipelineResourceDraftCRUD(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	srcFile := writeTempPipelineFile(t, "pipeline.py", pipelinePySrc)

	id := uuid.NewString()
	draftPipeline := pipelineFixture(id, client.PipelineModeDraft, nil, nil)

	// Create
	mockService.EXPECT().CreatePipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(draftPipeline, nil)
	mockService.EXPECT().GetPipeline(gomock.Any(), id).Return(draftPipeline, nil) // post-create / pre-destroy combined

	// Destroy
	mockService.EXPECT().DeletePipeline(gomock.Any(), id).Return(nil)

	const rn = "datarobot_pipeline.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: pipelineDraftConfig(srcFile),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "mode", "draft"),
					resource.TestCheckResourceAttrSet(rn, "source_file_hash"),
					resource.TestCheckResourceAttr(rn, "source_file", srcFile),
				),
			},
		},
	})
}

func TestIntegrationPipelineResourceLockTransition(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	srcFile := writeTempPipelineFile(t, "pipeline.py", pipelinePySrc)

	id := uuid.NewString()
	taskNames := []string{"task1"}
	version := client.PipelineVersion{
		Version:       1,
		Status:        client.PipelineVersionStatusReady,
		ElectronNames: taskNames,
	}
	draftPipeline := pipelineFixture(id, client.PipelineModeDraft, nil, nil)
	lockedPipeline := pipelineFixture(id, client.PipelineModeLocked, taskNames, []client.PipelineVersion{version})

	// Step 1: Create draft
	mockService.EXPECT().CreatePipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(draftPipeline, nil)
	mockService.EXPECT().GetPipeline(gomock.Any(), id).Return(draftPipeline, nil) // post-create Read

	// Step 2: Lock (draft→locked update, same file → no UpdatePipelineDraft)
	mockService.EXPECT().GetPipeline(gomock.Any(), id).Return(draftPipeline, nil) // pre-step2 plan
	mockService.EXPECT().LockPipeline(gomock.Any(), id).Return(lockedPipeline, nil)

	// Destroy
	mockService.EXPECT().GetPipeline(gomock.Any(), id).Return(lockedPipeline, nil) // pre-destroy plan
	mockService.EXPECT().DeletePipeline(gomock.Any(), id).Return(nil)

	const rn = "datarobot_pipeline.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: pipelineDraftConfig(srcFile),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "mode", "draft"),
					resource.TestCheckResourceAttrSet(rn, "id"),
				),
			},
			{
				Config: pipelineLockedConfig(srcFile),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "mode", "locked"),
					resource.TestCheckResourceAttr(rn, "current_version", "1"),
					resource.TestCheckResourceAttr(rn, "task_names.0", "task1"),
				),
			},
		},
	})
}

func TestIntegrationPipelineResourceLockedReplaceOnDescriptionChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	srcFile := writeTempPipelineFile(t, "pipeline.py", pipelinePySrc)

	id1 := uuid.NewString()
	id2 := uuid.NewString()
	desc1 := "version one"
	desc2 := "version two"

	version := client.PipelineVersion{Version: 1, Status: client.PipelineVersionStatusReady}
	pipeline1Draft := pipelineFixture(id1, client.PipelineModeDraft, nil, nil)
	pipeline1Draft.Description = &desc1
	lockedPipeline1 := pipelineFixture(id1, client.PipelineModeLocked, nil, []client.PipelineVersion{version})
	lockedPipeline1.Description = &desc1

	pipeline2Draft := pipelineFixture(id2, client.PipelineModeDraft, nil, nil)
	pipeline2Draft.Description = &desc2
	lockedPipeline2 := pipelineFixture(id2, client.PipelineModeLocked, nil, []client.PipelineVersion{version})
	lockedPipeline2.Description = &desc2

	// Step 1: Create locked pipeline
	mockService.EXPECT().CreatePipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(pipeline1Draft, nil)
	mockService.EXPECT().LockPipeline(gomock.Any(), id1).Return(lockedPipeline1, nil)
	mockService.EXPECT().GetPipeline(gomock.Any(), id1).Return(lockedPipeline1, nil) // post-create Read

	// Step 2: Change description → RequiresReplace (locked pipeline)
	mockService.EXPECT().GetPipeline(gomock.Any(), id1).Return(lockedPipeline1, nil) // pre-replace plan
	mockService.EXPECT().DeletePipeline(gomock.Any(), id1).Return(nil)
	mockService.EXPECT().CreatePipeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(pipeline2Draft, nil)
	mockService.EXPECT().LockPipeline(gomock.Any(), id2).Return(lockedPipeline2, nil)
	mockService.EXPECT().GetPipeline(gomock.Any(), id2).Return(lockedPipeline2, nil) // post-replace / pre-destroy combined

	// Destroy
	mockService.EXPECT().DeletePipeline(gomock.Any(), id2).Return(nil)

	const rn = "datarobot_pipeline.test"
	var initialID string

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: pipelineLockedWithDescConfig(srcFile, desc1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "mode", "locked"),
					resource.TestCheckResourceAttr(rn, "description", desc1),
					captureAttr(rn, "id", &initialID),
				),
			},
			{
				Config: pipelineLockedWithDescConfig(srcFile, desc2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "description", desc2),
					checkIDChanged(rn, &initialID),
				),
			},
		},
	})
}

// ─── fixtures ─────────────────────────────────────────────────────────────────

func pipelineFixture(id string, mode client.PipelineMode, electronNames []string, versions []client.PipelineVersion) *client.Pipeline {
	return &client.Pipeline{
		PipelineID:    id,
		Mode:          mode,
		ElectronNames: electronNames,
		Versions:      versions,
		CreatedAt:     "2025-01-01T00:00:00Z",
		UpdatedAt:     "2025-01-01T00:00:00Z",
	}
}

// ─── config helpers ────────────────────────────────────────────────────────────

func pipelineDraftConfig(srcFile string) string {
	return fmt.Sprintf(`
resource "datarobot_pipeline" "test" {
  source_file = %q
  mode        = "draft"
}
`, srcFile)
}

func pipelineLockedConfig(srcFile string) string {
	return fmt.Sprintf(`
resource "datarobot_pipeline" "test" {
  source_file = %q
  mode        = "locked"
}
`, srcFile)
}

func pipelineLockedWithDescConfig(srcFile, description string) string {
	return fmt.Sprintf(`
resource "datarobot_pipeline" "test" {
  source_file = %q
  mode        = "locked"
  description = %q
}
`, srcFile, description)
}

// ─── utility ──────────────────────────────────────────────────────────────────

func writeTempPipelineFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writeTempPipelineFile: %v", err)
	}
	return path
}
