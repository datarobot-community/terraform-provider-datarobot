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

func TestIntegrationPipelineScheduleResource(t *testing.T) {
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
	schedID := uuid.NewString()
	version := 1
	cron1 := "0 9 * * 1-5"
	cron2 := "0 8 * * *"

	sched1 := pipelineScheduleFixture(schedID, pipelineID, version, cron1, "UTC")
	sched2 := pipelineScheduleFixture(schedID, pipelineID, version, cron2, "UTC")

	// Step 1: Create
	mockService.EXPECT().CreatePipelineSchedule(gomock.Any(), pipelineID, version, gomock.Any()).Return(sched1, nil)
	mockService.EXPECT().GetPipelineSchedule(gomock.Any(), pipelineID, version, schedID).Return(sched1, nil) // post-create Read

	// Step 2: Update cron expression
	mockService.EXPECT().GetPipelineSchedule(gomock.Any(), pipelineID, version, schedID).Return(sched1, nil) // pre-step2 plan
	mockService.EXPECT().UpdatePipelineSchedule(gomock.Any(), pipelineID, version, schedID, gomock.Any()).Return(sched2, nil)

	// Destroy
	mockService.EXPECT().GetPipelineSchedule(gomock.Any(), pipelineID, version, schedID).Return(sched2, nil) // pre-destroy plan
	mockService.EXPECT().DeletePipelineSchedule(gomock.Any(), pipelineID, version, schedID).Return(nil)

	const rn = "datarobot_pipeline_schedule.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: pipelineScheduleConfig(pipelineID, version, inputID, cron1, "UTC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "pipeline_id", pipelineID),
					resource.TestCheckResourceAttr(rn, "version", fmt.Sprintf("%d", version)),
					resource.TestCheckResourceAttr(rn, "pipeline_input_id", inputID),
					resource.TestCheckResourceAttr(rn, "cron_expression", cron1),
					resource.TestCheckResourceAttr(rn, "timezone", "UTC"),
					resource.TestCheckResourceAttr(rn, "status", string(client.PipelineScheduleStatusActive)),
				),
			},
			{
				Config: pipelineScheduleConfig(pipelineID, version, inputID, cron2, "UTC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "cron_expression", cron2),
					// pipeline_input_id must be preserved (not returned by API GET)
					resource.TestCheckResourceAttr(rn, "pipeline_input_id", inputID),
				),
			},
		},
	})
}

func TestPipelineScheduleInvalidVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      pipelineScheduleConfig(uuid.NewString(), 0, uuid.NewString(), "0 9 * * *", "UTC"),
				ExpectError: regexp.MustCompile("version must be >= 1"),
			},
		},
	})
}

// ─── fixtures ─────────────────────────────────────────────────────────────────

func pipelineScheduleFixture(schedID, pipelineID string, version int, cron, timezone string) *client.PipelineSchedule {
	return &client.PipelineSchedule{
		ScheduleID:     schedID,
		PipelineID:     pipelineID,
		Version:        version,
		CronExpression: cron,
		Timezone:       timezone,
		Status:         client.PipelineScheduleStatusActive,
		CreatedAt:      "2025-01-01T00:00:00Z",
		UpdatedAt:      "2025-01-01T00:00:00Z",
	}
}

// ─── config helpers ────────────────────────────────────────────────────────────

func pipelineScheduleConfig(pipelineID string, version int, inputID, cron, timezone string) string {
	return fmt.Sprintf(`
resource "datarobot_pipeline_schedule" "test" {
  pipeline_id       = %q
  version           = %d
  pipeline_input_id = %q
  cron_expression   = %q
  timezone          = %q
}
`, pipelineID, version, inputID, cron, timezone)
}
