package provider

import (
	"fmt"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestIntegrationPipelineInputDraftCRUD(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	pipelineID := uuid.NewString()
	inputID := uuid.NewString()
	payload1 := map[string]any{"param1": "value1"}
	payload2 := map[string]any{"param1": "value2"}

	input1 := pipelineInputFixture(inputID, pipelineID, nil, payload1)
	input2 := pipelineInputFixture(inputID, pipelineID, nil, payload2)

	// Step 1: Create draft input
	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "PIPELINES_API_ENABLED").Return(true, nil)
	mockService.EXPECT().CreateDraftPipelineInput(gomock.Any(), pipelineID, gomock.Any()).Return(input1, nil)
	mockService.EXPECT().GetDraftPipelineInput(gomock.Any(), pipelineID, inputID).Return(input1, nil) // post-create Read

	// Step 2: Update payload (draft: patch in-place)
	mockService.EXPECT().GetDraftPipelineInput(gomock.Any(), pipelineID, inputID).Return(input1, nil) // pre-step2 plan
	mockService.EXPECT().UpdateDraftPipelineInput(gomock.Any(), pipelineID, inputID, gomock.Any()).Return(input2, nil)

	// Destroy
	mockService.EXPECT().GetDraftPipelineInput(gomock.Any(), pipelineID, inputID).Return(input2, nil) // pre-destroy plan
	mockService.EXPECT().DeleteDraftPipelineInput(gomock.Any(), pipelineID, inputID).Return(nil)

	const rn = "datarobot_pipeline_input.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: pipelineInputDraftConfig(pipelineID, `{"param1":"value1"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "pipeline_id", pipelineID),
					resource.TestCheckResourceAttr(rn, "state", string(client.PipelineInputStateValid)),
					resource.TestCheckNoResourceAttr(rn, "version"),
				),
			},
			{
				Config: pipelineInputDraftConfig(pipelineID, `{"param1":"value2"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "pipeline_id", pipelineID),
					resource.TestCheckResourceAttr(rn, "id", inputID),
				),
			},
		},
	})
}

func TestIntegrationPipelineInputLockedCRUD(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	pipelineID := uuid.NewString()
	inputID := uuid.NewString()
	version := 1
	payload := map[string]any{"param1": "value1"}

	lockedInput := pipelineInputFixture(inputID, pipelineID, &version, payload)

	// Create locked input
	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "PIPELINES_API_ENABLED").Return(true, nil)
	mockService.EXPECT().CreateLockedPipelineInput(gomock.Any(), pipelineID, version, gomock.Any()).Return(lockedInput, nil)
	mockService.EXPECT().GetLockedPipelineInput(gomock.Any(), pipelineID, version, inputID).Return(lockedInput, nil) // post-create / pre-destroy combined

	// Destroy
	mockService.EXPECT().DeleteLockedPipelineInput(gomock.Any(), pipelineID, version, inputID).Return(nil)

	const rn = "datarobot_pipeline_input.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: pipelineInputLockedConfig(pipelineID, version, `{"param1":"value1"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "pipeline_id", pipelineID),
					resource.TestCheckResourceAttr(rn, "version", fmt.Sprintf("%d", version)),
					resource.TestCheckResourceAttr(rn, "state", string(client.PipelineInputStateValid)),
				),
			},
		},
	})
}

func TestIntegrationPipelineInputLockedPayloadUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	pipelineID := uuid.NewString()
	inputID1 := uuid.NewString()
	inputID2 := uuid.NewString()
	version := 1
	payload1 := map[string]any{"param1": "value1"}
	payload2 := map[string]any{"param1": "value2"}

	lockedInput1 := pipelineInputFixture(inputID1, pipelineID, &version, payload1)
	lockedInput2 := pipelineInputFixture(inputID2, pipelineID, &version, payload2)

	// Step 1: Create locked input
	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "PIPELINES_API_ENABLED").Return(true, nil)
	mockService.EXPECT().CreateLockedPipelineInput(gomock.Any(), pipelineID, version, gomock.Any()).Return(lockedInput1, nil)
	mockService.EXPECT().GetLockedPipelineInput(gomock.Any(), pipelineID, version, inputID1).Return(lockedInput1, nil) // post-create Read

	// Step 2: payload change → ModifyPlan forces RequiresReplace for locked inputs
	mockService.EXPECT().GetLockedPipelineInput(gomock.Any(), pipelineID, version, inputID1).Return(lockedInput1, nil) // pre-replace plan
	mockService.EXPECT().DeleteLockedPipelineInput(gomock.Any(), pipelineID, version, inputID1).Return(nil)
	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "PIPELINES_API_ENABLED").Return(true, nil)
	mockService.EXPECT().CreateLockedPipelineInput(gomock.Any(), pipelineID, version, gomock.Any()).Return(lockedInput2, nil)
	mockService.EXPECT().GetLockedPipelineInput(gomock.Any(), pipelineID, version, inputID2).Return(lockedInput2, nil) // post-replace / pre-destroy combined

	// Destroy
	mockService.EXPECT().DeleteLockedPipelineInput(gomock.Any(), pipelineID, version, inputID2).Return(nil)

	const rn = "datarobot_pipeline_input.test"
	var initialID string

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: pipelineInputLockedConfig(pipelineID, version, `{"param1":"value1"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "id", inputID1),
					captureAttr(rn, "id", &initialID),
				),
			},
			{
				Config: pipelineInputLockedConfig(pipelineID, version, `{"param1":"value2"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "id", inputID2),
					checkIDChanged(rn, &initialID),
				),
			},
		},
	})
}

// ─── fixtures ─────────────────────────────────────────────────────────────────

func pipelineInputFixture(inputID, pipelineID string, versionID *int, payload map[string]any) *client.PipelineInput {
	return &client.PipelineInput{
		InputID:    inputID,
		PipelineID: pipelineID,
		VersionID:  versionID,
		IsDraft:    versionID == nil,
		Payload:    payload,
		State:      client.PipelineInputStateValid,
		CreatedAt:  "2025-01-01T00:00:00Z",
		UpdatedAt:  "2025-01-01T00:00:00Z",
	}
}

// ─── config helpers ────────────────────────────────────────────────────────────

func pipelineInputDraftConfig(pipelineID, payloadJSON string) string {
	return fmt.Sprintf(`
resource "datarobot_pipeline_input" "test" {
  pipeline_id = %q
  payload     = %q
}
`, pipelineID, payloadJSON)
}

func pipelineInputLockedConfig(pipelineID string, version int, payloadJSON string) string {
	return fmt.Sprintf(`
resource "datarobot_pipeline_input" "test" {
  pipeline_id = %q
  version     = %d
  payload     = %q
}
`, pipelineID, version, payloadJSON)
}
