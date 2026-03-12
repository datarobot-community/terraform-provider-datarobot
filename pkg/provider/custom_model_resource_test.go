package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

type CustomModelResourceSettings struct {
	MemoryMB      basetypes.Int64Value  `json:"memory_mb,omitempty"`
	Replicas      basetypes.Int64Value  `json:"replicas,omitempty"`
	NetworkAccess basetypes.StringValue `json:"network_access,omitempty"`
}

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
				Config: customModelFromLlmBlueprintResourceConfig("example_name", "example_description", nameSalt),
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
				Config: customModelFromLlmBlueprintResourceConfig("new_example_name", "new_example_description", nameSalt),
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

	resourceType := "datarobot_custom_model"
	resourceTestName := "test_without_llm_blueprint"
	resourceName := resourceType + "." + resourceTestName

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	baseEnvironmentID := testGenAIBaseEnvID
	baseEnvironmentID2 := testStreamlitBaseEnvID

	fileName := "requirements.txt"
	folderPath := "custom_model_without_llm_blueprint"
	err := os.WriteFile(fileName, []byte(`langchain == 0.2.8`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fileName)

	metadataFileName := "model-metadata.yaml"
	metadataContents := `name: test-model
runtimeParameterDefinitions:
  - fieldName: GUARD_CONFIG_PLACEHOLDER
    type: string
    description: Placeholder for guard configuration schema
    defaultValue: null
  - fieldName: MODERATION_OOTB_RESPONSE_FAITHFULNESS_AZURE_OPENAI_API_KEY
    type: credential
    description: Azure OpenAI API key for Faithfulness guard
    defaultValue: null
  - fieldName: MODERATION_NEMO_GUARDRAILS_PROMPT_AZURE_OPENAI_API_KEY
    type: credential
    description: Azure OpenAI API key for NeMo guard
    defaultValue: null
  - fieldName: MODERATION_OOTB_RESPONSE_FAITHFULNESS_OPENAI_API_KEY
    type: credential
    description: OpenAI API key for Faithfulness guard
    defaultValue: null
`
	err = os.WriteFile(metadataFileName, []byte(metadataContents), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(metadataFileName)

	os.RemoveAll(folderPath)
	err = os.Mkdir(folderPath, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(folderPath)

	err = os.WriteFile(folderPath+"/"+fileName, []byte(`langchain == 0.2.9`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(folderPath+"/"+metadataFileName, []byte(metadataContents), 0644)
	if err != nil {
		t.Fatal(err)
	}

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
					resourceTestName,
					"example_name",
					"example_description",
					baseEnvironmentID,
					sourceRemoteRepositories,
					nil,
					[]FileTuple{{LocalPath: fileName}, {LocalPath: metadataFileName}},
					[]GuardConfiguration{
						{
							TemplateName: basetypes.NewStringValue("Rouge 1"),
							Name:         basetypes.NewStringValue("Rouge 1 response"),
							Stages:       []basetypes.StringValue{basetypes.NewStringValue("response")},
							Intervention: GuardIntervention{
								Action:    basetypes.NewStringValue("report"),
								Message:   basetypes.NewStringValue("you have been blocked by Rouge 1"),
								Condition: basetypes.NewStringValue(`{"comparand": 0.2, "comparator": "lessThan"}`),
							},
						},
						{
							TemplateName: basetypes.NewStringValue("Faithfulness"),
							Name:         basetypes.NewStringValue("Faithfulness response"),
							Stages:       []basetypes.StringValue{basetypes.NewStringValue("response")},
							Intervention: GuardIntervention{
								Action:    basetypes.NewStringValue("block"),
								Message:   basetypes.NewStringValue("you have been blocked by Faithfulness"),
								Condition: basetypes.NewStringValue(`{"comparand": 0, "comparator": "equals"}`),
							},
							OpenAICredential:   basetypes.NewStringValue("test"),
							OpenAIApiBase:      basetypes.NewStringValue("https://datarobot-genai-enablement.openai.azure.com/"),
							OpenAIDeploymentID: basetypes.NewStringValue("test"),
							LlmType:            basetypes.NewStringValue("azureOpenAi"),
						},
						{
							TemplateName: basetypes.NewStringValue("Emotions Classifier"),
							Name:         basetypes.NewStringValue("Emotions Classifier response"),
							Stages:       []basetypes.StringValue{basetypes.NewStringValue("response")},
							Intervention: GuardIntervention{
								Action:    basetypes.NewStringValue("block"),
								Message:   basetypes.NewStringValue("you have been blocked by Emotions Classifier"),
								Condition: basetypes.NewStringValue(`{"comparand": ["anger", "amusement"], "comparator": "matches"}`),
							},
						},
						{
							TemplateName: basetypes.NewStringValue("Stay on topic for inputs"),
							Name:         basetypes.NewStringValue("Stay on topic for inputs"),
							Stages:       []basetypes.StringValue{basetypes.NewStringValue("prompt")},
							Intervention: GuardIntervention{
								Action:    basetypes.NewStringValue("block"),
								Message:   basetypes.NewStringValue("you have been blocked by Nemo"),
								Condition: basetypes.NewStringValue(`{"comparand": "TRUE", "comparator": "equals"}`),
							},
							OpenAICredential:   basetypes.NewStringValue("test"),
							OpenAIApiBase:      basetypes.NewStringValue("https://datarobot-genai-enablement.openai.azure.com/"),
							OpenAIDeploymentID: basetypes.NewStringValue("test"),
							LlmType:            basetypes.NewStringValue("azureOpenAi"),
							NemoInfo: &NemoInfo{
								BlockedTerms: basetypes.NewStringValue("term1\nterm2\nterm3\n"),
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
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "target_name", "document"),
					resource.TestCheckResourceAttr(resourceName, "language", "Python"),
					resource.TestCheckResourceAttrSet(resourceName, "source_remote_repositories.0.id"),
					resource.TestCheckResourceAttr(resourceName, "source_remote_repositories.0.ref", "master"),
					resource.TestCheckResourceAttr(resourceName, "source_remote_repositories.0.source_paths.0", "custom_inference/python/gan_mnist/custom.py"),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", fileName),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.template_name", "Rouge 1"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.name", "Rouge 1 response"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.stages.0", "response"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.intervention.action", "report"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.intervention.message", "you have been blocked by Rouge 1"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.intervention.condition", `{"comparand":0.2,"comparator":"lessThan"}`),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.1.template_name", "Faithfulness"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.1.intervention.condition", `{"comparand":0,"comparator":"equals"}`),
					resource.TestCheckResourceAttrSet(resourceName, "guard_configurations.1.openai_credential"),
					resource.TestCheckResourceAttrSet(resourceName, "guard_configurations.1.openai_api_base"),
					resource.TestCheckResourceAttrSet(resourceName, "guard_configurations.1.openai_deployment_id"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.1.llm_type", "azureOpenAi"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.2.template_name", "Emotions Classifier"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.2.intervention.condition", `{"comparand":["anger","amusement"],"comparator":"matches"}`),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.3.template_name", "Stay on topic for inputs"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.3.intervention.condition", `{"comparand":"TRUE","comparator":"equals"}`),
					resource.TestCheckResourceAttrSet(resourceName, "guard_configurations.3.openai_credential"),
					resource.TestCheckResourceAttrSet(resourceName, "guard_configurations.3.openai_api_base"),
					resource.TestCheckResourceAttrSet(resourceName, "guard_configurations.3.openai_deployment_id"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.3.llm_type", "azureOpenAi"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.3.nemo_info.blocked_terms", "term1\nterm2\nterm3\n"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Add and update guards + update base environment
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					resourceTestName,
					"example_name",
					"example_description",
					baseEnvironmentID2,
					sourceRemoteRepositories,
					nil,
					[]FileTuple{{LocalPath: fileName, PathInModel: "new_dir/" + fileName}, {LocalPath: metadataFileName}},
					[]GuardConfiguration{
						{
							TemplateName: basetypes.NewStringValue("Faithfulness"),
							Name:         basetypes.NewStringValue("Faithfulness response"),
							Stages:       []basetypes.StringValue{basetypes.NewStringValue("response")},
							Intervention: GuardIntervention{
								Action:    basetypes.NewStringValue("block"),
								Message:   basetypes.NewStringValue("you have been blocked by Faithfulness"),
								Condition: basetypes.NewStringValue(`{"comparand": 0, "comparator": "equals"}`),
							},
							OpenAICredential: basetypes.NewStringValue("test"),
							LlmType:          basetypes.NewStringValue("openAi"),
						},
						{
							TemplateName: basetypes.NewStringValue("Prompt Tokens"),
							Name:         basetypes.NewStringValue("prompt tokens"),
							Stages:       []basetypes.StringValue{basetypes.NewStringValue("prompt")},
							Intervention: GuardIntervention{
								Action:    basetypes.NewStringValue("block"),
								Message:   basetypes.NewStringValue("you have been blocked by prompt token count"),
								Condition: basetypes.NewStringValue(`{"comparand": 10, "comparator": "greaterThan"}`),
							},
						},
						{
							TemplateName: basetypes.NewStringValue("Stay on topic for inputs"),
							Name:         basetypes.NewStringValue("Stay on topic for inputs"),
							Stages:       []basetypes.StringValue{basetypes.NewStringValue("prompt")},
							Intervention: GuardIntervention{
								Action:    basetypes.NewStringValue("block"),
								Message:   basetypes.NewStringValue("you have been blocked by Stay on topic"),
								Condition: basetypes.NewStringValue(`{"comparand": 10, "comparator": "greaterThan"}`),
							},
						},
						{
							TemplateName: basetypes.NewStringValue("Stay on topic for output"),
							Name:         basetypes.NewStringValue("Stay on topic for output"),
							Stages:       []basetypes.StringValue{basetypes.NewStringValue("response")},
							Intervention: GuardIntervention{
								Action:    basetypes.NewStringValue("block"),
								Message:   basetypes.NewStringValue("you have been blocked by Stay on topic"),
								Condition: basetypes.NewStringValue(`{"comparand": 10, "comparator": "greaterThan"}`),
							},
						},
						{
							TemplateName: basetypes.NewStringValue("Cost"),
							Name:         basetypes.NewStringValue("Cost Response"),
							Stages:       []basetypes.StringValue{basetypes.NewStringValue("response")},
							Intervention: GuardIntervention{
								Action:    basetypes.NewStringValue("report"),
								Message:   basetypes.NewStringValue("Unused"),
								Condition: basetypes.NewStringValue(`{"comparand": "ignore", "comparator": "is"}`),
							},
							AdditionalGuardConfig: &AdditionalGuardConfig{
								Cost: GuardCostInfo{
									Currency:    basetypes.NewStringValue("USD"),
									InputPrice:  basetypes.NewFloat64Value(0.001),
									InputUnit:   basetypes.NewInt64Value(1000),
									OutputPrice: basetypes.NewFloat64Value(0.01),
									OutputUnit:  basetypes.NewInt64Value(1000),
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
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID2),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.template_name", "Faithfulness"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.name", "Faithfulness response"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.stages.0", "response"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.intervention.action", "block"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.intervention.message", "you have been blocked by Faithfulness"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.intervention.condition", `{"comparand":0,"comparator":"equals"}`),
					resource.TestCheckResourceAttrSet(resourceName, "guard_configurations.0.openai_credential"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.0.llm_type", "openAi"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.4.template_name", "Cost"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.4.name", "Cost Response"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.4.stages.0", "response"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.4.additional_guard_config.cost.currency", "USD"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.4.additional_guard_config.cost.input_price", "0.001"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.4.additional_guard_config.cost.input_unit", "1000"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.4.additional_guard_config.cost.output_price", "0.01"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.4.additional_guard_config.cost.output_unit", "1000"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.4.intervention.action", "report"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.4.intervention.message", "Unused"),
					resource.TestCheckResourceAttr(resourceName, "guard_configurations.4.intervention.condition", `{"comparand":"ignore","comparator":"is"}`),

					resource.TestCheckResourceAttr(resourceName, "files.0.1", "new_dir/"+fileName),
				),
			},
			// Remove guards
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					resourceTestName,
					"new_example_name",
					"new_example_description",
					baseEnvironmentID2,
					sourceRemoteRepositories,
					nil,
					[]FileTuple{{LocalPath: fileName}},
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
					resourceTestName,
					"new_example_name",
					"new_example_description",
					baseEnvironmentID2,
					[]SourceRemoteRepository{
						{
							Ref:         basetypes.NewStringValue("master"),
							SourcePaths: []basetypes.StringValue{basetypes.NewStringValue("custom_inference/python/gan_mnist/gan_weights.h5")},
						},
					},
					nil,
					[]FileTuple{{LocalPath: fileName}},
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
					resourceTestName,
					"new_example_name",
					"new_example_description",
					baseEnvironmentID2,
					nil,
					nil,
					[]FileTuple{{LocalPath: fileName}},
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
			// Update files, base environment, and rebuild dependencies
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					resourceTestName,
					"new_example_name",
					"new_example_description",
					baseEnvironmentID,
					nil,
					nil,
					[]FileTuple{{LocalPath: folderPath + "/" + fileName}},
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
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", folderPath+"/"+fileName),
				),
			},
			// Remove files, add folder path
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					resourceTestName,
					"new_example_name",
					"new_example_description",
					baseEnvironmentID,
					nil,
					&folderPath,
					nil,
					nil,
					nil,
					false),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("folder_path_hash"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckNoResourceAttr(resourceName, "files.0.0"),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttrSet(resourceName, "folder_path_hash"),
				),
			},
			// Add file in folder path
			{
				PreConfig: func() {
					if err := os.WriteFile(folderPath+"/newfile.txt", []byte("contents..."), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: customModelWithoutLlmBlueprintResourceConfig(
					resourceTestName,
					"new_example_name",
					"new_example_description",
					baseEnvironmentID,
					nil,
					&folderPath,
					nil,
					nil,
					nil,
					false),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("folder_path_hash"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttrSet(resourceName, "folder_path_hash"),
				),
			},
			// update file in folder path
			{
				PreConfig: func() {
					if err := os.WriteFile(folderPath+"/newfile.txt", []byte("new contents..."), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: customModelWithoutLlmBlueprintResourceConfig(
					resourceTestName,
					"new_example_name",
					"new_example_description",
					baseEnvironmentID,
					nil,
					&folderPath,
					nil,
					nil,
					nil,
					false),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("folder_path_hash"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttrSet(resourceName, "folder_path_hash"),
				),
			},
			// Add resource settings
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					resourceTestName,
					"new_example_name",
					"new_example_description",
					baseEnvironmentID,
					nil,
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
					resource.TestCheckResourceAttr(resourceName, "memory_mb", "256"),
					resource.TestCheckResourceAttr(resourceName, "replicas", "2"),
					resource.TestCheckResourceAttr(resourceName, "network_access", "NONE"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestAccCustomModelWithTrainingDatasetResource(t *testing.T) {
	t.Parallel()

	resourceType := "datarobot_custom_model"
	resourceTestName := "test_with_training_dataset"
	resourceName := resourceType + "." + resourceTestName

	baseEnvironmentID := testGenAIBaseEnvID

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
			// Create with training dataset
			{
				Config: customModelWithoutLlmBlueprintResourceConfig(
					resourceTestName,
					"example_name",
					"example_description",
					baseEnvironmentID,
					sourceRemoteRepositories,
					nil,
					nil,
					nil,
					nil,
					true),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "training_dataset_id"),
					resource.TestCheckResourceAttrSet(resourceName, "training_dataset_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "training_dataset_name"),
				),
			},
		},
	})
}

func TestAccCustomModelWithRuntimeParamsResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_model.test_with_runtime_params"

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	baseEnvironmentID := testGenAIBaseEnvID

	folderPath := "custom_model_with_runtime_params"
	fileName := "model-metadata.yaml"
	fileContents := `name: runtime-params

runtimeParameterDefinitions:
  - fieldName: STRING_PARAMETER
    type: string
    description: An example of a string parameter
    defaultValue: null`

	os.RemoveAll(folderPath)
	err := os.Mkdir(folderPath, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(folderPath)

	if err = os.WriteFile(folderPath+"/"+fileName, []byte(fileContents), 0644); err != nil {
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
				Config: customModelWithRuntimeParamsConfig("val"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.0.value", "val"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// update runtime param value
			{
				Config: customModelWithRuntimeParamsConfig("newVal"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.0.value", "newVal"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// add new file
			{
				PreConfig: func() {
					if err := os.WriteFile(folderPath+"/newfile.txt", []byte("contents..."), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: customModelWithRuntimeParamsConfig("newVal"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.0.value", "newVal"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// remove runtime param
			{
				Config: customModelWithoutRuntimeParamsConfig(),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckNoResourceAttr(resourceName, "runtime_parameter_values.0.value"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
		},
	})
}

// TestAccBinaryCustomModelResource must be run with Resource Bundle feature enabled.
func TestAccBinaryCustomModelResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_model.test_binary"
	resourceBundle := "cpu.micro"
	resourceBundle2 := "cpu.small"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: binaryCustomModelResourceConfig("example_name", "target", "python", "1", "0", 0.5,
					&CustomModelResourceSettings{
						MemoryMB:      basetypes.NewInt64Value(256),
						Replicas:      basetypes.NewInt64Value(2),
						NetworkAccess: basetypes.NewStringValue("NONE"),
					},
					nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "target"),
					resource.TestCheckResourceAttr(resourceName, "positive_class_label", "1"),
					resource.TestCheckResourceAttr(resourceName, "negative_class_label", "0"),
					resource.TestCheckResourceAttr(resourceName, "language", "python"),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", "0.5"),
					resource.TestCheckResourceAttr(resourceName, "memory_mb", "256"),
					resource.TestCheckResourceAttr(resourceName, "replicas", "2"),
					resource.TestCheckResourceAttr(resourceName, "network_access", "NONE"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update parameters
			{
				Config: binaryCustomModelResourceConfig("new_example_name", "new_target", "r", "yes", "no", 0.8,
					&CustomModelResourceSettings{
						MemoryMB:      basetypes.NewInt64Value(512),
						Replicas:      basetypes.NewInt64Value(1),
						NetworkAccess: basetypes.NewStringValue("PUBLIC"),
					},
					nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "positive_class_label", "yes"),
					resource.TestCheckResourceAttr(resourceName, "negative_class_label", "no"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", "0.8"),
					resource.TestCheckResourceAttr(resourceName, "memory_mb", "512"),
					resource.TestCheckResourceAttr(resourceName, "replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "network_access", "PUBLIC"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Remove resource settings and add resource bundle
			{
				Config: binaryCustomModelResourceConfig("new_example_name", "new_target", "r", "yes", "no", 0.8, nil, &resourceBundle),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "positive_class_label", "yes"),
					resource.TestCheckResourceAttr(resourceName, "negative_class_label", "no"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", "0.8"),
					resource.TestCheckNoResourceAttr(resourceName, "memory_mb"),
					resource.TestCheckResourceAttr(resourceName, "replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "network_access", "PUBLIC"),
					resource.TestCheckResourceAttr(resourceName, "resource_bundle_id", resourceBundle),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update resource bundle
			{
				Config: binaryCustomModelResourceConfig("new_example_name", "new_target", "r", "yes", "no", 0.8, nil, &resourceBundle2),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "positive_class_label", "yes"),
					resource.TestCheckResourceAttr(resourceName, "negative_class_label", "no"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", "0.8"),
					resource.TestCheckNoResourceAttr(resourceName, "memory_mb"),
					resource.TestCheckResourceAttr(resourceName, "replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "network_access", "PUBLIC"),
					resource.TestCheckResourceAttr(resourceName, "resource_bundle_id", resourceBundle2),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

// TestAccMulticlassCustomModelResource must be run with Resource Bundle feature enabled.
func TestAccMulticlassCustomModelResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_model.test_multiclass"
	resourceBundle := "cpu.micro"
	resourceBundle2 := "cpu.small"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: multiclassCustomModelResourceConfig("example_name", "target", "python", []string{"class1", "class2", "class3"}, &resourceBundle),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "target"),
					resource.TestCheckResourceAttr(resourceName, "language", "python"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.0", "class1"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.1", "class2"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.2", "class3"),
					resource.TestCheckResourceAttr(resourceName, "resource_bundle_id", resourceBundle),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update parameters
			{
				Config: multiclassCustomModelResourceConfig("new_example_name", "new_target", "r", []string{"class1", "class8", "class3"}, &resourceBundle2),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.0", "class1"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.1", "class8"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.2", "class3"),
					resource.TestCheckResourceAttr(resourceName, "resource_bundle_id", resourceBundle2),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Remove resource bundle
			{
				Config: multiclassCustomModelResourceConfig("new_example_name", "new_target", "r", []string{"class1", "class8", "class3"}, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.0", "class1"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.1", "class8"),
					resource.TestCheckResourceAttr(resourceName, "class_labels.2", "class3"),
					resource.TestCheckResourceAttr(resourceName, "memory_mb", "512"),
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
	useCaseResourceName := "test_regression"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: regressionCustomModelResourceConfig("example_name", "target", "python", &useCaseResourceName),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "target"),
					resource.TestCheckResourceAttr(resourceName, "language", "python"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update parameters and remove use case
			{
				Config: regressionCustomModelResourceConfig("new_example_name", "new_target", "r", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckNoResourceAttr(resourceName, "use_case_ids.0"),
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

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	resourceName := "datarobot_custom_model.test_text_generation"
	useCaseResourceName := "test_text_generation"
	useCaseResourceName2 := "test_new_text_generation"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: textGenerationCustomModelResourceConfig("example_name", "target", "python", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "target"),
					resource.TestCheckResourceAttr(resourceName, "language", "python"),
					resource.TestCheckNoResourceAttr(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update parameters and add use case
			{
				Config: textGenerationCustomModelResourceConfig("new_example_name", "new_target", "r", &useCaseResourceName),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update use case
			{
				Config: textGenerationCustomModelResourceConfig("new_example_name", "new_target", "r", &useCaseResourceName2),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Remove use case
			{
				Config: textGenerationCustomModelResourceConfig("new_example_name", "new_target", "r", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckNoResourceAttr(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestAccMCPCustomModelResource(t *testing.T) {
	t.Parallel()

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	resourceName := "datarobot_custom_model.test_mcp_server"
	useCaseResourceName := "test_mcp_server"
	useCaseResourceName2 := "test_new_mcp_server"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: mcpServerCustomModelResourceConfig("example_name", "target", "python", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_type", "MCP"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "target"),
					resource.TestCheckResourceAttr(resourceName, "language", "python"),
					resource.TestCheckNoResourceAttr(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update parameters and add use case
			{
				Config: mcpServerCustomModelResourceConfig("new_example_name", "new_target", "r", &useCaseResourceName),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_type", "MCP"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update use case
			{
				Config: mcpServerCustomModelResourceConfig("new_example_name", "new_target", "r", &useCaseResourceName2),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_type", "MCP"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Remove use case
			{
				Config: mcpServerCustomModelResourceConfig("new_example_name", "new_target", "r", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_type", "MCP"),
					resource.TestCheckResourceAttr(resourceName, "target_name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "language", "r"),
					resource.TestCheckNoResourceAttr(resourceName, "use_case_ids.0"),
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
				Config: basicCustomModelResourceConfig("example_name", "Unstructured", "python"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_type", "Unstructured"),
					resource.TestCheckResourceAttr(resourceName, "language", "python"),
					resource.TestCheckResourceAttr(resourceName, "deployments_count", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update parameters
			{
				Config: basicCustomModelResourceConfig("new_example_name", "Unstructured", "r"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_type", "Unstructured"),
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

func TestAccAnomalyCustomModelResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_model.test_anomaly"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: basicCustomModelResourceConfig("example_name", "Anomaly", "python"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_type", "Anomaly"),
					resource.TestCheckResourceAttr(resourceName, "language", "python"),
					resource.TestCheckResourceAttr(resourceName, "deployments_count", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update parameters
			{
				Config: basicCustomModelResourceConfig("new_example_name", "Anomaly", "r"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "target_type", "Anomaly"),
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

func TestAccCustomModelWithTagsResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_model.test_with_tags"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with tags
			{
				Config: customModelResourceConfigWithTags("example_name_tags", "Unstructured", "python"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomModelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name_tags"),
					resource.TestCheckResourceAttr(resourceName, "target_type", "Unstructured"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "tags.*", map[string]string{
						"name":  "team",
						"value": "engineering",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "tags.*", map[string]string{
						"name":  "env",
						"value": "test",
					}),
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

func TestIsRuntimeParameterValuesUsed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	nonEmptyList, diags := listValueFromRuntimParameters(ctx, []RuntimeParameterValue{
		{Key: types.StringValue("FOO"), Type: types.StringValue("string"), Value: types.StringValue("bar")},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	emptyList, diags := listValueFromRuntimParameters(ctx, []RuntimeParameterValue{})
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	tests := []struct {
		name                   string
		runtimeParameterValues types.List
		expected               bool
	}{
		{
			name:                   "non-empty runtime_parameter_values returns true",
			runtimeParameterValues: nonEmptyList,
			expected:               true,
		},
		{
			name:                   "empty runtime_parameter_values returns false",
			runtimeParameterValues: emptyList,
			expected:               false,
		},
		{
			name:                   "null runtime_parameter_values returns false",
			runtimeParameterValues: types.ListNull(types.StringType),
			expected:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRuntimeParameterValuesUsed(tt.runtimeParameterValues)
			if got != tt.expected {
				t.Errorf("IsRuntimeParameterValuesUsed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCustomModelResourceConflictingRuntimeFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	r := NewCustomModelResource()

	configValidatable, ok := r.(fwresource.ResourceWithConfigValidators)
	if !ok {
		t.Fatal("CustomModelResource does not implement ResourceWithConfigValidators")
	}

	validators := configValidatable.ConfigValidators(ctx)

	if len(validators) < 6 {
		t.Fatalf("Expected at least 6 config validators (including runtime_parameter_values/runtime_parameters conflict), got %d", len(validators))
	}
}

func customModelFromLlmBlueprintResourceConfig(name, description, nameSalt string) string {
	return fmt.Sprintf(`
resource "datarobot_use_case" "test_custom_model" {
	name = "test custom model %s"
	description = "test"
}
resource "datarobot_dataset_from_file" "test_custom_model" {
	file_path = "../../test/datarobot_english_documentation_docsassist.zip"
	use_case_ids = ["${datarobot_use_case.test_custom_model.id}"]
}
resource "datarobot_vector_database" "test_custom_model" {
	  name = "test custom model %s"
	  dataset_id = "${datarobot_dataset_from_file.test_custom_model.id}"
	  use_case_id = "${datarobot_use_case.test_custom_model.id}"
}
resource "datarobot_playground" "test_custom_model" {
	name = "test custom model %s"
	description = "test"
	use_case_id = "${datarobot_use_case.test_custom_model.id}"
}
resource "datarobot_llm_blueprint" "test_custom_model" {
	name = "test custom model %s"
	description = "test"
	vector_database_id = "${datarobot_vector_database.test_custom_model.id}"
	playground_id = "${datarobot_playground.test_custom_model.id}"
	llm_id = "azure-openai-gpt-4-o"
}
resource "datarobot_api_token_credential" "test_custom_model" {
	name = "test custom model %s"
	description = "test"
	api_token = "test"
}
resource "datarobot_custom_model" "test_from_llm_blueprint" {
	name = "%s"
	description = "%s"
	source_llm_blueprint_id = "${datarobot_llm_blueprint.test_custom_model.id}"
	base_environment_id = "` + testGenAIBaseEnvID + `"
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
`, nameSalt, nameSalt, nameSalt, nameSalt, nameSalt, name, description)
}

func customModelWithoutLlmBlueprintResourceConfig(
	resourceName,
	name,
	description string,
	baseEnvironmentID string,
	remoteRepositories []SourceRemoteRepository,
	folderPath *string,
	files []FileTuple,
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
				id  = datarobot_remote_repository.%s.id
				ref = %s
				source_paths = %v
			},`, resourceName, remoteRepository.Ref, remoteRepository.SourcePaths)
		}
		remoteRepositoriesStr += "]"
	}

	folderStr := ""
	if folderPath != nil {
		folderStr = fmt.Sprintf(`
		folder_path = "%s"
		`, *folderPath)
	}

	filesStr := ""
	if len(files) > 0 {
		filesStr = "files = ["
		for _, file := range files {
			if file.PathInModel != "" {
				filesStr += fmt.Sprintf(`
				["%s", "%s"],`, file.LocalPath, file.PathInModel)
			} else {
				filesStr += fmt.Sprintf(`
				["%s"],`, file.LocalPath)
			}
		}

		filesStr += "]"
	}

	guardsStr := ""
	if len(guards) > 0 {
		guardsStr = "guard_configurations = ["
		for _, guard := range guards {
			guardCredentialStr := ""
			if guard.OpenAICredential != types.StringNull() {
				guardCredentialStr = fmt.Sprintf(`
				openai_credential = "${datarobot_api_token_credential.%s.id}"
				llm_type = %s
				`, resourceName, guard.LlmType)
				if IsKnown(guard.OpenAIApiBase) {
					guardCredentialStr += fmt.Sprintf(`
					openai_api_base = %s
					openai_deployment_id = %s
					`, guard.OpenAIApiBase, guard.OpenAIDeploymentID)
				}
			}

			nemoInfoStr := ""
			if guard.NemoInfo != nil {
				nemoInfoStr = fmt.Sprintf(`
				nemo_info = {
					blocked_terms = %s
				}`, guard.NemoInfo.BlockedTerms)
			}

			additionalGuardConfigStr := ""
			if guard.AdditionalGuardConfig != nil {
				additionalGuardConfigStr = fmt.Sprintf(`
				additional_guard_config = {
					cost = {
						currency = %s
						input_price = %s
						input_unit = %s
						output_price = %s
						output_unit = %s
					}
				}`,
					guard.AdditionalGuardConfig.Cost.Currency,
					guard.AdditionalGuardConfig.Cost.InputPrice,
					guard.AdditionalGuardConfig.Cost.InputUnit,
					guard.AdditionalGuardConfig.Cost.OutputPrice,
					guard.AdditionalGuardConfig.Cost.OutputUnit,
				)
			}
			guardsStr += fmt.Sprintf(`
			{
				template_name = %s
				name          = %s
				stages        = %v
				intervention = {
					action  = %s
					message = %s
					condition = jsonencode(%s)
				}
				%s
				%s
				%s
			},`,
				guard.TemplateName,
				guard.Name,
				guard.Stages,
				guard.Intervention.Action,
				guard.Intervention.Message,
				guard.Intervention.Condition.ValueString(),
				guardCredentialStr,
				nemoInfoStr,
				additionalGuardConfigStr)
		}
		guardsStr += "]"
	}

	resourceSettingsStr := ""
	if resourceSettings != nil {
		resourceSettingsStr = fmt.Sprintf(`
		memory_mb	    = %d
		replicas 	    = %d
		network_access  = %s
		`, resourceSettings.MemoryMB.ValueInt64(), resourceSettings.Replicas.ValueInt64(), resourceSettings.NetworkAccess)
	}

	trainingDatasetStr := ""
	if addTrainingData {
		trainingDatasetStr = fmt.Sprintf(`
		training_dataset_id = "${datarobot_dataset_from_file.%s.id}"
		`, resourceName)
	}

	return fmt.Sprintf(`
resource "datarobot_use_case" "%s" {
	name = "test custom model without llm blueprint %s"
}

resource "datarobot_dataset_from_file" "%s" {
	file_path = "../../test/datarobot_english_documentation_docsassist.zip"
	use_case_ids = ["${datarobot_use_case.%s.id}"]
}

resource "datarobot_remote_repository" "%s" {
	name        = "Test Custom Model from Remote Repository %s"
	description = "test"
	location    = "https://github.com/datarobot-community/custom-models"
	source_type = "github"
}

resource "datarobot_api_token_credential" "%s" {
	name = "open ai %s %s"
	api_token = "test"
}

resource "datarobot_custom_model" "%s" {
	name        		  = "%s"
	description 		  = "%s"
	target_type           = "TextGeneration"
	target_name           = "document"
	language 			  = "Python"
	base_environment_id   = "%s"
	%s
	%s
	%s
	%s
	%s
	%s
}
`, resourceName,
		nameSalt,
		resourceName,
		resourceName,
		resourceName,
		nameSalt,
		resourceName,
		resourceName,
		nameSalt,
		resourceName,
		name,
		description,
		baseEnvironmentID,
		remoteRepositoriesStr,
		folderStr,
		filesStr,
		guardsStr,
		resourceSettingsStr,
		trainingDatasetStr)
}

func customModelWithRuntimeParamsConfig(value string) string {
	return fmt.Sprintf(`
	resource "datarobot_custom_model" "test_with_runtime_params" {
		name        		     = "with runtime params"
		target_type              = "TextGeneration"
		target_name              = "target"
		base_environment_id      = "` + testGenAIBaseEnvID + `"
		folder_path 			 = "custom_model_with_runtime_params"
		runtime_parameter_values = [
			{
				key="STRING_PARAMETER",
				type="string",
				value="%s"
			},
		]
	}
	`, value)
}

func customModelWithoutRuntimeParamsConfig() string {
	return `
	resource "datarobot_custom_model" "test_with_runtime_params" {
		name        		     = "with runtime params"
		target_type              = "TextGeneration"
		target_name              = "target"
		base_environment_id      = "` + testGenAIBaseEnvID + `"
		folder_path 			 = "custom_model_with_runtime_params"
	}
	`
}

func binaryCustomModelResourceConfig(
	name,
	targetName,
	language,
	positiveClassLabel,
	negativeClassLabel string,
	predictionThreshold float64,
	resourceSettings *CustomModelResourceSettings,
	resourceBundleID *string) string {
	resourceBlock, customModelBlock := remoteRepositoryResource("test_custom_model_binary")

	resourceSettingsStr := ""
	if resourceSettings != nil {
		resourceSettingsStr = fmt.Sprintf(`
		memory_mb	    = %d
		replicas 	    = %d
		network_access  = %s
		`, resourceSettings.MemoryMB.ValueInt64(), resourceSettings.Replicas.ValueInt64(), resourceSettings.NetworkAccess)
	}

	resourceBundleStr := ""
	if resourceBundleID != nil {
		resourceBundleStr = fmt.Sprintf(`
	resource_bundle_id = "%s"
	`, *resourceBundleID)
	}

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
	base_environment_id   = "` + testGenAIBaseEnvID + `"
	%s
	%s
	%s
}
`, resourceBlock, name, targetName, language, positiveClassLabel, negativeClassLabel, predictionThreshold, customModelBlock, resourceSettingsStr, resourceBundleStr)
}

func multiclassCustomModelResourceConfig(
	name,
	targetName,
	language string,
	classLabels []string,
	resourceBundleID *string) string {
	resourceBlock, customModelBlock := remoteRepositoryResource("test_custom_model_multiclass")

	resourceBundleStr := ""
	if resourceBundleID != nil {
		resourceBundleStr = fmt.Sprintf(`
	resource_bundle_id = "%s"
	`, *resourceBundleID)
	} else {
		resourceBundleStr = "memory_mb = 512"
	}

	return fmt.Sprintf(`
%s

resource "datarobot_custom_model" "test_multiclass" {
	name        		  							  = "%s"
	target_type           							  = "Multiclass"
	target_name           							  = "%s"
	language 			  							  = "%s"
	class_labels  		  							  = [%s]
	base_environment_id 							  = "` + testGenAIBaseEnvID + `"
	is_proxy 										  = true
	%s
	%s
}
`, resourceBlock, name, targetName, language, "\""+strings.Join(classLabels, "\",\"")+"\"", resourceBundleStr, customModelBlock)
}

func regressionCustomModelResourceConfig(
	name,
	targetName,
	language string,
	useCaseResourceName *string) string {
	resourceBlock, customModelBlock := remoteRepositoryResource("test_custom_model_regression")

	useCaseIDsStr := ""
	if useCaseResourceName != nil {
		useCaseIDsStr = fmt.Sprintf(`use_case_ids = ["${datarobot_use_case.%s.id}"]`, *useCaseResourceName)
	}

	return fmt.Sprintf(`
%s
resource "datarobot_use_case" "test_regression" {
	name = "test custom model regression %s"
}

resource "datarobot_custom_model" "test_regression" {
	name        		  							  = "%s"
	target_type           							  = "Regression"
	target_name           							  = "%s"
	language 			  							  = "%s"
	base_environment_id 					  		  = "` + testGenAIBaseEnvID + `"
	%s
	%s
}
`, resourceBlock, nameSalt, name, targetName, language, useCaseIDsStr, customModelBlock)
}

func textGenerationCustomModelResourceConfig(
	name,
	targetName,
	language string,
	useCaseResourceName *string) string {
	resourceBlock, customModelBlock := remoteRepositoryResource("test_custom_model_text_generation")

	useCaseIDsStr := ""
	if useCaseResourceName != nil {
		useCaseIDsStr = fmt.Sprintf(`use_case_ids = ["${datarobot_use_case.%s.id}"]`, *useCaseResourceName)
	}

	return fmt.Sprintf(`
%s
resource "datarobot_use_case" "test_text_generation" {
	name = "test custom model text generation %s"
}
resource "datarobot_use_case" "test_new_text_generation" {
	name = "test new custom model text generation %s"
}

resource "datarobot_custom_model" "test_text_generation" {
	name        		= "%s"
	target_type         = "TextGeneration"
	target_name         = "%s"
	language 			= "%s"
	base_environment_id = "` + testGenAIBaseEnvID + `"
	is_proxy 			= true
	%s
	%s
}
`, resourceBlock, nameSalt, nameSalt, name, targetName, language, useCaseIDsStr, customModelBlock)
}

func mcpServerCustomModelResourceConfig(
	name,
	targetName,
	language string,
	useCaseResourceName *string) string {
	resourceBlock, customModelBlock := remoteRepositoryResource("test_custom_model_mcp_server")

	useCaseIDsStr := ""
	if useCaseResourceName != nil {
		useCaseIDsStr = fmt.Sprintf(`use_case_ids = ["${datarobot_use_case.%s.id}"]`, *useCaseResourceName)
	}

	return fmt.Sprintf(`
%s
resource "datarobot_use_case" "test_mcp_server" {
	name = "test custom model mcp server %s"
}
resource "datarobot_use_case" "test_new_mcp_server" {
	name = "test new custom model mcp server %s"
}

resource "datarobot_custom_model" "test_mcp_server" {
	name        		= "%s"
	target_type         = "MCP"
	target_name         = "%s"
	language 			= "%s"
	base_environment_id = "` + testGenAIBaseEnvID + `"
	is_proxy 			= true
	%s
	%s
}
`, resourceBlock, nameSalt, nameSalt, name, targetName, language, useCaseIDsStr, customModelBlock)
}

func basicCustomModelResourceConfig(
	name,
	targetType,
	language string) string {
	resourceBlock, customModelBlock := remoteRepositoryResource("test_custom_model_basic")

	return fmt.Sprintf(`
%s

resource "datarobot_custom_model" "test_%s" {
	name        		  							  = "%s"
	target_type           							  = "%s"
	language 			  							  = "%s"
	base_environment_id 							  = "` + testGenAIBaseEnvID + `"
	%s
}
`, resourceBlock, strings.ToLower(targetType), name, targetType, language, customModelBlock)
}

func remoteRepositoryResource(resourceName string) (string, string) {
	resourceBlock := fmt.Sprintf(`
resource "datarobot_remote_repository" "%s" {
	name        = "Test Custom Model from Remote Repository %s"
	description = "test"
	location    = "https://github.com/datarobot-community/custom-models"
	source_type = "github"
}
		`, resourceName, nameSalt)

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

		traceAPICall("GetGuardConfigurationsForCustomModelVersion")
		getGuardConfigsResp, err := p.service.GetGuardConfigurationsForCustomModelVersion(context.TODO(), customModel.LatestVersion.ID)
		if err != nil {
			return err
		}

		if customModel.Name == rs.Primary.Attributes["name"] &&
			customModel.Description == rs.Primary.Attributes["description"] {
			if rs.Primary.Attributes["runtime_parameter_values.0.value"] != "" {
				found := false
				for _, runtimeParam := range customModel.LatestVersion.RuntimeParameters {
					if runtimeParam.FieldName == rs.Primary.Attributes["runtime_parameter_values.0.key"] &&
						runtimeParam.CurrentValue == rs.Primary.Attributes["runtime_parameter_values.0.value"] {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("Runtime parameter value %s does not match", rs.Primary.Attributes["runtime_parameter_values.0.value"])
				}
			}
			if rs.Primary.Attributes["guard_configurations.0.name"] != "" {
				found := false
				for _, guardConfig := range getGuardConfigsResp.Data {
					// fmt.Printf("Guard Config: %s, Stage: %s, Expected Name: %s, Expected Stage: %s\n",
					// 	guardConfig.Name, guardConfig.Stages[0], rs.Primary.Attributes["guard_configurations.0.name"], rs.Primary.Attributes["guard_configurations.0.stages.0"])
					// Debugging output to check guard configuration values

					if guardConfig.Name == rs.Primary.Attributes["guard_configurations.0.name"] &&
						guardConfig.Stages[0] == rs.Primary.Attributes["guard_configurations.0.stages.0"] {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("Guard configuration %s does not match", rs.Primary.Attributes["guard_configurations.0.name"])
				}
			} else {
				if len(getGuardConfigsResp.Data) > 0 {
					return fmt.Errorf("Guard configuration found in state but not in config")
				}
			}
			return nil
		}

		return fmt.Errorf("Custom Model not found")
	}
}

func customModelResourceConfigWithTags(
	name,
	targetType,
	language string) string {
	resourceBlock, customModelBlock := remoteRepositoryResource("test_custom_model_tags")

	return fmt.Sprintf(`
%s

resource "datarobot_custom_model" "test_with_tags" {
	name        		  							  = "%s"
	target_type           							  = "%s"
	language 			  							  = "%s"
	base_environment_id 							  = "` + testGenAIBaseEnvID + `"
	%s

	tags = [
		{
			name  = "team"
			value = "engineering"
		},
		{
			name  = "env"
			value = "test"
		}
	]
}
`, resourceBlock, name, targetType, language, customModelBlock)
}

type hasRuntimeParamsMatcher struct{}

func (hasRuntimeParamsMatcher) Matches(x interface{}) bool {
	req, ok := x.(*client.CreateCustomModelVersionFromLatestRequest)
	return ok && req.RuntimeParameters != ""
}

func (hasRuntimeParamsMatcher) String() string {
	return "CreateCustomModelVersionFromLatestRequest with RuntimeParameters set"
}

func TestIntegrationCustomModelResourceRuntimeParameters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	modelID := uuid.NewString()
	versionID := uuid.NewString()
	name := uuid.NewString()
	baseEnvID := uuid.NewString()

	customModel := &client.CustomModel{
		ID:         modelID,
		Name:       name,
		TargetType: "TextGeneration",
		LatestVersion: client.CustomModelVersion{
			ID:                versionID,
			BaseEnvironmentID: baseEnvID,
		},
	}

	// Create: base environment version
	baseEnvCall := mockService.EXPECT().
		CreateCustomModelVersionCreateFromLatest(gomock.Any(), modelID, gomock.Any()).
		Return(&client.CustomModelVersion{BaseEnvironmentID: baseEnvID}, nil)

	// Create: model creation
	mockService.EXPECT().CreateCustomModel(gomock.Any(), gomock.Any()).Return(customModel, nil)

	// Create: files upload (empty)
	mockService.EXPECT().
		CreateCustomModelVersionFromFiles(gomock.Any(), modelID, gomock.Any()).
		Return(&client.CustomModelVersion{}, nil)

	// Create: wait after files
	mockService.EXPECT().IsCustomModelReady(gomock.Any(), modelID).Return(true, nil)
	mockService.EXPECT().GetCustomModel(gomock.Any(), modelID).Return(customModel, nil)

	// Create: runtime_parameters call (key assertion – must have RuntimeParameters set)
	runtimeParamsCall := mockService.EXPECT().
		CreateCustomModelVersionCreateFromLatest(gomock.Any(), modelID, hasRuntimeParamsMatcher{}).
		Return(&client.CustomModelVersion{}, nil).
		After(baseEnvCall)

	// Create: wait after runtime_parameters
	mockService.EXPECT().IsCustomModelReady(gomock.Any(), modelID).Return(true, nil)
	mockService.EXPECT().GetCustomModel(gomock.Any(), modelID).Return(customModel, nil)

	// Create: replicas/network version
	mockService.EXPECT().
		CreateCustomModelVersionCreateFromLatest(gomock.Any(), modelID, gomock.Any()).
		Return(&client.CustomModelVersion{}, nil).
		After(runtimeParamsCall)

	// Create: wait after replicas
	mockService.EXPECT().IsCustomModelReady(gomock.Any(), modelID).Return(true, nil)
	mockService.EXPECT().GetCustomModel(gomock.Any(), modelID).Return(customModel, nil)

	// Create: final GetCustomModel (line ~834)
	mockService.EXPECT().GetCustomModel(gomock.Any(), modelID).Return(customModel, nil)

	// Read (post-create refresh by the test framework)
	mockService.EXPECT().GetCustomModel(gomock.Any(), modelID).Return(customModel, nil)

	// ModifyPlan during idempotency plan check (state is non-null after Create)
	mockService.EXPECT().GetCustomModel(gomock.Any(), modelID).Return(customModel, nil)

	// Destroy: Read then Delete
	mockService.EXPECT().GetCustomModel(gomock.Any(), modelID).Return(customModel, nil)
	mockService.EXPECT().DeleteCustomModel(gomock.Any(), modelID).Return(nil)

	resourceName := "datarobot_custom_model.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: customModelWithRuntimeParametersConfig(name, baseEnvID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "runtime_parameters.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameters.0.key", "FOO"),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameters.0.type", "string"),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameters.0.value", "bar"),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.#", "0"),
				),
			},
		},
	})
}

func TestIntegrationCustomModelResourceRuntimeParametersOldAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	modelID := uuid.NewString()
	versionID := uuid.NewString()
	name := uuid.NewString()
	baseEnvID := uuid.NewString()

	customModel := &client.CustomModel{
		ID:         modelID,
		Name:       name,
		TargetType: "TextGeneration",
		LatestVersion: client.CustomModelVersion{
			ID:                versionID,
			BaseEnvironmentID: baseEnvID,
		},
	}

	// Create partially succeeds...
	mockService.EXPECT().CreateCustomModel(gomock.Any(), gomock.Any()).Return(customModel, nil)
	mockService.EXPECT().
		CreateCustomModelVersionCreateFromLatest(gomock.Any(), modelID, gomock.Any()).
		Return(&client.CustomModelVersion{BaseEnvironmentID: baseEnvID}, nil)
	mockService.EXPECT().
		CreateCustomModelVersionFromFiles(gomock.Any(), modelID, gomock.Any()).
		Return(&client.CustomModelVersion{}, nil)
	mockService.EXPECT().IsCustomModelReady(gomock.Any(), modelID).Return(true, nil)
	mockService.EXPECT().GetCustomModel(gomock.Any(), modelID).Return(customModel, nil)

	// ...then fails with the old-API error when runtime_parameters are applied.
	mockService.EXPECT().
		CreateCustomModelVersionCreateFromLatest(gomock.Any(), modelID, hasRuntimeParamsMatcher{}).
		Return(nil, fmt.Errorf("runtimeParameters is not allowed key"))

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      customModelWithRuntimeParametersConfig(name, baseEnvID),
				ExpectError: regexp.MustCompile("runtime_parameters not supported by this API"),
			},
		},
	})
}

func customModelWithRuntimeParametersConfig(name, baseEnvID string) string {
	return fmt.Sprintf(`
resource "datarobot_custom_model" "test" {
  name                = %q
  target_type         = "TextGeneration"
  base_environment_id = %q
  runtime_parameters  = [
    {
      key   = "FOO"
      type  = "string"
      value = "bar"
    }
  ]
}
`, name, baseEnvID)
}
