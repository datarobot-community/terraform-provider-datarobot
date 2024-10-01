package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
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

	folderPath := "dir"
	if err = os.Mkdir(folderPath, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(folderPath)

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

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
				},
				Config: applicationSourceResourceConfig(
					"",
					[]FileTuple{
						{
							LocalPath: startAppFileName,
						},
					},
					nil,
					1),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", startAppFileName),
					resource.TestCheckResourceAttrSet(resourceName, "files_hashes.0"),
					resource.TestCheckResourceAttr(resourceName, "resource_settings.replicas", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update name, files, and replicas
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("files_hashes"),
					),
				},
				Config: applicationSourceResourceConfig(
					"new_example_name",
					[]FileTuple{
						{
							LocalPath: appCodeFileName,
						},
					},
					nil,
					2),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", appCodeFileName),
					resource.TestCheckResourceAttrSet(resourceName, "files_hashes.0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
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
				},
				Config: applicationSourceResourceConfig(
					"new_example_name",
					[]FileTuple{
						{
							LocalPath: appCodeFileName,
						},
					},
					nil,
					2),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", appCodeFileName),
					resource.TestCheckResourceAttrSet(resourceName, "files_hashes.0"),
					resource.TestCheckResourceAttr(resourceName, "resource_settings.replicas", "2"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Remove files and add folder_path
			{
				PreConfig: func() {
					if err := os.WriteFile("dir/"+startAppFileName, []byte(startAppScript), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: applicationSourceResourceConfig(
					"new_example_name",
					[]FileTuple{},
					&folderPath,
					2),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("folder_path_hash"),
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
				),
			},
			// Add new file to folder_path
			{
				PreConfig: func() {
					if err := os.WriteFile("dir/"+appCode, []byte(appCode), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: applicationSourceResourceConfig(
					"new_example_name",
					nil,
					&folderPath,
					2),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("folder_path_hash"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttrSet(resourceName, "folder_path_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// update the contents of a file in folder_path
			{
				PreConfig: func() {
					if err := os.WriteFile("dir/"+appCode, []byte("new app code"), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: applicationSourceResourceConfig(
					"new_example_name",
					nil,
					&folderPath,
					2),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("folder_path_hash"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttrSet(resourceName, "folder_path_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func applicationSourceResourceConfig(name string, files []FileTuple, folderPath *string, replicas int) string {
	resourceSettingsStr := ""
	if replicas > 1 {
		resourceSettingsStr = fmt.Sprintf(`
	resource_settings = {
		replicas = %d
	}
`, replicas)
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

	return fmt.Sprintf(`
resource "datarobot_application_source" "test" {
	%s
	%s
	%s
	%s
  }
`, nameStr, filesStr, folderPathStr, resourceSettingsStr)
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
			strconv.FormatInt(applicationSourceVersion.Resources.Replicas, 10) == rs.Primary.Attributes["resource_settings.replicas"] {
			return nil
		}

		return fmt.Errorf("Application Source not found")
	}
}
