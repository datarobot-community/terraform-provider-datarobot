package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccCustomModelLLMValidationResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_model_llm_validation.test"

	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())

	folderPath := "custom_model_llm_validation"
	if err := os.Mkdir(folderPath, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(folderPath)

	modelContents := `import pandas as pd

def load_model(code_dir):
	return True

def score(data, model, **kwargs):
	return pd.DataFrame({"answer": ["answer"]})`

	if err := os.WriteFile(folderPath+"/custom.py", []byte(modelContents), 0644); err != nil {
		t.Fatal(err)
	}

	name := "example_name " + nameSalt
	newName := "new_example_name " + nameSalt

	promptColumnName := "promptText"
	newPromptColumnName := "newPromptText"

	targetColumnName := "resultText"

	predictionTimeout := 100
	newPredictionTimeout := 200

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
				Config: customModelLLMValidationResourceConfig(
					name,
					nil,
					&promptColumnName,
					&targetColumnName,
					predictionTimeout,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelLLMValidationResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckNoResourceAttr(resourceName, "chat_model_id"),
					resource.TestCheckResourceAttr(resourceName, "prompt_column_name", promptColumnName),
					resource.TestCheckResourceAttr(resourceName, "target_column_name", targetColumnName),
					resource.TestCheckResourceAttr(resourceName, "prediction_timeout", "100"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, promptColumnName, predictionTimeout
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: customModelLLMValidationResourceConfig(
					newName,
					nil,
					&newPromptColumnName,
					&targetColumnName,
					newPredictionTimeout,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelLLMValidationResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckNoResourceAttr(resourceName, "chat_model_id"),
					resource.TestCheckResourceAttr(resourceName, "prompt_column_name", newPromptColumnName),
					resource.TestCheckResourceAttr(resourceName, "target_column_name", targetColumnName),
					resource.TestCheckResourceAttr(resourceName, "prediction_timeout", "200"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func customModelLLMValidationResourceConfig(
	name string,
	chatModelID,
	promptColumnName,
	targetColumnName *string,
	predictionTimeout int,
) string {
	chatModelIDStr := ""
	if chatModelID != nil {
		chatModelIDStr = fmt.Sprintf(`chat_model_id = "%s"`, *chatModelID)
	}

	promptColumnNameStr := ""
	if promptColumnName != nil {
		promptColumnNameStr = fmt.Sprintf(`prompt_column_name = "%s"`, *promptColumnName)
	}

	targetColumnNameStr := ""
	if targetColumnName != nil {
		targetColumnNameStr = fmt.Sprintf(`target_column_name = "%s"`, *targetColumnName)
	}

	return fmt.Sprintf(`
resource "datarobot_use_case" "test_custom_model_llm_validation" {
	name = "test custom model llm validation"
}
resource "datarobot_playground" "test_custom_model_llm_validation" {
	name = "llm validation test"
	description = "test"
	use_case_id = "${datarobot_use_case.test_custom_model_llm_validation.id}"
}
resource "datarobot_custom_model" "test_custom_model_llm_validation" {
	name = "test custom model llm validation"
	description = "test"
	target_type = "TextGeneration"
	target_name = "resultText"
	base_environment_id = "65f9b27eab986d30d4c64268"
	folder_path = "custom_model_llm_validation"
}
resource "datarobot_registered_model" "test_custom_model_llm_validation" {
	name = "test custom model llm validation %s"
	custom_model_version_id = "${datarobot_custom_model.test_custom_model_llm_validation.version_id}"
}
resource "datarobot_prediction_environment" "test_custom_model_llm_validation" {
	name = "test custom model llm validation"
	description = "test"
	platform = "datarobotServerless"
}
resource "datarobot_deployment" "test_custom_model_llm_validation" {
	label = "test custom model llm validation"
	importance = "LOW"
	prediction_environment_id = datarobot_prediction_environment.test_custom_model_llm_validation.id
	registered_model_version_id = datarobot_registered_model.test_custom_model_llm_validation.version_id
}
resource "datarobot_custom_model_llm_validation" "test" {
	name = "%s"
	deployment_id = datarobot_deployment.test_custom_model_llm_validation.id
	%s
	%s
	%s
	prediction_timeout = %d
	use_case_id = datarobot_use_case.test_custom_model_llm_validation.id
}
resource "datarobot_llm_blueprint" "test_custom_model_llm_validation" {
	name = "test custom model llm validation"
	playground_id = "${datarobot_playground.test_custom_model_llm_validation.id}"
	llm_id = "custom-model"
}
`, nameSalt,
		name,
		chatModelIDStr,
		promptColumnNameStr,
		targetColumnNameStr,
		predictionTimeout)
}

func checkCustomModelLLMValidationResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_custom_model_llm_validation.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_custom_model_llm_validation.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = NewService(cl)

		traceAPICall("GetCustomModelLLMValidation")
		customModelLLMValidation, err := p.service.GetCustomModelLLMValidation(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if customModelLLMValidation.Name == rs.Primary.Attributes["name"] &&
			customModelLLMValidation.PromptColumnName == rs.Primary.Attributes["prompt_column_name"] &&
			customModelLLMValidation.TargetColumnName == rs.Primary.Attributes["target_column_name"] &&
			customModelLLMValidation.UseCaseID == rs.Primary.Attributes["use_case_id"] &&
			customModelLLMValidation.DeploymentID == rs.Primary.Attributes["deployment_id"] &&
			customModelLLMValidation.ModelID == rs.Primary.Attributes["model_id"] {
			return nil
		}

		return fmt.Errorf("Custom Model LLM Validation not found")
	}
}
