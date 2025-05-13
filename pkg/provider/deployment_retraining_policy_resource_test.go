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

func TestAccDeploymentRetrainingPolicyResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_deployment_retraining_policy.test"

	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	name := "deployment_retraining_policy " + nameSalt
	newName := "new_deployment_retraining_policy " + nameSalt

	description := "test description"
	newDescription := "new test description"

	action := "create_model_package"
	newAction := "model_replacement"

	modelSelectionStrategy := "custom_job"
	newModelSelectionStrategy := "autopilot_recommended"

	folderPath := "retraining_policy"
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

	jobFolderPath := "retraining_policy_custom_job"
	if err := os.Mkdir(jobFolderPath, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(jobFolderPath)

	metadataFileName := "metadata.yaml"
	metadataFileContents := `name: runtime-params

runtimeParameterDefinitions:
  - fieldName: DEPLOYMENT
    type: deployment
    description: Deployment that will be used for retraining job
  - fieldName: RETRAINING_POLICY_ID
    type: string
    description: Retraining policy ID`

	if err := os.WriteFile(jobFolderPath+"/"+metadataFileName, []byte(metadataFileContents), 0644); err != nil {
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
				Config: deploymentRetrainingPolicyResourceConfig(
					name,
					description,
					action,
					modelSelectionStrategy,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDeploymentRetrainingPolicyResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "model_selection_strategy", modelSelectionStrategy),
					resource.TestCheckResourceAttr(resourceName, "action", action),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
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
				Config: deploymentRetrainingPolicyResourceConfig(
					newName,
					newDescription,
					newAction,
					newModelSelectionStrategy,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDeploymentRetrainingPolicyResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", newDescription),
					resource.TestCheckResourceAttr(resourceName, "action", newAction),
					resource.TestCheckResourceAttr(resourceName, "model_selection_strategy", newModelSelectionStrategy),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestDeploymentRetrainingPolicyResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewDeploymentRetrainingPolicyResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func deploymentRetrainingPolicyResourceConfig(
	name,
	description,
	action,
	modelSelectionStrategy string,
) string {
	return fmt.Sprintf(`
resource "datarobot_custom_job" "deployment_retraining_policy" {
	name = "%s"
	job_type = "retraining"
	environment_id = "66d07fae0513a1edf18595bb"
	folder_path = "retraining_policy_custom_job"
}
resource "datarobot_custom_model" "deployment_retraining_policy" {
    name = "test deployment retraining policy"
    target_type = "Binary"
    target_name = "t"
    base_environment_id = "65f9b27eab986d30d4c64268"
    folder_path = "retraining_policy"
}
resource "datarobot_registered_model" "deployment_retraining_policy" {
    name = "test deployment retraining policy %s"
    custom_model_version_id = "${datarobot_custom_model.deployment_retraining_policy.version_id}"
}
resource "datarobot_prediction_environment" "deployment_retraining_policy" {
    name = "test deployment retraining policy"
    platform = "datarobotServerless"
}
resource "datarobot_deployment" "deployment_retraining_policy" {
    label = "%s"
    importance = "LOW"
    prediction_environment_id = datarobot_prediction_environment.deployment_retraining_policy.id
    registered_model_version_id = datarobot_registered_model.deployment_retraining_policy.version_id
}
resource "datarobot_deployment_retraining_policy" "test" {
	name = "%s"
	description = "%s"
	action = "%s"
	model_selection_strategy = "%s"
	deployment_id = datarobot_deployment.deployment_retraining_policy.id
	feature_list_strategy = "informative_features"
	project_options_strategy = "custom"
	trigger = {
		type = "schedule"
		schedule = {
			minute 			= ["10"]
			hour 			= ["10"]
			month 			= ["*"]
			day_of_month 	= ["*"]
			day_of_week 	= ["*"]
		}
	}

}
`, name, name, name, name, description, action, modelSelectionStrategy)
}

func checkDeploymentRetrainingPolicyResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_deployment_retraining_policy.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_deployment_retraining_policy.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = NewService(cl)

		traceAPICall("GetRetrainingPolicy")
		job, err := p.service.GetRetrainingPolicy(context.TODO(), rs.Primary.Attributes["deployment_id"], rs.Primary.ID)
		if err != nil {
			return err
		}

		if job.Name == rs.Primary.Attributes["name"] &&
			job.Description == rs.Primary.Attributes["description"] &&
			job.FeatureListStrategy == rs.Primary.Attributes["feature_list_strategy"] &&
			job.ProjectOptionsStrategy == rs.Primary.Attributes["project_options_strategy"] &&
			job.ModelSelectionStrategy == rs.Primary.Attributes["model_selection_strategy"] {
			return nil
		}

		return fmt.Errorf("Retraining Policy not found or attributes mismatch")
	}
}
