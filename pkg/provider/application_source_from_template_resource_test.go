package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccApplicationSourceFromTemplateResource(t *testing.T) {
	t.Parallel()
	testApplicationSourceFromTemplateResource(t, false)
}

func TestApplicationSourceFromTemplateResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewApplicationSourceFromTemplateResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func testApplicationSourceFromTemplateResource(t *testing.T, isMock bool) {
	resourceName := "datarobot_application_source_from_template.test"

	newName := "new_from_template " + nameSalt

	baseEnvironmentID := "6542cd582a9d3d51bf4ac71e"
	baseEnvironmentVersionID := "668548c1b8e086572a96fbf5"

	appCodeFileName := "flask_app.py"
	appCode := `import streamlit as st
from datarobot import Client
from datarobot.client import set_client


def start_streamlit():
    set_client(Client())

    st.title("Example Custom Application Source from template.")

if __name__ == "__main__":
    start_streamlit()
	`

	err := os.WriteFile(appCodeFileName, []byte(appCode), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(appCodeFileName)

	folderPath := "application_source_from_template"
	if err = os.Mkdir(folderPath, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(folderPath)

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	resourceLabel := "cpu.medium"
	resourceLabel2 := "cpu.small"

	slackbotTemplateID := "67126757e7819551baceb22b"
	qaTemplateID := "670fb324bf9bbb1081114333"
	streamlitTemplateID := "671267597665e0b33f7acdb7"
	flaskTemplateID := "67126750e8342440587acd74"

	resource.Test(t, resource.TestCase{
		IsUnitTest: isMock,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("files_hashes"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Config: applicationSourceFromTemplateResourceConfig(
					"",
					&baseEnvironmentID,
					nil,
					[]FileTuple{
						{
							LocalPath: appCodeFileName,
						},
					},
					nil,
					&ApplicationSourceResources{
						Replicas:        types.Int64Value(1),
						ResourceLabel:   types.StringValue(resourceLabel),
						SessionAffinity: types.BoolValue(true),
					},
					slackbotTemplateID),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceFromTemplateResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", appCodeFileName),
					resource.TestCheckResourceAttrSet(resourceName, "files_hashes.0"),
					resource.TestCheckResourceAttr(resourceName, "resources.replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "resources.resource_label", resourceLabel),
					resource.TestCheckResourceAttr(resourceName, "resources.session_affinity", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
					resource.TestCheckResourceAttr(resourceName, "template_id", slackbotTemplateID),
				),
			},
			// Update name, files, resources, and environment
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("files_hashes"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Config: applicationSourceFromTemplateResourceConfig(
					newName,
					nil,
					&baseEnvironmentVersionID,
					[]FileTuple{
						{
							LocalPath: appCodeFileName,
						},
					},
					nil,
					&ApplicationSourceResources{
						Replicas:        types.Int64Value(2),
						ResourceLabel:   types.StringValue(resourceLabel2),
						SessionAffinity: types.BoolValue(false),
					},
					slackbotTemplateID),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceFromTemplateResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", appCodeFileName),
					resource.TestCheckResourceAttrSet(resourceName, "files_hashes.0"),
					resource.TestCheckResourceAttr(resourceName, "resources.replicas", "2"),
					resource.TestCheckResourceAttr(resourceName, "resources.resource_label", resourceLabel2),
					resource.TestCheckResourceAttr(resourceName, "resources.session_affinity", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
					resource.TestCheckResourceAttr(resourceName, "template_id", slackbotTemplateID),
				),
			},
			// Update file contents
			{
				PreConfig: func() {
					if err := os.WriteFile(appCodeFileName, []byte("app code..."), 0644); err != nil {
						t.Fatal(err)
					}
				},
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("files_hashes"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Config: applicationSourceFromTemplateResourceConfig(
					newName,
					&baseEnvironmentID,
					nil,
					[]FileTuple{
						{
							LocalPath: appCodeFileName,
						},
					},
					nil,
					nil,
					slackbotTemplateID),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceFromTemplateResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", appCodeFileName),
					resource.TestCheckResourceAttrSet(resourceName, "files_hashes.0"),
					resource.TestCheckResourceAttr(resourceName, "resources.replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "resources.resource_label", "cpu.small"),
					resource.TestCheckResourceAttr(resourceName, "resources.session_affinity", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
					resource.TestCheckResourceAttr(resourceName, "template_id", slackbotTemplateID),
				),
			},
			// Remove files and add folder_path
			{
				PreConfig: func() {
					if err := os.WriteFile(folderPath+"/new"+appCodeFileName, []byte(appCode), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: applicationSourceFromTemplateResourceConfig(
					newName,
					&baseEnvironmentID,
					nil,
					[]FileTuple{},
					&folderPath,
					nil,
					slackbotTemplateID),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("folder_path_hash"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceFromTemplateResourceExists(),
					resource.TestCheckNoResourceAttr(resourceName, "files.0.0"),
					resource.TestCheckNoResourceAttr(resourceName, "files_hashes.0"),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttrSet(resourceName, "folder_path_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
					resource.TestCheckResourceAttr(resourceName, "template_id", slackbotTemplateID),
				),
			},
			// Add new file to folder_path
			{
				PreConfig: func() {
					if err := os.WriteFile(folderPath+"/new2"+appCodeFileName, []byte(appCode), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: applicationSourceFromTemplateResourceConfig(
					newName,
					&baseEnvironmentID,
					nil,
					nil,
					&folderPath,
					nil,
					slackbotTemplateID),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("folder_path_hash"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceFromTemplateResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttrSet(resourceName, "folder_path_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
					resource.TestCheckResourceAttr(resourceName, "template_id", slackbotTemplateID),
				),
			},
			// update the contents of a file in folder_path
			{
				PreConfig: func() {
					if err := os.WriteFile(folderPath+"/new2"+appCodeFileName, []byte("new app code"), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: applicationSourceFromTemplateResourceConfig(
					newName,
					&baseEnvironmentID,
					nil,
					nil,
					&folderPath,
					nil,
					slackbotTemplateID),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("folder_path_hash"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceFromTemplateResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttrSet(resourceName, "folder_path_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
					resource.TestCheckResourceAttr(resourceName, "template_id", slackbotTemplateID),
				),
			},
			// update template_id triggers replace
			{
				Config: applicationSourceFromTemplateResourceConfig(
					newName,
					&baseEnvironmentID,
					nil,
					nil,
					&folderPath,
					nil,
					qaTemplateID),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("folder_path_hash"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceFromTemplateResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttrSet(resourceName, "folder_path_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
					resource.TestCheckResourceAttr(resourceName, "template_id", qaTemplateID),
				),
			},
			// update template_id triggers replace
			{
				Config: applicationSourceFromTemplateResourceConfig(
					newName,
					&baseEnvironmentID,
					nil,
					nil,
					&folderPath,
					nil,
					streamlitTemplateID),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("folder_path_hash"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceFromTemplateResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttrSet(resourceName, "folder_path_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
					resource.TestCheckResourceAttr(resourceName, "template_id", streamlitTemplateID),
				),
			},
			// update template_id triggers replace
			{
				Config: applicationSourceFromTemplateResourceConfig(
					newName,
					&baseEnvironmentID,
					nil,
					nil,
					&folderPath,
					nil,
					flaskTemplateID),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("folder_path_hash"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceFromTemplateResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttrSet(resourceName, "folder_path_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
					resource.TestCheckResourceAttr(resourceName, "template_id", flaskTemplateID),
				),
			},
			// Delete is tested automatically
		},
	})
}

func applicationSourceFromTemplateResourceConfig(
	name string,
	baseEnvironmentID *string,
	baseEnvironmentVersionID *string,
	files []FileTuple,
	folderPath *string,
	resources *ApplicationSourceResources,
	templateID string,
) string {
	baseEnvironmentIDStr := ""
	if baseEnvironmentID != nil {
		baseEnvironmentIDStr = fmt.Sprintf(`
	base_environment_id = "%s"
`, *baseEnvironmentID)
	}

	baseEnvironmentVersionIDStr := ""
	if baseEnvironmentVersionID != nil {
		baseEnvironmentVersionIDStr = fmt.Sprintf(`
	base_environment_version_id = "%s"
`, *baseEnvironmentVersionID)
	}

	resourcesStr := ""
	if resources != nil {
		resourcesStr = fmt.Sprintf(`
	resources = {
		replicas = %d
		resource_label = "%s"
		session_affinity = %t
	}
`, resources.Replicas.ValueInt64(),
			resources.ResourceLabel.ValueString(),
			resources.SessionAffinity.ValueBool())
	}

	nameStr := ""
	if name != "" {
		nameStr = fmt.Sprintf(`
	name = "%s"
`, name)
	}

	folderPathStr := ""
	if folderPath != nil {
		folderPathStr = fmt.Sprintf(`
	folder_path = "%s"
`, *folderPath)
	}

	filesStr := ""
	runtimeParamValueStr := ""
	if len(files) > 0 {
		runtimeParamValueStr = `
		runtime_parameter_values = [
			{ 
				key="SLACK_APP_TOKEN", 
				type="credential", 
				value="${datarobot_api_token_credential.test_app_source_from_template.id}",
			},
		  ]`

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

	return fmt.Sprintf(`
resource "datarobot_api_token_credential" "test_app_source_from_template" {
	name = "test app source from template %s"
	api_token = "test"
}
resource "datarobot_application_source_from_template" "test" {
	template_id = "%s"
	%s
	%s
	%s
	%s
	%s
	%s
	%s
  }
`, nameSalt,
		templateID,
		nameStr,
		baseEnvironmentIDStr,
		baseEnvironmentVersionIDStr,
		filesStr,
		folderPathStr,
		resourcesStr,
		runtimeParamValueStr)
}

func checkApplicationSourceFromTemplateResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_application_source_from_template.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_application_source_from_template.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = NewService(cl)

		traceAPICall("GetApplicationSource")
		ApplicationSourceFromTemplate, err := p.service.GetApplicationSource(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		traceAPICall("GetApplicationSourceVersion")
		ApplicationSourceFromTemplateVersion, err := p.service.GetApplicationSourceVersion(context.TODO(), rs.Primary.ID, rs.Primary.Attributes["version_id"])
		if err != nil {
			return err
		}

		if ApplicationSourceFromTemplate.Name == rs.Primary.Attributes["name"] &&
			ApplicationSourceFromTemplate.LatestVersion.BaseEnvironmentID == rs.Primary.Attributes["base_environment_id"] &&
			ApplicationSourceFromTemplate.LatestVersion.BaseEnvironmentVersionID == rs.Primary.Attributes["base_environment_version_id"] &&
			strconv.FormatInt(ApplicationSourceFromTemplateVersion.Resources.Replicas, 10) == rs.Primary.Attributes["resources.replicas"] {
			if runtimeParamValue, ok := rs.Primary.Attributes["runtime_parameter_values.0.value"]; ok {
				if runtimeParamValue != ApplicationSourceFromTemplateVersion.RuntimeParameters[0].OverrideValue {
					return fmt.Errorf("Runtime parameter value does not match")
				}

			}
			return nil
		}

		return fmt.Errorf("Application Source not found")
	}
}
