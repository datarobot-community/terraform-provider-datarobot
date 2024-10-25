package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDeploymentResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_deployment.test"
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	useCaseResourceName := "test_deployment"
	useCaseResourceName2 := "test_new_deployment"

	folderPath := "deployment"
	if err := os.Mkdir(folderPath, 0755); err != nil {
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

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: deploymentResourceConfig("example_label", "MODERATE", "target", &useCaseResourceName, false, false, false, false, false, false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDeploymentResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "label", "example_label"),
					resource.TestCheckResourceAttr(resourceName, "importance", "MODERATE"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckNoResourceAttr(resourceName, "predictions_by_forecast_date_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "challenger_models_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "segment_analysis_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "bias_and_fairness_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "challenger_replay_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "drift_tracking_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "association_id_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "predictions_data_collection_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "prediction_warning_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "prediction_intervals_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "health_settings"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update label, importance, settings, and use case id
			{
				Config: deploymentResourceConfig("new_example_label", "LOW", "target", &useCaseResourceName2, true, true, true, true, true, true, true),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDeploymentResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "label", "new_example_label"),
					resource.TestCheckResourceAttr(resourceName, "importance", "LOW"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "predictions_by_forecast_date_settings.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "segment_analysis_settings.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "segment_analysis_settings.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "challenger_replay_settings.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "predictions_data_collection_settings.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "health_settings.service.batch_count", "5"),
					resource.TestCheckResourceAttr(resourceName, "health_settings.data_drift.batch_count", "5"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Remove settings and use case id
			{
				Config: deploymentResourceConfig("new_example_label", "LOW", "target", nil, false, false, false, false, false, false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("registered_model_version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDeploymentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "label", "new_example_label"),
					resource.TestCheckResourceAttr(resourceName, "importance", "LOW"),
					resource.TestCheckNoResourceAttr(resourceName, "use_case_ids.0"),
					resource.TestCheckNoResourceAttr(resourceName, "predictions_by_forecast_date_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "challenger_models_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "segment_analysis_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "bias_and_fairness_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "challenger_replay_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "drift_tracking_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "association_id_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "predictions_data_collection_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "prediction_warning_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "prediction_intervals_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "health_settings"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Try to update target_name of Custom Model (should fail)
			{
				Config: deploymentResourceConfig("new_example_label", "LOW", "new_target", nil, false, false, false, false, false, false, false),
				ExpectError: regexp.MustCompile(`target_name cannot be changed if the model was deployed.`),
			},
			// Update custom model version (by updating the file contents) updates registered model version of deployment
			// which triggers a model replacement for the Deployment
			{
				PreConfig: func() {
					if err := os.WriteFile(folderPath+"/custom.py", []byte(fmt.Sprintf(modelContentsTemplate, "dummy2")), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: deploymentResourceConfig("new_example_label", "LOW", "target", nil, false, false, false, false, false, false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("registered_model_version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDeploymentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "label", "new_example_label"),
					resource.TestCheckResourceAttr(resourceName, "importance", "LOW"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestDeploymentResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewDeploymentResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func deploymentResourceConfig(
	label,
	importance string,
	customModelTargetName string,
	useCaseResourceName *string,
	isPredictionsByForecastDateEnabled,
	isSegmentAnalysisEnabled,
	isChallengerReplayEnabled,
	isAssociationIDEnabled,
	isPredictionsDataCollectionEnabled,
	isPredictionsSettingsEnabled,
	isHealthSettingsEnabled bool,

) string {
	useCaseIDsStr := ""
	if useCaseResourceName != nil {
		useCaseIDsStr = fmt.Sprintf(`use_case_ids = ["${datarobot_use_case.%s.id}"]`, *useCaseResourceName)
	}

	deploymentSettings := ""

	if isPredictionsByForecastDateEnabled {
		deploymentSettings = `
	predictions_by_forecast_date_settings = {
		enabled = true 
		column_name = "column_name"
		datetime_format = "%H:%M"
	}`
	}

	if isSegmentAnalysisEnabled {
		deploymentSettings += `
	segment_analysis_settings = {
		enabled = true
	}`
	}

	if isChallengerReplayEnabled {
		deploymentSettings += `
	challenger_replay_settings = {
		enabled = true
	}`
	}

	if isAssociationIDEnabled {
		deploymentSettings += `
	association_id_settings = {
		auto_generate_id = true
		column_names = ["example_column"]
		required_in_prediction_requests = true
	}`
	}

	if isPredictionsDataCollectionEnabled {
		deploymentSettings += `
	predictions_data_collection_settings = {
		enabled = true
	}`
	}

	if isPredictionsSettingsEnabled {
		deploymentSettings += `
	predictions_settings = {
		min_computes = 0
		max_computes = 2
	}`
	}

	if isHealthSettingsEnabled {
		deploymentSettings += `
	health_settings = {
		service = {
			batch_count = 5
		}
		data_drift = {
			time_interval = "P7D"
			batch_count = 5
			drift_threshold = 0.2
			importance_threshold = 0.3
			low_importance_warning_count = 2
			low_importance_failing_count = 5
			high_importance_warning_count = 2
			high_importance_failing_count = 5
		}
		accuracy = {
			batch_count = 1000
		}
		prediction_timeliness = {
			enabled = true
			expected_frequency = "P7D"
		}
		actuals_timeliness = {
			enabled = true
			expected_frequency = "P30D"
		}
	}`
	}

	return fmt.Sprintf(`
resource "datarobot_use_case" "test_deployment" {
	name = "test deployment"
}
resource "datarobot_use_case" "test_new_deployment" {
	name = "test new deployment"
}
resource "datarobot_custom_model" "test_deployment" {
	name = "test deployment"
	description = "test"
	target_type = "Binary"
	target_name = "%s"
	base_environment_id = "65f9b27eab986d30d4c64268"
	folder_path = "deployment"
}
resource "datarobot_registered_model" "test_deployment" {
	name = "test deployment"
	description = "test"
	custom_model_version_id = "${datarobot_custom_model.test_deployment.version_id}"
}
resource "datarobot_prediction_environment" "test_deployment" {
	name = "test deployment"
	description = "test"
	platform = "datarobotServerless"
}
resource "datarobot_deployment" "test" {
	label = "%s"
	importance = "%s"
	prediction_environment_id = datarobot_prediction_environment.test_deployment.id
	registered_model_version_id = datarobot_registered_model.test_deployment.version_id
	%s
	%s
}
`, customModelTargetName, label, importance, useCaseIDsStr, deploymentSettings)
}

func checkDeploymentResourceExists() resource.TestCheckFunc {
	resourceName := "datarobot_deployment.test"
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = client.NewService(cl)

		traceAPICall("GetDeployment")
		deployment, err := p.service.GetDeployment(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if deployment.Label == rs.Primary.Attributes["label"] &&
			deployment.ModelPackage.ID == rs.Primary.Attributes["registered_model_version_id"] &&
			deployment.PredictionEnvironment.ID == rs.Primary.Attributes["prediction_environment_id"] {
			return nil
		}

		return fmt.Errorf("Deployment not found")
	}
}
