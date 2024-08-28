package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccCustomModelFromLlmBlueprintResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_model.test_from_llm_blueprint"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: customModelFromLlmBlueprintResourceConfig("example_name", "example_description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update name, description
			{
				Config: customModelFromLlmBlueprintResourceConfig("new_example_name", "new_example_description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
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

func TestAccCustomModelWithoutLlmBlueprintResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_model.test_without_llm_blueprint"
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	sourceRemoteRepositories := []SourceRemoteRepository{
		{
			Ref:         basetypes.NewStringValue("master"),
			SourcePaths: []basetypes.StringValue{basetypes.NewStringValue("custom_inference/python/gan_mnist/custom.py")},
		},
	}
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					"example_name",
					"example_description",
					sourceRemoteRepositories,
					[]basetypes.StringValue{basetypes.NewStringValue("custom_model_resource_test.go")},
					[]GuardConfiguration{
						{
							TemplateName: basetypes.NewStringValue("Rouge 1"),
							Name:         basetypes.NewStringValue("Rouge 1 response"),
							Stages:       []basetypes.StringValue{basetypes.NewStringValue("response")},
							Intervention: GuardIntervention{
								Action:  basetypes.NewStringValue("report"),
								Message: basetypes.NewStringValue("you have been blocked by Rouge 1"),
								Condition: GuardCondition{
									Comparand:  basetypes.NewFloat64Value(0.2),
									Comparator: basetypes.NewStringValue("lessThan"),
								},
							},
						},
					}),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttrSet(resourceName, "source_remote_repositories.0.id"),
					resource.TestCheckResourceAttr(resourceName, "source_remote_repositories.0.ref", "master"),
					resource.TestCheckResourceAttr(resourceName, "source_remote_repositories.0.source_paths.0", "custom_inference/python/gan_mnist/custom.py"),
					resource.TestCheckResourceAttr(resourceName, "local_files.0", "custom_model_resource_test.go"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.template_name", "Rouge 1"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.name", "Rouge 1 response"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.stages.0", "response"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.intervention.action", "report"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.intervention.message", "you have been blocked by Rouge 1"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.intervention.condition.comparand", "0.2"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.intervention.condition.comparator", "lessThan"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Add and update guards
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					"example_name",
					"example_description",
					sourceRemoteRepositories,
					[]basetypes.StringValue{basetypes.NewStringValue("custom_model_resource_test.go")},
					[]GuardConfiguration{
						{
							TemplateName: basetypes.NewStringValue("Faithfulness"),
							Name:         basetypes.NewStringValue("Faithfulness response"),
							Stages:       []basetypes.StringValue{basetypes.NewStringValue("response")},
							Intervention: GuardIntervention{
								Action:  basetypes.NewStringValue("block"),
								Message: basetypes.NewStringValue("you have been blocked by Faithfulness"),
								Condition: GuardCondition{
									Comparand:  basetypes.NewFloat64Value(0),
									Comparator: basetypes.NewStringValue("equals"),
								},
							},
						},
						{
							TemplateName: basetypes.NewStringValue("Token Count"),
							Name:         basetypes.NewStringValue("Token Count prompt"),
							Stages:       []basetypes.StringValue{basetypes.NewStringValue("prompt")},
							Intervention: GuardIntervention{
								Action:  basetypes.NewStringValue("block"),
								Message: basetypes.NewStringValue("you have been blocked by Token Count"),
								Condition: GuardCondition{
									Comparand:  basetypes.NewFloat64Value(10),
									Comparator: basetypes.NewStringValue("greaterThan"),
								},
							},
						},
						{
							TemplateName: basetypes.NewStringValue("Stay on topic for inputs"),
							Name:         basetypes.NewStringValue("Stay on topic for inputs"),
							Stages:       []basetypes.StringValue{basetypes.NewStringValue("prompt")},
							Intervention: GuardIntervention{
								Action:  basetypes.NewStringValue("block"),
								Message: basetypes.NewStringValue("you have been blocked by Stay on topic"),
								Condition: GuardCondition{
									Comparand:  basetypes.NewFloat64Value(10),
									Comparator: basetypes.NewStringValue("greaterThan"),
								},
							},
						},
						{
							TemplateName: basetypes.NewStringValue("Stay on topic for output"),
							Name:         basetypes.NewStringValue("Stay on topic for output"),
							Stages:       []basetypes.StringValue{basetypes.NewStringValue("response")},
							Intervention: GuardIntervention{
								Action:  basetypes.NewStringValue("block"),
								Message: basetypes.NewStringValue("you have been blocked by Stay on topic"),
								Condition: GuardCondition{
									Comparand:  basetypes.NewFloat64Value(10),
									Comparator: basetypes.NewStringValue("greaterThan"),
								},
							},
						},
					}),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.template_name", "Faithfulness"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.name", "Faithfulness response"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.stages.0", "response"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.intervention.action", "block"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.intervention.message", "you have been blocked by Faithfulness"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.intervention.condition.comparand", "0"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.intervention.condition.comparator", "equals"),
				),
			},
			// Remove guards
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					"new_example_name",
					"new_example_description",
					sourceRemoteRepositories,
					[]basetypes.StringValue{basetypes.NewStringValue("custom_model_resource_test.go")},
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckNoResourceAttr(resourceName, "guard_configurations.0.name"),
				),
			},
			// Update source remote repositories
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					"new_example_name",
					"new_example_description",
					[]SourceRemoteRepository{
						{
							Ref:         basetypes.NewStringValue("master"),
							SourcePaths: []basetypes.StringValue{basetypes.NewStringValue("custom_inference/python/gan_mnist/gan_weights.h5")},
						},
					},
					[]basetypes.StringValue{basetypes.NewStringValue("custom_model_resource_test.go")},
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "source_remote_repositories.0.id"),
					resource.TestCheckResourceAttr(resourceName, "source_remote_repositories.0.ref", "master"),
					resource.TestCheckResourceAttr(resourceName, "source_remote_repositories.0.source_paths.0", "custom_inference/python/gan_mnist/gan_weights.h5"),
				),
			},
			// Remove source remote repositories
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					"new_example_name",
					"new_example_description",
					nil,
					[]basetypes.StringValue{basetypes.NewStringValue("custom_model_resource_test.go")},
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckNoResourceAttr(resourceName, "source_remote_repositories.0.id"),
				),
			},
			// Update local files
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					"new_example_name",
					"new_example_description",
					nil,
					[]basetypes.StringValue{basetypes.NewStringValue("custom_model_resource.go")},
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "local_files.0", "custom_model_resource.go"),
				),
			},
			// Remove local files
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					"new_example_name",
					"new_example_description",
					nil,
					nil,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckNoResourceAttr(resourceName, "local_files.0"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func customModelFromLlmBlueprintResourceConfig(name, description string) string {
	return fmt.Sprintf(`
resource "datarobot_use_case" "test_custom_model" {
	name = "test custom model"
	description = "test"
}
resource "datarobot_dataset_from_file" "test_custom_model" {
	source_file = "../../test/datarobot_english_documentation_docsassist.zip"
	use_case_id = "${datarobot_use_case.test_custom_model.id}"
}
resource "datarobot_vector_database" "test_custom_model" {
	  name = "test custom model"
	  dataset_id = "${datarobot_dataset_from_file.test_custom_model.id}"
	  use_case_id = "${datarobot_use_case.test_custom_model.id}"
	  chunking_parameters = {}
}
resource "datarobot_playground" "test_custom_model" {
	name = "test custom model"
	description = "test"
	use_case_id = "${datarobot_use_case.test_custom_model.id}"
}
resource "datarobot_llm_blueprint" "test_custom_model" {
	name = "test custom model"
	description = "test"
	vector_database_id = "${datarobot_vector_database.test_custom_model.id}"
	playground_id = "${datarobot_playground.test_custom_model.id}"
	llm_id = "azure-openai-gpt-3.5-turbo"
}
resource "datarobot_api_token_credential" "test_custom_model" {
	name = "test custom model"
	description = "test"
	api_token = "test"
}
resource "datarobot_custom_model" "test_from_llm_blueprint" {
	name = "%s"
	description = "%s"
	source_llm_blueprint_id = "${datarobot_llm_blueprint.test_custom_model.id}"
	runtime_parameters = [
	  { 
		  key="OPENAI_API_BASE", 
		  type="string", 
		  value="https://datarobot-genai-enablement.openai.azure.com/"
	  },
	  { 
		  key="OPENAI_API_KEY", 
		  type="credential", 
		  value=datarobot_api_token_credential.test_custom_model.id
	  }
	]
}
`, name, description)
}

func customModelWithoutLlmBlueprintResourceConfig(
	name,
	description string,
	remoteRepositories []SourceRemoteRepository,
	localFiles []basetypes.StringValue,
	guards []GuardConfiguration,
) string {
	remoteRepositoriesStr := ""
	if len(remoteRepositories) > 0 {
		remoteRepositoriesStr = "source_remote_repositories = ["
		for _, remoteRepository := range remoteRepositories {
			remoteRepositoriesStr += fmt.Sprintf(`
			{
				id  = datarobot_remote_repository.test_custom_model_from_remote_repository.id
				ref = %s
				source_paths = %v
			},`, remoteRepository.Ref, remoteRepository.SourcePaths)
		}
		remoteRepositoriesStr += "]"
	}

	localFilesStr := ""
	if len(localFiles) > 0 {
		localFilesStr = fmt.Sprintf(`
		local_files = %v
		`, localFiles)
	}

	guardsStr := ""
	if len(guards) > 0 {
		guardsStr = "guard_configurations = ["
		for _, guard := range guards {
			guardsStr += fmt.Sprintf(`
			{
				template_name = %s
				name          = %s
				stages        = %v
				intervention = {
					action  = %s
					message = %s
					condition = {
						comparand  = %v
						comparator = %s
					}
				}
			},`, guard.TemplateName, guard.Name, guard.Stages, guard.Intervention.Action, guard.Intervention.Message, guard.Intervention.Condition.Comparand, guard.Intervention.Condition.Comparator)
		}
		guardsStr += "]"
	}

	return fmt.Sprintf(`
resource "datarobot_remote_repository" "test_custom_model_from_remote_repository" {
	name        = "Test Custom Model from Remote Repository"
	description = "test"
	location    = "https://github.com/datarobot-community/custom-models"
	source_type = "github"
	}
	
resource "datarobot_custom_model" "test_without_llm_blueprint" {
	name        = "%s"
	description = "%s"
	target_type           = "Binary"
	target                = "my_label"
	base_environment_name = "[GenAI] Python 3.11 with Moderations"
	%s
	%s
	%s
	}
`, name, description, remoteRepositoriesStr, localFilesStr, guardsStr)
}

func checkCustomModelResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetCustomModel")
		customModel, err := p.service.GetCustomModel(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if customModel.Name == rs.Primary.Attributes["name"] &&
			customModel.Description == rs.Primary.Attributes["description"] {
			return nil
		}

		return fmt.Errorf("Custom Model not found")
	}
}
