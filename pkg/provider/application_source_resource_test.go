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

	testUniqueID := nameSalt + "-" + t.Name()
	name := "application_source " + testUniqueID
	newName := "new_application_source " + testUniqueID

	baseEnvironmentID := "6542cd582a9d3d51bf4ac71e"
	baseEnvironmentVersionID := "668548c1b8e086572a96fbf5"

	// Create a unique directory for this test to avoid parallel test interference
	testDir := fmt.Sprintf("test_app_source_%s", testUniqueID)
	os.RemoveAll(testDir)
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	// File basenames for references
	startAppFileBase := "start-app.sh"
	appCodeFileBase := "streamlit-app.py"
	metadataFileBase := "metadata.yaml"

	// Full paths for file operations
	startAppFileName := fmt.Sprintf("%s/%s", testDir, startAppFileBase)
	appCodeFileName := fmt.Sprintf("%s/%s", testDir, appCodeFileBase)
	metadataFileName := fmt.Sprintf("%s/%s", testDir, metadataFileBase)

	startAppScript := `#!/usr/bin/env bash

echo "Starting App"

streamlit run streamlit-app.py
`

	appCode := `import streamlit as st
from datarobot import Client
from datarobot.client import set_client


def start_streamlit():
    set_client(Client())

    st.title("Example Custom Application")

if __name__ == "__main__":
    start_streamlit()
	`

	metadata := `name: runtime-params

runtimeParameterDefinitions:
  - fieldName: STRING_PARAMETER
    type: string
    description: An example of a string parameter`

	err := os.WriteFile(startAppFileName, []byte(startAppScript), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(appCodeFileName, []byte(appCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(metadataFileName, []byte(metadata), 0644)
	if err != nil {
		t.Fatal(err)
	}

	folderPath := fmt.Sprintf("%s/application_source", testDir)
	if err = os.Mkdir(folderPath, 0755); err != nil {
		t.Fatal(err)
	}

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
					name,
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
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", metadataFileName),
					resource.TestCheckResourceAttr(resourceName, "files.1.0", startAppFileName),
					resource.TestCheckResourceAttrSet(resourceName, "files_hashes.0"),
					// Resources are now populated from API (computed field)
					resource.TestCheckResourceAttrSet(resourceName, "resources.replicas"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.resource_label"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.session_affinity"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.service_web_requests_on_root_path"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
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
					resource.TestCheckResourceAttr(resourceName, "resources.service_web_requests_on_root_path", "true"),
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
					// Resources are now populated from API (computed field)
					resource.TestCheckResourceAttrSet(resourceName, "resources.replicas"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.resource_label"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.session_affinity"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.service_web_requests_on_root_path"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Remove files and add folder_path
			{
				PreConfig: func() {
					if err := os.WriteFile(folderPath+"/"+startAppFileBase, []byte(startAppScript), 0644); err != nil {
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
					// Resources are now populated from API (computed field)
					resource.TestCheckResourceAttrSet(resourceName, "resources.replicas"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.resource_label"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.session_affinity"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.service_web_requests_on_root_path"),
				),
			},
			// Add new file to folder_path
			{
				PreConfig: func() {
					if err := os.WriteFile(folderPath+"/"+appCodeFileBase, []byte(appCode), 0644); err != nil {
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
					// Resources are now populated from API (computed field)
					resource.TestCheckResourceAttrSet(resourceName, "resources.replicas"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.resource_label"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.session_affinity"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.service_web_requests_on_root_path"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
				),
			},
			// update the contents of a file in folder_path
			{
				PreConfig: func() {
					if err := os.WriteFile(folderPath+"/"+appCodeFileBase, []byte("new app code"), 0644); err != nil {
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
					// Resources are now populated from API (computed field)
					resource.TestCheckResourceAttrSet(resourceName, "resources.replicas"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.resource_label"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.session_affinity"),
					resource.TestCheckResourceAttrSet(resourceName, "resources.service_web_requests_on_root_path"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(resourceName, "base_environment_version_id", baseEnvironmentVersionID),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestAccApplicationSourceResourceBatchFiles(t *testing.T) {
	t.Parallel()
	testApplicationSourceResourceBatchFiles(t, false)
}

func testApplicationSourceResourceBatchFiles(t *testing.T, isMock bool) {
	resourceName := "datarobot_application_source.test"

	baseEnvironmentID := "6542cd582a9d3d51bf4ac71e"

	testUniqueID := nameSalt + "-" + t.Name()
	testDir := fmt.Sprintf("test_batch_files_%s", testUniqueID)
	os.RemoveAll(testDir)
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)
	// Create 150 test files to test batching (more than the 100 file limit)
	numFiles := 150
	fileTuples := make([]FileTuple, numFiles+1) // +1 for metadata.yaml

	// First, create metadata.yaml to define runtime parameters
	metadataContent := `name: batch-test
runtimeParameterDefinitions:
  - fieldName: STRING_PARAMETER
    type: string
    description: A test string parameter
`
	metadataFile := fmt.Sprintf("%s/metadata.yaml", testDir)
	err := os.WriteFile(metadataFile, []byte(metadataContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	fileTuples[0] = FileTuple{
		LocalPath:   metadataFile,
		PathInModel: "metadata.yaml",
	}

	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("%s/file_%03d.txt", testDir, i)
		content := fmt.Sprintf("This is test file number %d\nGenerated for batch upload testing.", i)

		err := os.WriteFile(fileName, []byte(content), 0644)
		if err != nil {
			t.Fatal(err)
		}

		fileTuples[i+1] = FileTuple{
			LocalPath:   fileName,
			PathInModel: fmt.Sprintf("file_%03d.txt", i),
		}
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest: isMock,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read with batch files
			{
				Config: applicationSourceResourceConfig(
					"batch test application source "+testUniqueID,
					&baseEnvironmentID,
					nil,
					fileTuples,
					nil,
					nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "base_environment_id", baseEnvironmentID),
					// Check that we have the expected number of file hashes
					func(s *terraform.State) error {
						rs := s.RootModule().Resources[resourceName]
						if rs == nil {
							return fmt.Errorf("resource not found: %s", resourceName)
						}

						// Count file hashes (excluding metadata key)
						hashCount := 0
						for key := range rs.Primary.Attributes {
							if len(key) > 12 && key[:12] == "files_hashes" && key[12:13] == "." {
								// Skip the metadata key that contains the count
								if key == "files_hashes.#" {
									continue
								}
								hashCount++
							}
						}

						expectedFiles := numFiles + 1 // test files + metadata.yaml
						if hashCount != expectedFiles {
							return fmt.Errorf("expected %d file hashes, got %d", expectedFiles, hashCount)
						}

						return nil
					},
				),
			},
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
				["%s", "%s"],`, file.LocalPath, file.PathInModel)
			} else {
				filesStr += fmt.Sprintf(`
				["%s"],`, file.LocalPath)
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

func TestAccApplicationSourceRequiredKeyScopeLevel(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_application_source.test_scope"

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
    st.title("Scope Level Test Application")

if __name__ == "__main__":
    start_streamlit()
`

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

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with required_key_scope_level set to "admin"
			{
				Config: applicationSourceWithScopeLevelConfig("admin"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "required_key_scope_level", "admin"),
					checkApplicationSourceScopeLevel(resourceName, "admin"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func applicationSourceWithScopeLevelConfig(scopeLevel string) string {
	scopeLevelAttr := ""
	if scopeLevel != "" {
		scopeLevelAttr = fmt.Sprintf(`
	required_key_scope_level = "%s"`, scopeLevel)
	}

	return fmt.Sprintf(`
resource "datarobot_application_source" "test_scope" {
	base_environment_id = "6542cd582a9d3d51bf4ac71e"
	files = [
		["start-app.sh"],
		["streamlit-app.py"]
	]%s
}
`, scopeLevelAttr)
}

func checkApplicationSourceScopeLevel(resourceName, expectedLevel string) resource.TestCheckFunc {
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
		p.service = NewService(cl)

		traceAPICall("GetApplicationSourceInTest")
		applicationSource, err := p.service.GetApplicationSource(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		level := string(applicationSource.LatestVersion.RequiredKeyScopeLevel)
		if expectedLevel == "" {
			if level != "" { // empty string represents NoRequirements
				return fmt.Errorf("RequiredKeyScopeLevel should be empty but is %s", level)
			}
		} else {
			if level != expectedLevel {
				return fmt.Errorf("RequiredKeyScopeLevel is %s but should be %s", level, expectedLevel)
			}
		}

		return nil
	}
}

// Test-only struct for generating test configurations.
type ApplicationSourceResources struct {
	Replicas                     types.Int64
	SessionAffinity              types.Bool
	ResourceLabel                types.String
	ServiceWebRequestsOnRootPath types.Bool
}
