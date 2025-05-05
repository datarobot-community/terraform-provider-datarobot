package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccCustomJobScheduleResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_job_schedule.test"

	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())

	customJobID := "example_custom_job_id"
	schedule := map[string]string{
		"minute":      "0",
		"hour":        "12",
		"day_of_week": "1-5",
	}
	parameterOverrides := `[{
        "field_name": "example_field",
        "type": "string",
        "value": "example_value"
    }]`

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: customJobScheduleResourceConfig(
					customJobID,
					schedule,
					parameterOverrides,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomJobScheduleResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "custom_job_id", customJobID),
					resource.TestCheckResourceAttr(resourceName, "schedule.minute", schedule["minute"]),
					resource.TestCheckResourceAttr(resourceName, "schedule.hour", schedule["hour"]),
					resource.TestCheckResourceAttr(resourceName, "schedule.day_of_week", schedule["day_of_week"]),
					resource.TestCheckResourceAttr(resourceName, "parameter_overrides.0.field_name", "example_field"),
					resource.TestCheckResourceAttr(resourceName, "parameter_overrides.0.type", "string"),
					resource.TestCheckResourceAttr(resourceName, "parameter_overrides.0.value", "example_value"),
				),
			},
			// Update schedule
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: customJobScheduleResourceConfig(
					customJobID,
					map[string]string{
						"minute":      "30",
						"hour":        "14",
						"day_of_week": "1,3,5",
					},
					parameterOverrides,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomJobScheduleResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "schedule.minute", "30"),
					resource.TestCheckResourceAttr(resourceName, "schedule.hour", "14"),
					resource.TestCheckResourceAttr(resourceName, "schedule.day_of_week", "1,3,5"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func customJobScheduleResourceConfig(
	customJobID string,
	schedule map[string]string,
	parameterOverrides string,
) string {
	return fmt.Sprintf(`
resource "datarobot_custom_job_schedule" "test" {
    custom_job_id = "%s"

    schedule {
        minute      = "%s"
        hour        = "%s"
        day_of_week = "%s"
    }

    parameter_overrides = %s
}
`, customJobID, schedule["minute"], schedule["hour"], schedule["day_of_week"], parameterOverrides)
}

func checkCustomJobScheduleResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_custom_job_schedule.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_custom_job_schedule.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = NewService(cl)

		traceAPICall("ListCustomJobSchedules")
		schedules, err := p.service.ListCustomJobSchedules(context.TODO(), rs.Primary.Attributes["custom_job_id"])
		if err != nil {
			return err
		}

		var schedule *client.CustomJobSchedule
		for _, s := range schedules {
			if s.ID == rs.Primary.ID {
				schedule = &client.CustomJobSchedule{
					ID:          s.ID,
					CustomJobID: s.CustomJobID,
				}
				break
			}
		}

		if schedule == nil {
			return fmt.Errorf("Custom Job Schedule not found")
		}
		if err != nil {
			return err
		}
		if schedule.CustomJobID != rs.Primary.Attributes["custom_job_id"] {
			return fmt.Errorf("Custom Job ID does not match")
		}

		return fmt.Errorf("Custom Job Schedule not found")
	}
}
