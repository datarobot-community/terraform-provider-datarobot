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

func TestAccLLMBlueprintResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_llm_blueprint.test"
	llmID := "azure-openai-gpt-3.5-turbo"
	newLLMID := "amazon-titan"
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: llmBlueprintResourceConfig("example_name", "example_description", llmID),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkLlmBlueprintResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, description, and LLM ID
			{
				Config: llmBlueprintResourceConfig("new_example_name", "new_example_description", newLLMID),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkLlmBlueprintResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "llm_id", newLLMID),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func llmBlueprintResourceConfig(name, description, llmID string) string {
	return fmt.Sprintf(`
resource "datarobot_use_case" "test_llm_blueprint" {
	name = "test"
}
resource "datarobot_playground" "test_llm_blueprint" {
	name = "llm blueprint test"
	description = "test"
	use_case_id = "${datarobot_use_case.test_llm_blueprint.id}"
}
resource "datarobot_llm_blueprint" "test" {
	name = "%s"
	description = "%s"
	playground_id = "${datarobot_playground.test_llm_blueprint.id}"
	llm_id = "%s"
}
`, name, description, llmID)
}

func checkLlmBlueprintResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetLLMBlueprint")
		llmBlueprint, err := p.service.GetLLMBlueprint(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if llmBlueprint.Name == rs.Primary.Attributes["name"] &&
			llmBlueprint.Description == rs.Primary.Attributes["description"] &&
			llmBlueprint.VectorDatabaseID == rs.Primary.Attributes["vector_database_id"] &&
			llmBlueprint.PlaygroundID == rs.Primary.Attributes["playground_id"] &&
			llmBlueprint.LLMID == rs.Primary.Attributes["llm_id"] {
			return nil
		}

		return fmt.Errorf("LLM Blueprint not found")
	}
}
