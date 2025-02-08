package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

	externalLLMContextSize := 100
	systemPrompt := "Custom Model Prompt: "

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: llmBlueprintResourceConfig(
					"example_name",
					"example_description",
					llmID,
					nil,
					nil),
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
					resource.TestCheckResourceAttr(resourceName, "prompt_type", "CHAT_HISTORY_AWARE"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, description, and LLM ID
			{
				Config: llmBlueprintResourceConfig(
					"new_example_name",
					"new_example_description",
					newLLMID,
					nil,
					nil),
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
					resource.TestCheckResourceAttr(resourceName, "prompt_type", "CHAT_HISTORY_AWARE"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update LLM Settings
			{
				Config: llmBlueprintResourceConfig(
					"new_example_name",
					"new_example_description",
					newLLMID,
					&LLMSettings{
						MaxCompletionLength: basetypes.NewInt64Value(1000),
						Temperature:         basetypes.NewFloat64Value(0.5),
						TopP:                basetypes.NewFloat64Value(0.5),
						SystemPrompt:        basetypes.NewStringValue("Prompt:"),
					},
					nil),
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
					resource.TestCheckResourceAttr(resourceName, "llm_settings.max_completion_length", "1000"),
					resource.TestCheckResourceAttr(resourceName, "llm_settings.temperature", "0.5"),
					resource.TestCheckResourceAttr(resourceName, "llm_settings.top_p", "0.5"),
					resource.TestCheckResourceAttr(resourceName, "llm_settings.system_prompt", "Prompt:"),
					resource.TestCheckResourceAttr(resourceName, "prompt_type", "CHAT_HISTORY_AWARE"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update Custom Model LLM Settings
			{
				Config: llmBlueprintResourceConfig(
					"new_example_name",
					"new_example_description",
					newLLMID,
					&LLMSettings{
						MaxCompletionLength: basetypes.NewInt64Value(1000),
						Temperature:         basetypes.NewFloat64Value(0.5),
						TopP:                basetypes.NewFloat64Value(0.5),
						SystemPrompt:        basetypes.NewStringValue("Prompt:"),
					},
					&CustomModelLLMSettings{
						ExternalLLMContextSize: basetypes.NewInt64Value(int64(externalLLMContextSize)),
						SystemPrompt:           basetypes.NewStringValue(systemPrompt),
					}),
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
					resource.TestCheckResourceAttr(resourceName, "llm_settings.max_completion_length", "1000"),
					resource.TestCheckResourceAttr(resourceName, "llm_settings.temperature", "0.5"),
					resource.TestCheckResourceAttr(resourceName, "llm_settings.top_p", "0.5"),
					resource.TestCheckResourceAttr(resourceName, "llm_settings.system_prompt", "Prompt:"),
					resource.TestCheckResourceAttr(resourceName, "custom_model_llm_settings.system_prompt", systemPrompt),
					resource.TestCheckResourceAttr(resourceName, "custom_model_llm_settings.external_llm_context_size", "100"),
					resource.TestCheckResourceAttr(resourceName, "prompt_type", "CHAT_HISTORY_AWARE"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestLLMBlueprintResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewLLMBlueprintResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func llmBlueprintResourceConfig(
	name,
	description,
	llmID string,
	llmSettings *LLMSettings,
	customModelLLMSettings *CustomModelLLMSettings,
) string {
	llmSettingsStr := ""
	if llmSettings != nil {
		llmSettingsStr = fmt.Sprintf(`
		llm_settings = {
			max_completion_length = %d
			temperature = %f
			top_p = %f
			system_prompt = "%s"
		}`,
			llmSettings.MaxCompletionLength.ValueInt64(),
			llmSettings.Temperature.ValueFloat64(),
			llmSettings.TopP.ValueFloat64(),
			llmSettings.SystemPrompt.ValueString())
	}

	customModelLLMSettingsStr := ""
	if customModelLLMSettings != nil {
		customModelLLMSettingsStr = fmt.Sprintf(`
		custom_model_llm_settings = {
			external_llm_context_size = %d
			system_prompt = "%s"
		}`,
			customModelLLMSettings.ExternalLLMContextSize.ValueInt64(),
			customModelLLMSettings.SystemPrompt.ValueString())
	}

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
	%s
	%s
}
`, name,
		description,
		llmID,
		llmSettingsStr,
		customModelLLMSettingsStr)
}

func checkLlmBlueprintResourceExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_llm_blueprint.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_llm_blueprint.test")
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
			*llmBlueprint.LLMID == rs.Primary.Attributes["llm_id"] &&
			llmBlueprint.PromptType == rs.Primary.Attributes["prompt_type"] {
			return nil
		}

		return fmt.Errorf("LLM Blueprint not found")
	}
}
