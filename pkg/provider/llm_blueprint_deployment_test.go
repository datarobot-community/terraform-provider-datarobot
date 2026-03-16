package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccLLMBlueprintDeployment(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_deployment.llm_test"
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	llmID := "azure-openai-gpt-4-o"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create full chain: LLM Blueprint → Custom Model → Registered Model → Deployment
			{
				Config: llmBlueprintDeploymentConfig(llmID, "potato"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "registered_model_version_id"),
				),
			},
			// Update LLM Blueprint system_prompt - this triggers replacement which cascades down
			// This tests that the provider handles Terraform's non-deterministic deletion order
			// See: https://github.com/hashicorp/terraform/issues/37975 and #30439
			{
				Config: llmBlueprintDeploymentConfig(llmID, "tomato"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("registered_model_version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "registered_model_version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func llmBlueprintDeploymentConfig(llmID string, systemPrompt string) string {
	return fmt.Sprintf(`
resource "datarobot_use_case" "llm_test" {
	name = "llm blueprint deployment test %s"
}

resource "datarobot_playground" "llm_test" {
	name = "llm blueprint deployment test %s"
	use_case_id = datarobot_use_case.llm_test.id
}

resource "datarobot_llm_blueprint" "llm_test" {
	name = "llm blueprint deployment test %s"
	playground_id = datarobot_playground.llm_test.id
	llm_id = "%s"
	llm_settings = {
		system_prompt = "%s"
	}
}

resource "datarobot_custom_model" "llm_test" {
	name = "llm blueprint deployment test %s"
	target_type = "TextGeneration"
	target_name = "resultText"
	source_llm_blueprint_id = datarobot_llm_blueprint.llm_test.id
	base_environment_id = "`+testGenAIBaseEnvID+`"
}

resource "datarobot_registered_model" "llm_test" {
	name = "llm blueprint deployment test %s"
	custom_model_version_id = datarobot_custom_model.llm_test.version_id
	use_case_ids = [datarobot_use_case.llm_test.id]
}

resource "datarobot_prediction_environment" "llm_test" {
	name = "llm blueprint deployment test %s"
	platform = "datarobotServerless"
}

resource "datarobot_deployment" "llm_test" {
	label = "llm blueprint deployment test %s"
	registered_model_version_id = datarobot_registered_model.llm_test.version_id
	prediction_environment_id = datarobot_prediction_environment.llm_test.id
	predictions_settings = {
		min_computes = 0
		max_computes = 2
	}
}
`, nameSalt, nameSalt, nameSalt, llmID, systemPrompt, nameSalt, nameSalt, nameSalt, nameSalt)
}
