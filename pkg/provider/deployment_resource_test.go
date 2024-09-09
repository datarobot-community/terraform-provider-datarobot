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

func TestAccDeploymentResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_deployment.test"
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: deploymentResourceConfig("example_label", "1", &client.DeploymentSettings{
					AssociationID: &client.AssociationIDSetting{
						AutoGenerateID:               true,
						ColumnNames:                  []string{"example_column"},
						RequiredInPredictionRequests: true,
					},
					PredictionsDataCollection: &client.BasicSetting{
						Enabled: true,
					},
					PredictionsSettings: &client.PredictionsSettings{
						MinComputes: 0,
						MaxComputes: 2,
						RealTime:    true,
					},
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDeploymentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "label", "example_label"),
					resource.TestCheckResourceAttr(resourceName, "settings.association_id.auto_generate_id", "true"),
					resource.TestCheckResourceAttr(resourceName, "settings.association_id.feature_name", "example_column"),
					resource.TestCheckResourceAttr(resourceName, "settings.association_id.required_in_prediction_requests", "true"),
					resource.TestCheckResourceAttr(resourceName, "settings.prediction_row_storage", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update label and settings
			{
				Config: deploymentResourceConfig("new_example_label", "1", &client.DeploymentSettings{
					AssociationID: &client.AssociationIDSetting{
						AutoGenerateID:               false,
						ColumnNames:                  []string{"new_example_column"},
						RequiredInPredictionRequests: false,
					},
					PredictionsDataCollection: &client.BasicSetting{
						Enabled: false,
					},
					PredictionsSettings: &client.PredictionsSettings{
						MinComputes: 0,
						MaxComputes: 1,
						RealTime:    false,
					},
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDeploymentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "label", "new_example_label"),
					resource.TestCheckResourceAttr(resourceName, "settings.association_id.auto_generate_id", "false"),
					resource.TestCheckResourceAttr(resourceName, "settings.association_id.feature_name", "new_example_column"),
					resource.TestCheckResourceAttr(resourceName, "settings.association_id.required_in_prediction_requests", "false"),
					resource.TestCheckResourceAttr(resourceName, "settings.prediction_row_storage", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Remove settings
			{
				Config: deploymentResourceConfig("new_example_label", "1", nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("registered_model_version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDeploymentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "label", "new_example_label"),
					resource.TestCheckNoResourceAttr(resourceName, "settings"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update custom model version (by updating the Guard) updates registered model version of deployment
			{
				Config: deploymentResourceConfig("new_example_label", "2", nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("registered_model_version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDeploymentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "label", "new_example_label"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func deploymentResourceConfig(label, guardName string, settings *client.DeploymentSettings) string {
	deploymentSettings := ""

	if settings != nil {
		settings := *settings
		associationID := *settings.AssociationID
		predictionsDataCollection := *settings.PredictionsDataCollection
		predictionsSettings := *settings.PredictionsSettings
		deploymentSettings = fmt.Sprintf(`
	settings = {
		association_id = {
			auto_generate_id = %t
			feature_name = "%s"
			required_in_prediction_requests = %t
		}
		prediction_row_storage = %t
		predictions_settings = {
			min_computes = %d
			max_computes = %d
			real_time = %t
		}
	}
`,
			associationID.AutoGenerateID,
			associationID.ColumnNames[0],
			associationID.RequiredInPredictionRequests,
			predictionsDataCollection.Enabled,
			predictionsSettings.MinComputes,
			predictionsSettings.MaxComputes,
			predictionsSettings.RealTime)
	}

	return fmt.Sprintf(`
resource "datarobot_remote_repository" "test_deployment" {
	name        = "Test Deployment"
	description = "test"
	location    = "https://github.com/datarobot-community/custom-models"
	source_type = "github"
	}
resource "datarobot_custom_model" "test_deployment" {
	name = "test deployment"
	description = "test"
	target_type = "Binary"
	target_name = "my_label"
	base_environment_name = "[GenAI] Python 3.11 with Moderations"
	source_remote_repositories = [
		{
			id = datarobot_remote_repository.test_deployment.id
			ref = "master"
			source_paths = [
				"custom_inference/python/gan_mnist/custom.py",
			]
		},
	]
	guard_configurations = [
		{
			template_name = "Rouge 1"
			name = "Rouge 1 %v"
			stages = [ "response" ]
			intervention = {
				action = "block"
				message = "you have been blocked by rouge 1 guard"
				condition = {
					comparand = 0.8
					comparator = "lessThan"
				}
			}
		},
	]
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
	prediction_environment_id = datarobot_prediction_environment.test_deployment.id
	registered_model_version_id = datarobot_registered_model.test_deployment.version_id
	%s
}
`, guardName, label, deploymentSettings)
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
