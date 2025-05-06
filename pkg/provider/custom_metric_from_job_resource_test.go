package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccCustomMetricFromJobResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_metric_from_job.test"

	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	name := "custom_metric_from_job " + nameSalt
	newName := "new_custom_metric_from_job " + nameSalt

	timeStampColumn := "timestamp"
	newTimeStampColumn := "new_timestamp"

	valueColumn := "value"
	newValueColumn := "new_value"

	batchColumn := "batch"
	newBatchColumn := "new_batch"

	sampleCountColumn := "sample_count"
	newSampleCountColumn := "new_sample_count"

	scheduleHour := "10"
	newScheduleHour := "11"

	parameterOverrides := `[
		{
			key="OPENAI_API_BASE",
			type="string",
			value="val"
		}
	]
	`
	newParameterOverrides := `[
		{
			key="OPENAI_API_BASE",
			type="string",
			value="newVal"
		}
	]
	`

	folderPath := "custom_metric_from_job"
	if err := os.Mkdir(folderPath, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(folderPath)

	modelContents := `from typing import Any, Dict
import pandas as pd

def load_model(code_dir: str) -> Any:
	return "dummy"

def score(data: pd.DataFrame, model: Any, **kwargs: Dict[str, Any]) -> pd.DataFrame:
	positive_label = kwargs["positive_class_label"]
	negative_label = kwargs["negative_class_label"]
	preds = pd.DataFrame([[0.75, 0.25]] * data.shape[0], columns=[positive_label, negative_label])
	return preds
`

	if err := os.WriteFile(folderPath+"/custom.py", []byte(modelContents), 0644); err != nil {
		t.Fatal(err)
	}

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
				Config: customMetricFromJobResourceConfig(
					name,
					timeStampColumn,
					valueColumn,
					batchColumn,
					sampleCountColumn,
					scheduleHour,
					parameterOverrides,
					0.4,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomMetricFromJobResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "baseline_value", "0.4"),
					resource.TestCheckResourceAttr(resourceName, "timestamp.column_name", timeStampColumn),
					resource.TestCheckResourceAttr(resourceName, "value.column_name", valueColumn),
					resource.TestCheckResourceAttr(resourceName, "batch.column_name", batchColumn),
					resource.TestCheckResourceAttr(resourceName, "sample_count.column_name", sampleCountColumn),
					resource.TestCheckResourceAttr(resourceName, "schedule.hour.0", scheduleHour),
				),
			},
			// Update attributes
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: customMetricFromJobResourceConfig(
					newName,
					newTimeStampColumn,
					newValueColumn,
					newBatchColumn,
					newSampleCountColumn,
					scheduleHour,
					parameterOverrides,
					0.5,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomMetricFromJobResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "baseline_value", "0.5"),
					resource.TestCheckResourceAttr(resourceName, "timestamp.column_name", newTimeStampColumn),
					resource.TestCheckResourceAttr(resourceName, "value.column_name", newValueColumn),
					resource.TestCheckResourceAttr(resourceName, "batch.column_name", newBatchColumn),
					resource.TestCheckResourceAttr(resourceName, "sample_count.column_name", newSampleCountColumn),
					resource.TestCheckResourceAttr(resourceName, "schedule.hour.0", scheduleHour),
				),
			},
			// Update schedule triggers replacement
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: customMetricFromJobResourceConfig(
					newName,
					newTimeStampColumn,
					newValueColumn,
					newBatchColumn,
					newSampleCountColumn,
					newScheduleHour,
					parameterOverrides,
					0.5,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomMetricFromJobResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "baseline_value", "0.5"),
					resource.TestCheckResourceAttr(resourceName, "timestamp.column_name", newTimeStampColumn),
					resource.TestCheckResourceAttr(resourceName, "value.column_name", newValueColumn),
					resource.TestCheckResourceAttr(resourceName, "batch.column_name", newBatchColumn),
					resource.TestCheckResourceAttr(resourceName, "sample_count.column_name", newSampleCountColumn),
					resource.TestCheckResourceAttr(resourceName, "schedule.hour.0", newScheduleHour),
					resource.TestCheckResourceAttr(resourceName, "parameter_overrides.0.value", "val"),
				),
			},
			// Update parameterOverrides triggers replacement
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: customMetricFromJobResourceConfig(
					newName,
					newTimeStampColumn,
					newValueColumn,
					newBatchColumn,
					newSampleCountColumn,
					newScheduleHour,
					newParameterOverrides,
					0.5,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomMetricFromJobResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "baseline_value", "0.5"),
					resource.TestCheckResourceAttr(resourceName, "timestamp.column_name", newTimeStampColumn),
					resource.TestCheckResourceAttr(resourceName, "value.column_name", newValueColumn),
					resource.TestCheckResourceAttr(resourceName, "batch.column_name", newBatchColumn),
					resource.TestCheckResourceAttr(resourceName, "sample_count.column_name", newSampleCountColumn),
					resource.TestCheckResourceAttr(resourceName, "schedule.hour.0", newScheduleHour),
					resource.TestCheckResourceAttr(resourceName, "parameter_overrides.0.value", "newVal"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestCustomMetricFromJobResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewCustomMetricFromJobResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func customMetricFromJobResourceConfig(
	name,
	timeStampColumn,
	value,
	batch,
	sampleCount,
	scheduleHour,
	parameterOverrides string,
	baselineValue float64,
) string {
	timeFormat := "%Y-%m-%dT%H:%M:%SZ"
	return fmt.Sprintf(`
resource "datarobot_custom_metric_job" "datarobot_custom_metric_from_job" {
	name = "%s"
	environment_id = "66d07fae0513a1edf18595bb"
}
resource "datarobot_custom_model" "datarobot_custom_metric_from_job" {
	name = "test custom metric from job"
	target_type = "Binary"
	target_name = "t"
	base_environment_id = "65f9b27eab986d30d4c64268"
	folder_path = "custom_metric_from_job"
}
resource "datarobot_registered_model" "datarobot_custom_metric_from_job" {
	name = "test custom metric from job %s"
	custom_model_version_id = "${datarobot_custom_model.datarobot_custom_metric_from_job.version_id}"
}
resource "datarobot_prediction_environment" "datarobot_custom_metric_from_job" {
	name = "test custom metric from job"
	platform = "datarobotServerless"
}
resource "datarobot_deployment" "datarobot_custom_metric_from_job" {
	label = "%s"
	importance = "LOW"
	prediction_environment_id = datarobot_prediction_environment.datarobot_custom_metric_from_job.id
	registered_model_version_id = datarobot_registered_model.datarobot_custom_metric_from_job.version_id
}
resource "datarobot_custom_metric_from_job" "test" {
	deployment_id = datarobot_deployment.datarobot_custom_metric_from_job.id
	custom_job_id = datarobot_custom_metric_job.datarobot_custom_metric_from_job.id
	name = "%s"
	baseline_value = %f
	timestamp = {
		column_name = "%s"
		time_format = "%s"
	}
	value = {
		column_name = "%s"
	}
	batch = {
		column_name = "%s"
	}
	sample_count = {
		column_name = "%s"
	}
	schedule = {
		minute 			= ["10", "15"]
		hour 			= ["%s"]
		month 			= ["*"]
		day_of_month 	= ["*"]
		day_of_week 	= ["*"]
	}
	parameter_overrides = %s
}
`, name,
		name,
		name,
		name,
		baselineValue,
		timeStampColumn,
		timeFormat,
		value,
		batch,
		sampleCount,
		scheduleHour,
		parameterOverrides)
}

func checkCustomMetricFromJobResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_custom_metric_from_job.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_custom_metric_from_job.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = NewService(cl)

		traceAPICall("GetCustomMetric")
		job, err := p.service.GetCustomMetric(context.TODO(), rs.Primary.Attributes["deployment_id"], rs.Primary.ID)
		if err != nil {
			return err
		}

		if job.Name == rs.Primary.Attributes["name"] {
			return nil
		}

		return fmt.Errorf("Custom Metric not found")
	}
}
