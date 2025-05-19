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

func TestAccCustomMetricResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_metric.test"

	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	folderPath := "custom_metric"
	if err := createOrCleanDirectory(folderPath); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(folderPath)

	modelContentsTemplate := `from typing import Any, Dict
import pandas as pd

def load_model(code_dir: str) -> Any:
	return "%s"

def score(data: pd.DataFrame, model: Any, **kwargs: Dict[str, Any]) -> pd.DataFrame:
	positive_label = kwargs["positive_class_label"]
	negative_label = kwargs["negative_class_label"]
	preds = pd.DataFrame([[0.75, 0.25]] * data.shape[0], columns=[positive_label, negative_label])
	return preds
`

	if err := os.WriteFile(folderPath+"/custom.py", []byte(fmt.Sprintf(modelContentsTemplate, "dummy")), 0644); err != nil {
		t.Fatal(err)
	}

	name := "custom_metric " + nameSalt
	newName := "new_custom_metric " + nameSalt

	description := "example_description"
	newDescription := "new_example_description"

	units := "example_units"
	newUnits := "new_example_units"

	directionality := "higherIsBetter"
	newDirectionality := "lowerIsBetter"

	aggType := "sum"
	newAggType := "gauge"

	timeStampColumn := "timestamp"
	newTimeStampColumn := "new_timestamp"

	valueColumn := "value"
	newValueColumn := "new_value"

	batchColumn := "batch"
	newBatchColumn := "new_batch"

	sampleCountColumn := "sample_count"
	newSampleCountColumn := "new_sample_count"

	baselineValue := 0.4
	newBaselineValue := 0.5

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
				Config: customMetricResourceConfig(
					name,
					description,
					units,
					directionality,
					aggType,
					timeStampColumn,
					valueColumn,
					batchColumn,
					sampleCountColumn,
					baselineValue,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomMetricResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "units", units),
					resource.TestCheckResourceAttr(resourceName, "directionality", directionality),
					resource.TestCheckResourceAttr(resourceName, "type", aggType),
					resource.TestCheckResourceAttr(resourceName, "baseline_value", "0.4"),
					resource.TestCheckResourceAttr(resourceName, "timestamp.column_name", timeStampColumn),
					resource.TestCheckResourceAttr(resourceName, "value.column_name", valueColumn),
					resource.TestCheckResourceAttr(resourceName, "batch.column_name", batchColumn),
					resource.TestCheckResourceAttr(resourceName, "sample_count.column_name", sampleCountColumn),
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
				Config: customMetricResourceConfig(
					newName,
					newDescription,
					newUnits,
					newDirectionality,
					newAggType,
					newTimeStampColumn,
					newValueColumn,
					newBatchColumn,
					newSampleCountColumn,
					newBaselineValue,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomMetricResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", newDescription),
					resource.TestCheckResourceAttr(resourceName, "units", newUnits),
					resource.TestCheckResourceAttr(resourceName, "directionality", newDirectionality),
					resource.TestCheckResourceAttr(resourceName, "type", newAggType),
					resource.TestCheckResourceAttr(resourceName, "baseline_value", "0.5"),
					resource.TestCheckResourceAttr(resourceName, "timestamp.column_name", newTimeStampColumn),
					resource.TestCheckResourceAttr(resourceName, "value.column_name", newValueColumn),
					resource.TestCheckResourceAttr(resourceName, "batch.column_name", newBatchColumn),
					resource.TestCheckResourceAttr(resourceName, "sample_count.column_name", newSampleCountColumn),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestCustomMetricResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewCustomMetricResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func customMetricResourceConfig(
	name,
	description,
	untis,
	directionality,
	aggType,
	timeStampColumn,
	value,
	batch,
	sampleCount string,
	baselineValue float64,
) string {
	timeFormat := "%Y-%m-%dT%H:%M:%SZ"
	return fmt.Sprintf(`
resource "datarobot_custom_model" "test_custom_metric" {
	name = "test custom metric"
	description = "test"
	target_type = "Binary"
	target_name = "target"
	base_environment_id = "65f9b27eab986d30d4c64268"
	folder_path = "custom_metric"
}
resource "datarobot_registered_model" "test_custom_metric" {
	name = "test custom metric %s"
	description = "test"
	custom_model_version_id = "${datarobot_custom_model.test_custom_metric.version_id}"
}
resource "datarobot_prediction_environment" "test_custom_metric" {
	name = "test deployment"
	description = "test"
	platform = "datarobotServerless"
}
resource "datarobot_deployment" "test_custom_metric" {
	label = "test custom metric"
	importance = "LOW"
	prediction_environment_id = datarobot_prediction_environment.test_custom_metric.id
	registered_model_version_id = datarobot_registered_model.test_custom_metric.version_id
}
resource "datarobot_custom_metric" "test" {
	deployment_id = datarobot_deployment.test_custom_metric.id
	name = "%s"
	description = "%s"
	units = "%s"
	directionality = "%s"
	type = "%s"
	baseline_value = %f
	is_model_specific = true
	is_geospatial = false
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
}
`, nameSalt,
		name,
		description,
		untis,
		directionality,
		aggType,
		baselineValue,
		timeStampColumn,
		timeFormat,
		value,
		batch,
		sampleCount,
	)
}

func checkCustomMetricResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_custom_metric.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_custom_metric.test")
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
		customMetric, err := p.service.GetCustomMetric(context.TODO(), rs.Primary.Attributes["deployment_id"], rs.Primary.ID)
		if err != nil {
			return err
		}

		if customMetric.Name == rs.Primary.Attributes["name"] &&
			customMetric.Description == rs.Primary.Attributes["description"] &&
			customMetric.Units == rs.Primary.Attributes["units"] &&
			customMetric.Directionality == rs.Primary.Attributes["directionality"] &&
			customMetric.Type == rs.Primary.Attributes["type"] {
			return nil
		}

		return fmt.Errorf("Custom Metric not found")
	}
}
