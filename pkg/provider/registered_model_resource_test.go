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

func TestAccRegisteredModelResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_registered_model.test"
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: registeredModelResourceConfig("example_name", "example_description", "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update name, description
			{
				Config: registeredModelResourceConfig("new_example_name", "new_example_description", "1"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update custom model version (by updating the Guard) creates new registered model version
			{
				Config: registeredModelResourceConfig("new_example_name", "new_example_description", "2"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func registeredModelResourceConfig(name, description, guardName string) string {
	return fmt.Sprintf(`
resource "datarobot_remote_repository" "test_registered_model" {
	name        = "Test Registered Model"
	description = "test"
	location    = "https://github.com/datarobot-community/custom-models"
	source_type = "github"
	}
resource "datarobot_custom_model" "test_registered_model" {
	name = "test registered model"
	description = "test"
	target_type = "Binary"
	target = "my_label"
	base_environment_name = "[GenAI] Python 3.11 with Moderations"
	source_remote_repositories = [
		{
			id = datarobot_remote_repository.test_registered_model.id
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
resource "datarobot_registered_model" "test" {
	name = "%s"
	description = "%s"
	custom_model_version_id = "${datarobot_custom_model.test_registered_model.version_id}"
}
`, guardName, name, description)
}

func checkRegisteredModelResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetRegisteredModel")
		registeredModel, err := p.service.GetRegisteredModel(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		traceAPICall("ListRegisteredModelVersions")
		latestRegisteredModelVersion, err := p.service.GetLatestRegisteredModelVersion(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if registeredModel.Name == rs.Primary.Attributes["name"] &&
			registeredModel.Description == rs.Primary.Attributes["description"] &&
			latestRegisteredModelVersion.ID == rs.Primary.Attributes["version_id"] {
			return nil
		}

		return fmt.Errorf("Registered Model not found")
	}
}
