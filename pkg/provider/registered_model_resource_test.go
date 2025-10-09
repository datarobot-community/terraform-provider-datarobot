package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccRegisteredModelResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_registered_model.test"
	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	name := "registered model example name " + nameSalt
	newName := "new registered model example name" + nameSalt

	versionName := "version_name" + nameSalt
	newVersionName := "new_version_name" + nameSalt

	useCaseResourceName := "test_registered_model"
	useCaseResourceName2 := "test_new_registered_model"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: registeredModelResourceConfig(name, "example_description", nil, &useCaseResourceName, "1"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelResourceExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "version_name", name+" (v1)"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update name, description, and use case id
			{
				Config: registeredModelResourceConfig(newName, "new_example_description", &versionName, &useCaseResourceName2, "1"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelResourceExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "version_name", versionName),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update custom model version (by updating the Guard) creates new registered model version
			// and remove use case id
			{
				Config: registeredModelResourceConfig(newName, "new_example_description", &newVersionName, nil, "2"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelResourceExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "version_name", newVersionName),
					resource.TestCheckNoResourceAttr(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Test tags functionality
			{
				Config: registeredModelResourceConfigWithTags(newName, "new_example_description", "2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelResourceExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "tags.0.name", "team"),
					resource.TestCheckResourceAttr(resourceName, "tags.0.value", "engineering"),
					resource.TestCheckResourceAttr(resourceName, "tags.1.name", "env"),
					resource.TestCheckResourceAttr(resourceName, "tags.1.value", "test"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update tags
			{
				Config: registeredModelResourceConfigWithUpdatedTags(newName, "new_example_description", "2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelResourceExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "tags.0.name", "team"),
					resource.TestCheckResourceAttr(resourceName, "tags.0.value", "marketing"),
					resource.TestCheckResourceAttr(resourceName, "tags.1.name", "version"),
					resource.TestCheckResourceAttr(resourceName, "tags.1.value", "v2"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestAccTextGenerationRegisteredModelResource(t *testing.T) {
	t.Parallel()

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	nameSuffix := "test_registered_model_text_generation"
	nameSuffix2 := "test_registered_model_text_generation2"
	resourceName := "datarobot_registered_model."

	prompt := "prompt"

	folderPath := "registered_model_text_generation"
	fileName := "model-metadata.yaml"
	fileContents := `name: runtime-params

runtimeParameterDefinitions:
  - fieldName: PROMPT_COLUMN_NAME
    type: string
    defaultValue: null`

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
			// Create with prompt
			{
				Config: textGenerationRegisteredModelResourceConfig(nameSuffix, true, &prompt),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName+nameSuffix,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelResourceExists(resourceName+nameSuffix, &prompt),
					resource.TestCheckResourceAttrSet(resourceName+nameSuffix, "id"),
					resource.TestCheckResourceAttrSet(resourceName+nameSuffix, "version_id"),
				),
			},
			// Create without prompt
			{
				Config: textGenerationRegisteredModelResourceConfig(nameSuffix2, false, nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName+nameSuffix2,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelResourceExists(resourceName+nameSuffix2, nil),
					resource.TestCheckResourceAttrSet(resourceName+nameSuffix2, "id"),
					resource.TestCheckResourceAttrSet(resourceName+nameSuffix2, "version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestAccTextGenerationRegisteredModelUpdateResource(t *testing.T) {
	t.Parallel()

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	nameSuffix := "test_registered_model_update"
	resourceName := "datarobot_registered_model." + nameSuffix

	prompt := "test_prompt_column"
	newPrompt := "updated_prompt_column"

	folderPath := "registered_model_text_generation_update"
	fileName := "model-metadata.yaml"
	fileContents := `name: runtime-params

runtimeParameterDefinitions:
  - fieldName: PROMPT_COLUMN_NAME
    type: string
    defaultValue: null`

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
			// Create with initial prompt
			{
				Config: textGenerationRegisteredModelUpdateResourceConfig(nameSuffix, &prompt, "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelResourceExists(resourceName, &prompt),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update to new custom model version with different prompt - should preserve new prompt
			{
				Config: textGenerationRegisteredModelUpdateResourceConfig(nameSuffix, &newPrompt, "2"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelResourceExists(resourceName, &newPrompt),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestRegisteredModelResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewRegisteredModelResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func TestConvertTagsToClientTags(t *testing.T) {
	t.Parallel()

	// Test conversion from Terraform types to client types
	tfTags := []Tag{
		{
			Name:  types.StringValue("team"),
			Value: types.StringValue("engineering"),
		},
		{
			Name:  types.StringValue("env"),
			Value: types.StringValue("test"),
		},
	}

	clientTags := convertTagsToClientTags(tfTags)

	if len(clientTags) != 2 {
		t.Fatalf("Expected 2 client tags, got %d", len(clientTags))
	}

	if clientTags[0].Name != "team" || clientTags[0].Value != "engineering" {
		t.Errorf("Expected first tag to be {team: engineering}, got {%s: %s}", clientTags[0].Name, clientTags[0].Value)
	}

	if clientTags[1].Name != "env" || clientTags[1].Value != "test" {
		t.Errorf("Expected second tag to be {env: test}, got {%s: %s}", clientTags[1].Name, clientTags[1].Value)
	}
}

func TestConvertClientTagsToTfTags(t *testing.T) {
	t.Parallel()

	// Test conversion from client types to Terraform types
	clientTags := []client.Tag{
		{
			Name:  "team",
			Value: "engineering",
		},
		{
			Name:  "env",
			Value: "test",
		},
	}

	tfTags := convertClientTagsToTfTags(clientTags)

	if len(tfTags) != 2 {
		t.Fatalf("Expected 2 terraform tags, got %d", len(tfTags))
	}

	if tfTags[0].Name.ValueString() != "team" || tfTags[0].Value.ValueString() != "engineering" {
		t.Errorf("Expected first tag to be {team: engineering}, got {%s: %s}", tfTags[0].Name.ValueString(), tfTags[0].Value.ValueString())
	}

	if tfTags[1].Name.ValueString() != "env" || tfTags[1].Value.ValueString() != "test" {
		t.Errorf("Expected second tag to be {env: test}, got {%s: %s}", tfTags[1].Name.ValueString(), tfTags[1].Value.ValueString())
	}
}

func areTagsEqual(stateTags []Tag, planTags []Tag) bool {
	if len(stateTags) != len(planTags) {
		return false
	}

	// Create maps for easier comparison
	stateTagMap := make(map[string]string)
	for _, tag := range stateTags {
		stateTagMap[tag.Name.ValueString()] = tag.Value.ValueString()
	}

	for _, tag := range planTags {
		if stateTagMap[tag.Name.ValueString()] != tag.Value.ValueString() {
			return false
		}
	}

	return true
}

func TestAreTagsEqual(t *testing.T) {
	t.Parallel()

	// Test equal tags
	tags1 := []Tag{
		{
			Name:  types.StringValue("team"),
			Value: types.StringValue("engineering"),
		},
		{
			Name:  types.StringValue("env"),
			Value: types.StringValue("test"),
		},
	}

	tags2 := []Tag{
		{
			Name:  types.StringValue("team"),
			Value: types.StringValue("engineering"),
		},
		{
			Name:  types.StringValue("env"),
			Value: types.StringValue("test"),
		},
	}

	if !areTagsEqual(tags1, tags2) {
		t.Error("Expected equal tags to be equal")
	}

	// Test different tags
	tags3 := []Tag{
		{
			Name:  types.StringValue("team"),
			Value: types.StringValue("marketing"),
		},
		{
			Name:  types.StringValue("env"),
			Value: types.StringValue("prod"),
		},
	}

	if areTagsEqual(tags1, tags3) {
		t.Error("Expected different tags to not be equal")
	}

	// Test different lengths
	tags4 := []Tag{
		{
			Name:  types.StringValue("team"),
			Value: types.StringValue("engineering"),
		},
	}

	if areTagsEqual(tags1, tags4) {
		t.Error("Expected tags with different lengths to not be equal")
	}
}

func registeredModelResourceConfig(name, description string, versionName, useCaseResourceName *string, guardName string) string {
	versionNameStr := ""
	if versionName != nil {
		versionNameStr = `
		version_name = "` + *versionName + `"`
	}

	useCaseIDsStr := ""
	if useCaseResourceName != nil {
		useCaseIDsStr = fmt.Sprintf(`use_case_ids = ["${datarobot_use_case.%s.id}"]`, *useCaseResourceName)
	}

	return fmt.Sprintf(`
resource "datarobot_use_case" "test_registered_model" {
	name = "test registered model"
}
resource "datarobot_use_case" "test_new_registered_model" {
	name = "test new registered model"
}
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
	target_name = "my_label"
	base_environment_id = "65f9b27eab986d30d4c64268"
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
				condition = jsonencode({"comparand": 0.8, "comparator": "lessThan"})
			}
		},
	]
}
resource "datarobot_registered_model" "test" {
	name = "%s"
	description = "%s"
	custom_model_version_id = "${datarobot_custom_model.test_registered_model.version_id}"
	%s
	%s
}
`, guardName, name, description, versionNameStr, useCaseIDsStr)
}

func registeredModelResourceConfigWithTags(name, description string, guardName string) string {
	return fmt.Sprintf(`
resource "datarobot_use_case" "test_registered_model" {
	name = "test registered model"
}
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
	target_name = "my_label"
	base_environment_id = "65f9b27eab986d30d4c64268"
	source_remote_repositories = [
		{
			id = datarobot_remote_repository.test_registered_model.id
			ref = "master"
			source_paths = [
				"custom_inference/python/gan_mnist/custom.py",
				"custom_inference/python/gan_mnist/gan_weights.h5",
			]
		}
	]

	guard_configurations = [
		{
			template_name = "Rouge 1"
			name = "Rouge 1 %v"
			stages = [ "response" ]
			intervention = {
				action = "block"
				message = "you have been blocked by rouge 1 guard"
				condition = jsonencode({"comparand": 0.8, "comparator": "lessThan"})
			}
		},
	]
}
resource "datarobot_registered_model" "test" {
	name = "%s"
	description = "%s"
	custom_model_version_id = "${datarobot_custom_model.test_registered_model.version_id}"
	
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
`, guardName, name, description)
}

func registeredModelResourceConfigWithUpdatedTags(name, description string, guardName string) string {
	return fmt.Sprintf(`
resource "datarobot_use_case" "test_registered_model" {
	name = "test registered model"
}
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
	target_name = "my_label"
	base_environment_id = "65f9b27eab986d30d4c64268"
	source_remote_repositories = [
		{
			id = datarobot_remote_repository.test_registered_model.id
			ref = "master"
			source_paths = [
				"custom_inference/python/gan_mnist/custom.py",
				"custom_inference/python/gan_mnist/gan_weights.h5",
			]
		}
	]

	guard_configurations = [
		{
			template_name = "Rouge 1"
			name = "Rouge 1 %v"
			stages = [ "response" ]
			intervention = {
				action = "block"
				message = "you have been blocked by rouge 1 guard"
				condition = jsonencode({"comparand": 0.8, "comparator": "lessThan"})
			}
		},
	]
}
resource "datarobot_registered_model" "test" {
	name = "%s"
	description = "%s"
	custom_model_version_id = "${datarobot_custom_model.test_registered_model.version_id}"
	
	tags = [
		{
			name  = "team"
			value = "marketing"
		},
		{
			name  = "version"
			value = "v2"
		}
	]
}
`, guardName, name, description)
}

func textGenerationRegisteredModelResourceConfig(
	resourceName string,
	includePrompt bool,
	promptParameterValue *string,
) string {
	promptParamStr := ""
	if includePrompt {
		if promptParameterValue == nil {
			promptParamStr = `
		runtime_parameter_values = [
			{
				key="PROMPT_COLUMN_NAME",
				type="string",
				value=null
			},
		]`
		} else {
			promptParamStr = fmt.Sprintf(`
			runtime_parameter_values = [
				{
					key="PROMPT_COLUMN_NAME",
					type="string",
					value="%s"
				},
			]`, *promptParameterValue)
		}
	}

	return fmt.Sprintf(`
	resource "datarobot_custom_model" "%s" {
		name        			 = "test text generation registered model %s"
		target_type         	 = "TextGeneration"
		target_name         	 = "target"
		language 				 = "python"
		base_environment_id 	 = "65f9b27eab986d30d4c64268"
		is_proxy 				 = true
		folder_path 			 = "registered_model_text_generation"
		%s
	}

	resource "datarobot_registered_model" "%s" {
		name 					= "test text generation registered model %s"
		custom_model_version_id = "${datarobot_custom_model.%s.version_id}"
	}
	`, resourceName, nameSalt, promptParamStr, resourceName, nameSalt, resourceName)
}

func textGenerationRegisteredModelUpdateResourceConfig(
	resourceName string,
	promptParameterValue *string,
	version string,
) string {
	promptParamStr := ""
	if promptParameterValue != nil {
		promptParamStr = fmt.Sprintf(`
			runtime_parameter_values = [
				{
					key="PROMPT_COLUMN_NAME",
					type="string",
					value="%s"
				},
			]`, *promptParameterValue)
	}

	return fmt.Sprintf(`
	resource "datarobot_custom_model" "%s_v%s" {
		name        			 = "test text generation registered model update %s v%s"
		target_type         	 = "TextGeneration"
		target_name         	 = "target"
		language 				 = "python"
		base_environment_id 	 = "65f9b27eab986d30d4c64268"
		is_proxy 				 = true
		folder_path 			 = "registered_model_text_generation_update"
		%s
	}

	resource "datarobot_registered_model" "%s" {
		name 					= "test text generation registered model update %s"
		custom_model_version_id = "${datarobot_custom_model.%s_v%s.version_id}"
	}
	`, resourceName, version, nameSalt, version, promptParamStr, resourceName, nameSalt, resourceName, version)
}

func checkRegisteredModelResourceExists(resourceName string, prompt *string) resource.TestCheckFunc {
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

		if prompt != nil {
			if *latestRegisteredModelVersion.TextGeneration.Prompt != *prompt {
				return fmt.Errorf("Registered Model does not have prompt %s, instead: %s", *prompt, *latestRegisteredModelVersion.TextGeneration.Prompt)
			}
		}

		if registeredModel.Name == rs.Primary.Attributes["name"] &&
			registeredModel.Description == rs.Primary.Attributes["description"] &&
			latestRegisteredModelVersion.ID == rs.Primary.Attributes["version_id"] &&
			latestRegisteredModelVersion.Name == rs.Primary.Attributes["version_name"] {
			return nil
		}

		return fmt.Errorf("Registered Model not found")
	}
}
