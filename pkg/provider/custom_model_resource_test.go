package provider

import (
	"context"
	"fmt"
	"strings"
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
					},
					nil,
					false),
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
					resource.TestCheckResourceAttr(resourceName, "target_name", "document"),
					resource.TestCheckResourceAttr(resourceName, "language", "Python"),
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
					resource.TestCheckResourceAttr(resourceName, "resource_settings.memory_mb", "2048"),
					resource.TestCheckResourceAttr(resourceName, "resource_settings.replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "resource_settings.network_access", "PUBLIC"),
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
					},
					nil,
					false),
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
			// // Remove guards
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					"new_example_name",
					"new_example_description",
					sourceRemoteRepositories,
					[]basetypes.StringValue{basetypes.NewStringValue("custom_model_resource_test.go")},
					nil,
					nil,
					false),
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
					nil,
					nil,
					false),
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
					nil,
					nil,
					false),
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
			// // Update local files
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					"new_example_name",
					"new_example_description",
					nil,
					[]basetypes.StringValue{basetypes.NewStringValue("custom_model_resource.go")},
					nil,
					nil,
					false),
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
					nil,
					nil,
					false),
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
			// Add resource settings
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					"new_example_name",
					"new_example_description",
					nil,
					nil,
					nil,
					&CustomModelResourceSettings{
						MemoryMB:      basetypes.NewInt64Value(256),
						Replicas:      basetypes.NewInt64Value(2),
						NetworkAccess: basetypes.NewStringValue("NONE"),
					},
					false),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "resource_settings.memory_mb", "256"),
					resource.TestCheckResourceAttr(resourceName, "resource_settings.replicas", "2"),
					resource.TestCheckResourceAttr(resourceName, "resource_settings.network_access", "NONE"),
				),
			},
			// Add training dataset
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					"new_example_name",
					"new_example_description",
					sourceRemoteRepositories,
					nil,
					nil,
					&CustomModelResourceSettings{
						MemoryMB:      basetypes.NewInt64Value(256),
						Replicas:      basetypes.NewInt64Value(2),
						NetworkAccess: basetypes.NewStringValue("NONE"),
					},
					true),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "training_dataset_id"),
					resource.TestCheckResourceAttrSet(resourceName, "training_dataset_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "training_dataset_name"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestAccBinaryCustomModelResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_model.test_binary"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: binaryCustomModelResourceConfig("example_name", "target", "python", "1", "0", 0.5),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "target"),
					resource.TestCheckResourceAttr(resourceName, "positive_class_label", "1"),
					resource.TestCheckResourceAttr(resourceName, "negative_class_label", "0"),
					resource.TestCheckResourceAttr(resourceName, "language", "python"),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", "0.5"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update parameters
			{
				Config: binaryCustomModelResourceConfig("new_example_name", "new_target", "r", "yes", "no", 0.8),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "positive_class_label", "yes"),
					resource.TestCheckResourceAttr(resourceName, "negative_class_label", "no"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", "0.8"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestAccMulticlassCustomModelResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_model.test_multiclass"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: multiclassCustomModelResourceConfig("example_name", "target", "python", []string{"class1", "class2", "class3"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "target"),
					resource.TestCheckResourceAttr(resourceName, "language", "python"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.0", "class1"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.1", "class2"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.2", "class3"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update parameters
			{
				Config: multiclassCustomModelResourceConfig("new_example_name", "new_target", "r", []string{"class1", "class8", "class3"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.0", "class1"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.1", "class8"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.2", "class3"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestAccRegressionCustomModelResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_model.test_regression"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: regressionCustomModelResourceConfig("example_name", "target", "python"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "target"),
					resource.TestCheckResourceAttr(resourceName, "language", "python"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update parameters
			{
				Config: regressionCustomModelResourceConfig("new_example_name", "new_target", "r"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestAccTextGenerationCustomModelResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_model.test_text_generation"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: textGenerationCustomModelResourceConfig("example_name", "target", "python"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "target"),
					resource.TestCheckResourceAttr(resourceName, "language", "python"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update parameters
			{
				Config: textGenerationCustomModelResourceConfig("new_example_name", "new_target", "r"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestAccUnstructuredCustomModelResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_model.test_unstructured"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: unstructuredCustomModelResourceConfig("example_name", "python"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "language", "python"),
					resource.TestCheckResourceAttr(resourceName, "deployments_count", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update parameters
			{
				Config: unstructuredCustomModelResourceConfig("new_example_name", "r"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckResourceAttr(resourceName, "deployments_count", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestCustomModelResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewCustomModelResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
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
	runtime_parameter_values = [
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
	resourceSettings *CustomModelResourceSettings,
	addTrainingData bool,
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

	resourceSettingsStr := ""
	if resourceSettings != nil {
		resourceSettingsStr = fmt.Sprintf(`
		resource_settings = {
			memory_mb	    = %d
			replicas 	    = %d
			network_access  = %s
		}
		`, resourceSettings.MemoryMB.ValueInt64(), resourceSettings.Replicas.ValueInt64(), resourceSettings.NetworkAccess)
	}

	trainingDatasetStr := ""
	if addTrainingData {
		trainingDatasetStr = `
		training_dataset_id = "${datarobot_dataset_from_file.test_without_llm_blueprint.id}"
		`
	}

	return fmt.Sprintf(`
resource "datarobot_use_case" "test_without_llm_blueprint" {
	name = "test custom model without llm blueprint"
}

resource "datarobot_dataset_from_file" "test_without_llm_blueprint" {
	source_file = "../../test/datarobot_english_documentation_docsassist.zip"
	use_case_id = "${datarobot_use_case.test_without_llm_blueprint.id}"
}

resource "datarobot_remote_repository" "test_custom_model_from_remote_repository" {
	name        = "Test Custom Model from Remote Repository"
	description = "test"
	location    = "https://github.com/datarobot-community/custom-models"
	source_type = "github"
}
	
resource "datarobot_custom_model" "test_without_llm_blueprint" {
	name        		  = "%s"
	description 		  = "%s"
	target_type           = "TextGeneration"
	target_name           = "document"
	language 			  = "Python"
	base_environment_name = "[GenAI] Python 3.11 with Moderations"
	%s
	%s
	%s
	%s
	%s
}
`, name, description, remoteRepositoriesStr, localFilesStr, guardsStr, resourceSettingsStr, trainingDatasetStr)
}

func binaryCustomModelResourceConfig(
	name,
	targetName,
	language,
	positiveClassLabel,
	negativeClassLabel string,
	predictionThreshold float64) string {
	resourceBlock, customModelBlock := remoteRepositoryResource("test_custom_model_binary")

	return fmt.Sprintf(`
%s

resource "datarobot_custom_model" "test_binary" {
	name        		  = "%s"
	target_type           = "Binary"
	target_name           = "%s"
	language 			  = "%s"
	positive_class_label  = "%s"
	negative_class_label  = "%s"
	prediction_threshold  = %f
	base_environment_name = "[GenAI] Python 3.11 with Moderations"
	%s
}
`, resourceBlock, name, targetName, language, positiveClassLabel, negativeClassLabel, predictionThreshold, customModelBlock)
}

func multiclassCustomModelResourceConfig(
	name,
	targetName,
	language string,
	classLabels []string) string {
	resourceBlock, customModelBlock := remoteRepositoryResource("test_custom_model_multiclass")

	return fmt.Sprintf(`
%s

resource "datarobot_custom_model" "test_multiclass" {
	name        		  							  = "%s"
	target_type           							  = "Multiclass"
	target_name           							  = "%s"
	language 			  							  = "%s"
	class_labels  		  							  = [%s]
	base_environment_name 							  = "[GenAI] Python 3.11 with Moderations"
	is_proxy 										  = true
	%s
}
`, resourceBlock, name, targetName, language, "\""+strings.Join(classLabels, "\",\"")+"\"", customModelBlock)
}

func regressionCustomModelResourceConfig(
	name,
	targetName,
	language string) string {
	resourceBlock, customModelBlock := remoteRepositoryResource("test_custom_model_regression")

	return fmt.Sprintf(`
%s

resource "datarobot_custom_model" "test_regression" {
	name        		  							  = "%s"
	target_type           							  = "Regression"
	target_name           							  = "%s"
	language 			  							  = "%s"
	base_environment_name 							  = "[GenAI] Python 3.11 with Moderations"
	%s
}
`, resourceBlock, name, targetName, language, customModelBlock)
}

func textGenerationCustomModelResourceConfig(
	name,
	targetName,
	language string) string {
	resourceBlock, customModelBlock := remoteRepositoryResource("test_custom_model_text_generation")

	return fmt.Sprintf(`
%s

resource "datarobot_custom_model" "test_text_generation" {
	name        		  							  = "%s"
	target_type           							  = "TextGeneration"
	target_name           							  = "%s"
	language 			  							  = "%s"
	base_environment_name 							  = "[GenAI] Python 3.11 with Moderations"
	is_proxy 										  = true
	%s
}
`, resourceBlock, name, targetName, language, customModelBlock)
}

func unstructuredCustomModelResourceConfig(
	name,
	language string) string {
	resourceBlock, customModelBlock := remoteRepositoryResource("test_custom_model_unstructured")

	return fmt.Sprintf(`
%s

resource "datarobot_custom_model" "test_unstructured" {
	name        		  							  = "%s"
	target_type           							  = "Unstructured"
	language 			  							  = "%s"
	base_environment_name 							  = "[GenAI] Python 3.11 with Moderations"
	%s
}
`, resourceBlock, name, language, customModelBlock)
}

func remoteRepositoryResource(resourceName string) (string, string) {
	resourceBlock := fmt.Sprintf(`
resource "datarobot_remote_repository" "%s" {
	name        = "Test Custom Model from Remote Repository"
	description = "test"
	location    = "https://github.com/datarobot-community/custom-models"
	source_type = "github"
}
		`, resourceName)

	customModelBlock := fmt.Sprintf(`
	source_remote_repositories = [
		{
			id  = datarobot_remote_repository.%s.id
			ref = "master"
			source_paths = ["custom_inference/python/gan_mnist/custom.py"]
		}
	]
	`, resourceName)

	return resourceBlock, customModelBlock
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
