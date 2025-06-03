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

func TestAccBatchPredictionJobDefinitionResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_batch_prediction_job_definition.test"

	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())

	name := "test " + nameSalt
	newName := "new test " + nameSalt

	numConcurrent := 1
	newNumConcurrent := 2

	chunkSize := 100
	newChunkSize := 200

	maxExplanations := 1
	newMaxExplanations := 2

	thresholdHigh := 0.8
	newThresholdHigh := 0.9

	thresholdLow := 0.2
	newThresholdLow := 0.1

	predictionThreshold := 0.4
	newPredictionThreshold := 0.6

	folderPath := "batch_prediction_job_definition"
	if err := os.Mkdir(folderPath, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(folderPath)

	modelContents := `from typing import Any, Dict
import pandas as pd

class DummyModel:
	def __init__(self, positive_class_label: str, negative_class_label: str):
		self.positive_class_label = positive_class_label
		self.negative_class_label = negative_class_label

def load_model(code_dir: str) -> Any:
	return DummyModel("1", "0")

def score(data: pd.DataFrame, model: Any, **kwargs: Dict[str, Any]) -> pd.DataFrame:
	positive_label = model.positive_class_label
	negative_label = model.negative_class_label
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
				Config: batchPredictionJobDefinitionResourceConfig(
					name,
					numConcurrent,
					chunkSize,
					maxExplanations,
					thresholdHigh,
					thresholdLow,
					predictionThreshold,
					true,
					true,
					true,
					true),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkBatchPredictionJobDefinitionResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "num_concurrent", "1"),
					resource.TestCheckResourceAttr(resourceName, "chunk_size", "100"),
					resource.TestCheckResourceAttr(resourceName, "max_explanations", "1"),
					resource.TestCheckResourceAttr(resourceName, "threshold_high", "0.8"),
					resource.TestCheckResourceAttr(resourceName, "threshold_low", "0.2"),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", "0.4"),
					resource.TestCheckResourceAttr(resourceName, "include_prediction_status", "true"),
					resource.TestCheckResourceAttr(resourceName, "skip_drift_tracking", "true"),
					resource.TestCheckResourceAttr(resourceName, "abort_on_error", "true"),
					resource.TestCheckResourceAttr(resourceName, "include_probabilities", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "deployment_id"),
				),
			},
			// Update attributes
			{
				Config: batchPredictionJobDefinitionResourceConfig(
					newName,
					newNumConcurrent,
					newChunkSize,
					newMaxExplanations,
					newThresholdHigh,
					newThresholdLow,
					newPredictionThreshold,
					false,
					false,
					false,
					false),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkBatchPredictionJobDefinitionResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "num_concurrent", "2"),
					resource.TestCheckResourceAttr(resourceName, "chunk_size", "200"),
					resource.TestCheckResourceAttr(resourceName, "max_explanations", "2"),
					resource.TestCheckResourceAttr(resourceName, "threshold_high", "0.9"),
					resource.TestCheckResourceAttr(resourceName, "threshold_low", "0.1"),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", "0.6"),
					resource.TestCheckResourceAttr(resourceName, "include_prediction_status", "false"),
					resource.TestCheckResourceAttr(resourceName, "skip_drift_tracking", "false"),
					resource.TestCheckResourceAttr(resourceName, "abort_on_error", "false"),
					resource.TestCheckResourceAttr(resourceName, "include_probabilities", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "deployment_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestBatchPredictionJobDefinitionResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewBatchPredictionJobDefinitionResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func batchPredictionJobDefinitionResourceConfig(
	name string,
	numConcurrent,
	chunkSize,
	maxExplanations int,
	thresholdHigh,
	thresholdLow,
	predictionThreshold float64,
	includePredictionStatus,
	skipDriftTracking,
	abortOnError,
	includeProbabilities bool,
) string {
	return fmt.Sprintf(`
resource "datarobot_basic_credential" "batch_prediction_job_definition" {
	name = "%s"
	user = "user"
	password = "b@tch_prediction_Job"
}
resource "datarobot_custom_model" "batch_prediction_job_definition" {
	name = "test batch prediction job"
	target_type = "Binary"
	target_name = "t"
	base_environment_id = "65f9b27eab986d30d4c64268"
	folder_path = "batch_prediction_job_definition"
}
resource "datarobot_registered_model" "batch_prediction_job_definition" {
	name = "test batch prediction job %s"
	description = "test"
	custom_model_version_id = "${datarobot_custom_model.batch_prediction_job_definition.version_id}"
}
resource "datarobot_prediction_environment" "batch_prediction_job_definition" {
	name = "test batch prediction job"
	platform = "datarobotServerless"
}
resource "datarobot_deployment" "batch_prediction_job_definition" {
	label = "%s"
	importance = "LOW"
	prediction_environment_id = datarobot_prediction_environment.batch_prediction_job_definition.id
	registered_model_version_id = datarobot_registered_model.batch_prediction_job_definition.version_id
}
resource "datarobot_batch_prediction_job_definition" "test" {
	name = "%s"
	deployment_id = datarobot_deployment.batch_prediction_job_definition.id
	intake_settings = {
		type = "s3"
		url = "s3://datarobot-public-datasets-redistributable/1k_diabetes_simplified_features.csv"
		credential_id = "${datarobot_basic_credential.batch_prediction_job_definition.id}"
	}
	output_settings = {
		type = "s3"
		url = "s3://my-fake-bucket/predictions.csv"
		credential_id = "${datarobot_basic_credential.batch_prediction_job_definition.id}"
	}
	num_concurrent = %d
	chunk_size = %d
	max_explanations = %d
	threshold_high = %f
	threshold_low = %f
	prediction_threshold = %f
	include_prediction_status = %v
	skip_drift_tracking = %v
	passthrough_columns_set = "all"
	abort_on_error = %v
	include_probabilities = %v
	column_names_remapping = {
		"old_name" = "new_name"
	}
	schedule = {
		minute 			= ["10", "15"]
		hour 			= ["*"]
		month 			= ["*"]
		day_of_month 	= ["*"]
		day_of_week 	= ["*"]
	}
}`, name,
		name,
		name,
		name,
		numConcurrent,
		chunkSize,
		maxExplanations,
		thresholdHigh,
		thresholdLow,
		predictionThreshold,
		includePredictionStatus,
		skipDriftTracking,
		abortOnError,
		includeProbabilities)
}

func checkBatchPredictionJobDefinitionResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_batch_prediction_job_definition.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_batch_prediction_job_definition.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = NewService(cl)

		traceAPICall("GetBatchPredictionJobDefinition")
		batchPredictionJobDefinition, err := p.service.GetBatchPredictionJobDefinition(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if batchPredictionJobDefinition.BatchPredictionJob.DeploymentID == rs.Primary.Attributes["deployment_id"] {
			return nil
		}

		return fmt.Errorf("Batch Prediction Job Definition not found")
	}
}
