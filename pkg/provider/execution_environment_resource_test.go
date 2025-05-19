package provider

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	err := createOrCleanDirectory(dirName)
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

	if err := os.WriteFile(filepath.Join(dirName, "requirements.txt"), []byte("streamlit"), 0644); err != nil {
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
			// udpate tar file contents triggers new version
			{
				PreConfig: func() {
					os.Remove(tarFileName)

					// update tar file
					if err := os.WriteFile(filepath.Join(dirName, "requirements.txt"), []byte("streamlit\nflask"), 0644); err != nil {
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
					if err := os.WriteFile(filepath.Join(dirName, "requirements.txt"), []byte("streamlit"), 0644); err != nil {
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
					&dockerImageFileName),
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
					&dockerImage2FileName),
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

func executionEnvironmentResourceConfig(
	name,
	description,
	programmingLanguage,
	useCase,
	versionDescription string,
	dockerContextPath *string,
	dockerImage *string,
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

	return fmt.Sprintf(`
resource "datarobot_execution_environment" "test" {
	name = "%s"
	description = "%s"
	programming_language = "%s"
	use_cases = ["%s"]
	version_description = "%s"
	%s
	%s
}
`, name,
		description,
		programmingLanguage,
		useCase,
		versionDescription,
		dockerContextPathStr,
		dockerImageStr)
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

		if executionEnvironment.Name == rs.Primary.Attributes["name"] &&
			executionEnvironment.Description == rs.Primary.Attributes["description"] &&
			executionEnvironment.ProgrammingLanguage == rs.Primary.Attributes["programming_language"] &&
			executionEnvironment.UseCases[0] == rs.Primary.Attributes["use_cases.0"] &&
			executionEnvironment.LatestVersion.Description == rs.Primary.Attributes["version_description"] {
			return nil
		}

		return fmt.Errorf("Execution Environment not found")
	}
}

func createTarFile(tarWriter *tar.Writer, dirName string) (err error) {
	err = filepath.Walk(dirName, func(file string, fi os.FileInfo, err error) error {
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
