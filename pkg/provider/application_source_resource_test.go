package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccApplicationSourceResource(t *testing.T) {
	t.Parallel()
	testApplicationSourceResource(t, false)
}

func TestApplicationSourceResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewApplicationSourceResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func testApplicationSourceResource(t *testing.T, isMock bool) {
	resourceName := "datarobot_application_source.test"

	newName := "application_source " + nameSalt

	baseEnvironmentID := "6542cd582a9d3d51bf4ac71e"
	baseEnvironmentVersionID := "668548c1b8e086572a96fbf5"

	startAppFileName := "start-app.sh"
	startAppScript := `#!/usr/bin/env bash

echo "Starting App"

streamlit run streamlit-app.py
`

	appCodeFileName := "streamlit-app.py"
	appCode := `import streamlit as st
from datarobot import Client
from datarobot.client import set_client


def start_streamlit():
	set_client(Client())

	st.title("Example Custom Application")

if __name__ == "__main__":
    start_streamlit()
    `

    metadataFileName := "metadata.yaml"
    metadata := `name: runtime-params

runtimeParameterDefinitions:
  - fieldName: STRING_PARAMETER
    type: string
    description: An example of a string parameter`

	err := os.WriteFile(startAppFileName, []byte(startAppScript), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(startAppFileName)

	err = os.WriteFile(appCodeFileName, []byte(appCode), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(appCodeFileName)

	err = os.WriteFile(metadataFileName, []byte(metadata), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(metadataFileName)

	folderPath := "application_source"
	if err = os.Mkdir(folderPath, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(folderPath)

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	resourceLabel := "cpu.medium"

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
				Config: applicationSourceResourceConfig(
					"",
					&baseEnvironmentID,
					nil,
					[]FileTuple{
						{
							LocalPath: metadataFileName,
						},
						{
							LocalPath: startAppFileName,
						},
					},
					nil,
					nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", metadataFileName),
					resource.TestCheckResourceAttr(resourceName, "files.1.0", startAppFileName),
					resource.TestCheckResourceAttrSet(resourceName, "files_hashes.0"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.replicas"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.resource_label"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.session_affinity"),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.0.value", "val"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
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
				Config: applicationSourceResourceConfig(
					newName,
					nil,
					&baseEnvironmentVersionID,
					[]FileTuple{
						{
							LocalPath: metadataFileName,
						},
						{
							LocalPath: appCodeFileName,
						},
					},
					nil,
					&ApplicationSourceResources{
						Replicas:        types.Int64Value(2),
						ResourceLabel:   types.StringValue(resourceLabel),
						SessionAffinity: types.BoolValue(false),
					}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", metadataFileName),
					resource.TestCheckResourceAttr(resourceName, "files.1.0", appCodeFileName),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.0.value", "val"),
					resource.TestCheckResourceAttrSet(resourceName, "files_hashes.0"),
					resource.TestCheckResourceAttr(resourceName, "resources.replicas", "2"),
					resource.TestCheckResourceAttr(resourceName, "resources.resource_label", resourceLabel),
					resource.TestCheckResourceAttr(resourceName, "resources.session_affinity", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
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
				Config: applicationSourceResourceConfig(
					newName,
					&baseEnvironmentID,
					nil,
					[]FileTuple{
						{
							LocalPath: metadataFileName,
						},
						{
							LocalPath: appCodeFileName,
						},
					},
					nil,
					nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", metadataFileName),
					resource.TestCheckResourceAttr(resourceName, "files.1.0", appCodeFileName),
					resource.TestCheckResourceAttrSet(resourceName, "files_hashes.0"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.replicas"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.resource_label"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.session_affinity"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
				),
			},
			// Remove files and add folder_path
			{
				PreConfig: func() {
					if err := os.WriteFile(folderPath+"/"+startAppFileName, []byte(startAppScript), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: applicationSourceResourceConfig(
					newName,
					&baseEnvironmentID,
					nil,
					[]FileTuple{},
					&folderPath,
					nil),
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
					checkApplicationSourceResourceExists(),
					resource.TestCheckNoResourceAttr(resourceName, "files.0.0"),
					resource.TestCheckNoResourceAttr(resourceName, "files_hashes.0"),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttrSet(resourceName, "folder_path_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.replicas"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.resource_label"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.session_affinity"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
				),
			},
			// Add new file to folder_path
			{
				PreConfig: func() {
					if err := os.WriteFile(folderPath+"/"+appCodeFileName, []byte(appCode), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: applicationSourceResourceConfig(
					newName,
					&baseEnvironmentID,
					nil,
					nil,
					&folderPath,
					nil),
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
					checkApplicationSourceResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttrSet(resourceName, "folder_path_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.replicas"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.resource_label"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.session_affinity"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
				),
			},
			// update the contents of a file in folder_path
			{
				PreConfig: func() {
					if err := os.WriteFile(folderPath+"/"+appCodeFileName, []byte("new app code"), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: applicationSourceResourceConfig(
					newName,
					&baseEnvironmentID,
					nil,
					nil,
					&folderPath,
					nil),
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
					checkApplicationSourceResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttrSet(resourceName, "folder_path_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.replicas"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.resource_label"),
					resource.TestCheckNoResourceAttr(resourceName, "resources.session_affinity"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
				),
			},
			// Delete is tested automatically
		},
	})
}

func applicationSourceResourceConfig(
	name string,
	baseEnvironmentID *string,
	baseEnvironmentVersionID *string,
	files []FileTuple,
	folderPath *string,
	resources *ApplicationSourceResources,
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
				key="STRING_PARAMETER",
				type="string",
				value="val",
			},
		  ]`

		filesStr = "files = ["
		for _, file := range files {
			if file.PathInModel != "" {
				filesStr += fmt.Sprintf(`
				{ source = "%s", destination = "%s" },`, file.LocalPath, file.PathInModel)
			} else {
				filesStr += fmt.Sprintf(`
				{ source = "%s", destination = "%s" },`, file.LocalPath, file.LocalPath)
			}
		}
		filesStr += "]"
	}

	return fmt.Sprintf(`
resource "datarobot_application_source" "test" {
	%s
	%s
	%s
	%s
	%s
	%s
	%s
  }
`, nameStr, baseEnvironmentIDStr, baseEnvironmentVersionIDStr, filesStr, folderPathStr, resourcesStr, runtimeParamValueStr)
}

func checkApplicationSourceResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_application_source.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_application_source.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = NewService(cl)

		traceAPICall("GetApplicationSourceInTest")
		applicationSource, err := p.service.GetApplicationSource(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		traceAPICall("GetApplicationSourceVersionInTest")
		applicationSourceVersion, err := p.service.GetApplicationSourceVersion(context.TODO(), rs.Primary.ID, rs.Primary.Attributes["version_id"])
		if err != nil {
			return err
		}

		if applicationSource.Name == rs.Primary.Attributes["name"] &&
			applicationSource.LatestVersion.BaseEnvironmentID == rs.Primary.Attributes["base_environment_id"] &&
			applicationSource.LatestVersion.BaseEnvironmentVersionID == rs.Primary.Attributes["base_environment_version_id"] {
			if runtimeParamValue, ok := rs.Primary.Attributes["runtime_parameter_values.0.value"]; ok {
				if runtimeParamValue != applicationSourceVersion.RuntimeParameters[0].OverrideValue {
					return fmt.Errorf("Runtime parameter value does not match")
				}

			}
			return nil
		}

		return fmt.Errorf("Application Source not found")
	}
}
