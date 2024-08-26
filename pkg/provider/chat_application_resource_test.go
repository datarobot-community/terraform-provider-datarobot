package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccChatApplicationResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_chat_application.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: chatApplicationResourceConfig("example_name"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkChatApplicationResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name
			{
				Config: chatApplicationResourceConfig("new_example_name"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkChatApplicationResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func chatApplicationResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "datarobot_use_case" "test_chat_application" {
	name = "test chat application"
	description = "test"
}
resource "datarobot_dataset_from_file" "test_chat_application" {
	source_file = "../../test/datarobot_english_documentation_docsassist.zip"
	use_case_id = "${datarobot_use_case.test_chat_application.id}"
}
resource "datarobot_vector_database" "test_chat_application" {
	  name = "test chat application"
	  dataset_id = "${datarobot_dataset_from_file.test_chat_application.id}"
	  use_case_id = "${datarobot_use_case.test_chat_application.id}"
	  chunking_parameters = {}
}
resource "datarobot_playground" "test_chat_application" {
	name = "test chat application"
	description = "test"
	use_case_id = "${datarobot_use_case.test_chat_application.id}"
}
resource "datarobot_llm_blueprint" "test_chat_application" {
	name = "test chat application"
	description = "test"
	vector_database_id = "${datarobot_vector_database.test_chat_application.id}"
	playground_id = "${datarobot_playground.test_chat_application.id}"
	llm_id = "azure-openai-gpt-3.5-turbo"
}
resource "datarobot_api_token_credential" "test_chat_application" {
	name = "test chat application"
	description = "test"
	api_token = "test"
}
resource "datarobot_custom_model" "test_chat_application" {
	name = "test chat application"
	description = "test"
	source_llm_blueprint_id = "${datarobot_llm_blueprint.test_chat_application.id}"
	runtime_parameters = [
	  { 
		  key="OPENAI_API_BASE", 
		  type="string", 
		  value="https://datarobot-genai-enablement.openai.azure.com/"
	  },
	  { 
		  key="OPENAI_API_KEY", 
		  type="credential", 
		  value=datarobot_api_token_credential.test_chat_application.id
	  }
	]
}
resource "datarobot_registered_model" "test_chat_application" {
	name = "test chat application"
	description = "test"
	custom_model_version_id = "${datarobot_custom_model.test_chat_application.version_id}"
}
resource "datarobot_prediction_environment" "test_chat_application" {
	name = "test chat application"
	description = "test"
	platform = "aws"
}
resource "datarobot_deployment" "test_chat_application" {
	label = "test chat application"
	prediction_environment_id = datarobot_prediction_environment.test_chat_application.id
	registered_model_version_id = datarobot_registered_model.test_chat_application.version_id
}
resource "datarobot_chat_application" "test" {
	name = "%s"
	deployment_id = datarobot_deployment.test_chat_application.id
  }
`, name)
}

func checkChatApplicationResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetChatApplication")
		application, err := p.service.GetChatApplication(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if application.Name == rs.Primary.Attributes["name"] &&
			application.ApplicationUrl == rs.Primary.Attributes["application_url"] {
			return nil
		}

		return fmt.Errorf("Chat Application not found")
	}
}
