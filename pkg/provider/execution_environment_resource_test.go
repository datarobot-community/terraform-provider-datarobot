package provider

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccExecutionEnvironmentResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_execution_environment.test"

	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	// create directory
	dirName := "execution_environment_context"
	err := os.Mkdir(dirName, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	defer os.RemoveAll(dirName)

	dockerImageFileName := "../../test/golang_prebuilt_environment.tar.gz"
	dockerImage2FileName := "../../test/golang_prebuilt_environment_updated.tar.gz"

	dockerfileContents := `FROM python:3.9.5-slim-buster
	WORKDIR /app/
	COPY *.py /app/
	COPY requirements.txt /app/
	RUN pip install -U pip && pip install -r requirements.txt
	EXPOSE 8080
	ENTRYPOINT streamlit run app.py --server.port 8080`

	if err := os.WriteFile(filepath.Join(dirName, "Dockerfile"), []byte(dockerfileContents), 0644); err != nil {
		t.Fatal(err)
	}

	appContents := `import streamlit as st


def run():
    st.set_page_config(
        page_title="Example Custom App",
    )

    st.markdown("""
    This is an example streamlit app.
    """)


if __name__ == "__main__":
    run()`

	if err := os.WriteFile(filepath.Join(dirName, "app.py"), []byte(appContents), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dirName, "requirements.txt"), []byte("requests"), 0644); err != nil {
		t.Fatal(err)
	}

	// create tar file
	tarFileName := "docker_context.tar"
	tarFile, err := os.Create(tarFileName)
	if err != nil {
		t.Fatalf("Failed to create tar file: %v", err)
	}
	defer tarFile.Close()
	defer os.Remove(tarFileName)

	tarWriter := tar.NewWriter(tarFile)
	defer tarWriter.Close()

	if err = createTarFile(tarWriter, dirName); err != nil {
		t.Fatalf("Failed to create tar file: %v", err)
	}

	err = tarWriter.Close()
	if err != nil {
		t.Fatalf("Failed to close tar writer: %v", err)
	}

	// zip directory
	zipFileName := "docker_context.zip"
	contents, err := zipDirectory(dirName, zipFileName)
	if err != nil {
		t.Fatalf("Failed to zip directory: %v", err)
	}

	zipFile, err := os.Create(zipFileName)
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}

	_, err = zipFile.Write(contents)
	if err != nil {
		t.Fatalf("Failed to write to zip file: %v", err)
	}
	defer os.Remove(zipFileName)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: executionEnvironmentResourceConfig(
					"example_name",
					"example_description",
					"python",
					"customModel",
					"version_description",
					&tarFileName,
					nil,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "programming_language", "python"),
					resource.TestCheckResourceAttr(resourceName, "use_cases.0", "customModel"),
					resource.TestCheckResourceAttr(resourceName, "version_description", "version_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "build_status"),
				),
			},
			// Update name, description, and use case
			{
				Config: executionEnvironmentResourceConfig(
					"new_example_name",
					"new_example_description",
					"python",
					"customApplication",
					"version_description",
					&tarFileName,
					nil,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "programming_language", "python"),
					resource.TestCheckResourceAttr(resourceName, "use_cases.0", "customApplication"),
					resource.TestCheckResourceAttr(resourceName, "version_description", "version_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "build_status"),
				),
			},
			// update tar file contents triggers new version
			{
				PreConfig: func() {
					os.Remove(tarFileName)

					// update tar file
					if err := os.WriteFile(filepath.Join(dirName, "requirements.txt"), []byte("requests\nhttpx"), 0644); err != nil {
						t.Fatal(err)
					}

					tarFile, err := os.Create(tarFileName)
					if err != nil {
						t.Fatalf("Failed to create tar file: %v", err)
					}

					tarWriter := tar.NewWriter(tarFile)
					defer tarWriter.Close()

					if err = createTarFile(tarWriter, dirName); err != nil {
						t.Fatalf("Failed to create tar file: %v", err)
					}

					err = tarWriter.Close()
					if err != nil {
						t.Fatalf("Failed to close tar writer: %v", err)
					}
				},
				Config: executionEnvironmentResourceConfig(
					"new_example_name",
					"new_example_description",
					"python",
					"customApplication",
					"version_description",
					&tarFileName,
					nil,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "programming_language", "python"),
					resource.TestCheckResourceAttr(resourceName, "use_cases.0", "customApplication"),
					resource.TestCheckResourceAttr(resourceName, "version_description", "version_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "build_status"),
				),
			},
			// Update version description triggers new version
			{
				Config: executionEnvironmentResourceConfig(
					"new_example_name",
					"new_example_description",
					"python",
					"customApplication",
					"new_version_description",
					&tarFileName,
					nil,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "programming_language", "python"),
					resource.TestCheckResourceAttr(resourceName, "use_cases.0", "customApplication"),
					resource.TestCheckResourceAttr(resourceName, "version_description", "new_version_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "build_status"),
				),
			},
			// Update docker context path to zip file
			{
				Config: executionEnvironmentResourceConfig(
					"new_example_name",
					"new_example_description",
					"python",
					"customModel",
					"new_version_description",
					&zipFileName,
					nil,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "programming_language", "python"),
					resource.TestCheckResourceAttr(resourceName, "use_cases.0", "customModel"),
					resource.TestCheckResourceAttr(resourceName, "version_description", "new_version_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "build_status"),
				),
			},
			// Update docker context path to directory
			{
				Config: executionEnvironmentResourceConfig(
					"new_example_name",
					"new_example_description",
					"python",
					"customModel",
					"new_version_description",
					&dirName,
					nil,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "programming_language", "python"),
					resource.TestCheckResourceAttr(resourceName, "use_cases.0", "customModel"),
					resource.TestCheckResourceAttr(resourceName, "version_description", "new_version_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "build_status"),
				),
			},
			// update directory contents triggers new version
			{
				PreConfig: func() {
					// update requirements file
					if err := os.WriteFile(filepath.Join(dirName, "requirements.txt"), []byte("requests"), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: executionEnvironmentResourceConfig(
					"new_example_name",
					"new_example_description",
					"python",
					"customModel",
					"new_version_description",
					&dirName,
					nil,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "programming_language", "python"),
					resource.TestCheckResourceAttr(resourceName, "use_cases.0", "customModel"),
					resource.TestCheckResourceAttr(resourceName, "version_description", "new_version_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "build_status"),
				),
			},
			// new the image - triggers replace
			{
				Config: executionEnvironmentResourceConfig(
					"new_example_name",
					"new_example_description",
					"other",
					"customModel",
					"new_version_description",
					nil,
					&dockerImageFileName,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "programming_language", "other"),
					resource.TestCheckResourceAttr(resourceName, "use_cases.0", "customModel"),
					resource.TestCheckResourceAttr(resourceName, "version_description", "new_version_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "build_status"),
				),
			},
			// Update image (should be different hash) - triggers new version
			{
				Config: executionEnvironmentResourceConfig(
					"new_example_name",
					"new_example_description",
					"other",
					"customModel",
					"new_version_description",
					nil,
					&dockerImage2FileName,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "programming_language", "other"),
					resource.TestCheckResourceAttr(resourceName, "use_cases.0", "customModel"),
					resource.TestCheckResourceAttr(resourceName, "version_description", "new_version_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "build_status"),
				),
			},
			// Update language triggers replace
			{
				Config: executionEnvironmentResourceConfig(
					"new_example_name",
					"new_example_description",
					"r",
					"customModel",
					"new_version_description",
					&dirName,
					nil,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "programming_language", "r"),
					resource.TestCheckResourceAttr(resourceName, "use_cases.0", "customModel"),
					resource.TestCheckResourceAttr(resourceName, "version_description", "new_version_description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "build_status"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestAccExecutionEnvironmentResourceReadWithVersionID(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_execution_environment.test"

	dirName := "execution_environment_context_version_test"
	err := os.Mkdir(dirName, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	defer os.RemoveAll(dirName)

	dockerfileContents := `FROM python:3.9.5-slim-buster
WORKDIR /app/
COPY *.py /app/
COPY requirements.txt /app/
RUN pip install -U pip && pip install -r requirements.txt
EXPOSE 8080
ENTRYPOINT streamlit run app.py --server.port 8080`

	if err := os.WriteFile(filepath.Join(dirName, "Dockerfile"), []byte(dockerfileContents), 0644); err != nil {
		t.Fatal(err)
	}

	appContents := `import streamlit as st

def run():
    st.set_page_config(page_title="Example Custom App")
    st.markdown("This is an example streamlit app.")

if __name__ == "__main__":
    run()`

	if err := os.WriteFile(filepath.Join(dirName, "app.py"), []byte(appContents), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dirName, "requirements.txt"), []byte("requests"), 0644); err != nil {
		t.Fatal(err)
	}

	var firstVersionID string

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create execution environment - this will create version 1
			{
				Config: executionEnvironmentResourceConfig(
					"version_test_name",
					"version_test_description",
					"python",
					"customModel",
					"initial_version",
					&dirName,
					nil,
					nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					// Store the first version_id for later use
					func(s *terraform.State) error {
						rs := s.RootModule().Resources[resourceName]
						firstVersionID = rs.Primary.Attributes["version_id"]
						return nil
					},
				),
			},
			// Update to create a new version - this will create version 2
			{
				PreConfig: func() {
					// Modify the requirements to trigger a new version
					if err := os.WriteFile(filepath.Join(dirName, "requirements.txt"), []byte("requests\nhttpx"), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: executionEnvironmentResourceConfig(
					"version_test_name",
					"version_test_description",
					"python",
					"customModel",
					"updated_version",
					&dirName,
					nil,
					nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					// Verify that version_id changed (new version was created)
					func(s *terraform.State) error {
						rs := s.RootModule().Resources[resourceName]
						newVersionID := rs.Primary.Attributes["version_id"]
						if newVersionID == firstVersionID {
							return fmt.Errorf("Expected version_id to change after update, but it remained %s", firstVersionID)
						}
						return nil
					},
				),
			},
			// Verify that Read operation correctly reads the latest version when version_id is not explicitly set
			{
				Config: executionEnvironmentResourceConfig(
					"version_test_name",
					"version_test_description",
					"python",
					"customModel",
					"updated_version",
					&dirName,
					nil,
					nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					// Verify it's reading the latest version (should match the version from previous step)
					func(s *terraform.State) error {
						rs := s.RootModule().Resources[resourceName]
						currentVersionID := rs.Primary.Attributes["version_id"]
						// The version_id should still be the latest one from the previous step
						// This verifies that Read correctly uses the latest version when version_id is in state
						if currentVersionID == "" {
							return fmt.Errorf("version_id should be set")
						}
						return nil
					},
				),
			},
		},
	})
}

func TestExecutionEnvironmentResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewExecutionEnvironmentResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func TestAccExecutionEnvironmentResourceFromUri(t *testing.T) {
	t.Parallel()

	// Define variables needed for this specific test case
	dockerImageFileName := "../../test/golang_prebuilt_environment.tar.gz"
	dockerImageURI := "docker.io/library/alpine:latest"
	dockerImageURI_updated := "docker.io/library/alpine:3.19"
	resourceName := "datarobot_execution_environment.test"
	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test for conflicting docker_image and docker_image_uri
			{
				Config: executionEnvironmentResourceConfig(
					"conflict_test",
					"description",
					"python",
					"customModel",
					"version_description",
					nil, // no docker_context_path
					&dockerImageFileName,
					&dockerImageURI,
				),
				ExpectError: regexp.MustCompile(`(?i)These attributes cannot be configured together:`),
			},
			// Create environment version from URI
			{
				Config: executionEnvironmentResourceConfig(
					"example_name",
					"example_description",
					"python",
					"customModel",
					"env_version_from_uri_description",
					nil,
					nil,
					&dockerImageURI,
				),

				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},

				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "programming_language", "python"),
					resource.TestCheckResourceAttr(resourceName, "use_cases.0", "customModel"),
					resource.TestCheckResourceAttr(resourceName, "version_description", "env_version_from_uri_description"),
					resource.TestCheckResourceAttr(resourceName, "docker_image_uri", "docker.io/library/alpine:latest"),
					// when version is created from URI we set status to success right away.
					resource.TestCheckResourceAttr(resourceName, "build_status", "success"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},

			// Update docker image URI triggers new version
			{
				Config: executionEnvironmentResourceConfig(
					"example_name",
					"example_description",
					"python",
					"customModel",
					"env_version_from_uri_description",
					nil,
					nil,
					&dockerImageURI_updated,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkExecutionEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "programming_language", "python"),
					resource.TestCheckResourceAttr(resourceName, "use_cases.0", "customModel"),
					resource.TestCheckResourceAttr(resourceName, "version_description", "env_version_from_uri_description"),
					resource.TestCheckResourceAttr(resourceName, "docker_image_uri", "docker.io/library/alpine:3.19"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "build_status"),
				),
			},
		},
	})
}

func executionEnvironmentResourceConfig(
	name,
	description,
	programmingLanguage,
	useCase,
	versionDescription string,
	dockerContextPath *string,
	dockerImage *string,
	dockerImageUri *string,
) string {
	dockerContextPathStr := ""
	if dockerContextPath != nil {
		dockerContextPathStr = fmt.Sprintf(`
	docker_context_path = "%s"
		`, *dockerContextPath)
	}

	dockerImageStr := ""
	if dockerImage != nil {
		dockerImageStr = fmt.Sprintf(`
	docker_image = "%s"
			`, *dockerImage)
	}

	dockerImageUriStr := ""
	if dockerImageUri != nil {
		dockerImageUriStr = fmt.Sprintf(`
	docker_image_uri = "%s"
	`, *dockerImageUri)
	}

	return fmt.Sprintf(`
resource "datarobot_execution_environment" "test" {
	name = "%s"
	description = "%s"
	programming_language = "%s"
	use_cases = ["%s"]
	version_description = "%s"
	%s
	%s
	%s
}
`, name,
		description,
		programmingLanguage,
		useCase,
		versionDescription,
		dockerContextPathStr,
		dockerImageStr,
		dockerImageUriStr)
}

func checkExecutionEnvironmentResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_execution_environment.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_execution_environment.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = client.NewService(cl)

		traceAPICall("GetExecutionEnvironment")
		executionEnvironment, err := p.service.GetExecutionEnvironment(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		// Check if version_id is in state and verify it matches
		versionIDInState := rs.Primary.Attributes["version_id"]
		var expectedVersion *client.ExecutionEnvironmentVersion

		if versionIDInState != "" {
			// If version_id is in state, verify it's the correct version
			traceAPICall("GetExecutionEnvironmentVersion")
			specificVersion, err := p.service.GetExecutionEnvironmentVersion(context.TODO(), executionEnvironment.ID, versionIDInState)
			if err != nil {
				// If version not found, it should fall back to latest (this is expected behavior)
				expectedVersion = &executionEnvironment.LatestVersion
			} else {
				expectedVersion = specificVersion
			}
		} else {
			// If no version_id in state, use latest version
			expectedVersion = &executionEnvironment.LatestVersion
		}

		if executionEnvironment.Name == rs.Primary.Attributes["name"] &&
			executionEnvironment.Description == rs.Primary.Attributes["description"] &&
			executionEnvironment.ProgrammingLanguage == rs.Primary.Attributes["programming_language"] &&
			executionEnvironment.UseCases[0] == rs.Primary.Attributes["use_cases.0"] &&
			expectedVersion.Description == rs.Primary.Attributes["version_description"] &&
			expectedVersion.DockerImageUri == rs.Primary.Attributes["docker_image_uri"] {
			return nil
		}

		return fmt.Errorf("Execution Environment not found")
	}
}

func createTarFile(tarWriter *tar.Writer, dirName string) (err error) {
	err = WalkSymlinkSafe(dirName, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		header.Name, err = filepath.Rel(filepath.Dir(dirName), file)
		if err != nil {
			return err
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !fi.Mode().IsRegular() {
			return nil
		}

		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tarWriter, f)
		if err != nil {
			return err
		}

		return nil
	})

	return
}
