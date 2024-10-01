package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDatasetFromFileResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_dataset_from_file.test"

	datasetName := "example_dataset"
	newDatsetName := "new_example_dataset"

	useCase := "test_datasource"
	newUseCase := "test_new_datasource"

	fileName := "example.csv"
	fileName2 := "example2.csv"

	if err := os.WriteFile(fileName, []byte("col1,col2\nval1,val2"), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fileName)

	if err := os.WriteFile(fileName2, []byte("col11,col22\nval11,val22"), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fileName2)

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: datasetFromFileResourceConfig(fileName, &datasetName, &useCase),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasetFromFileResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "file_path", fileName),
					resource.TestCheckResourceAttr(resourceName, "name", datasetName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// update name and use case IDs
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: datasetFromFileResourceConfig(fileName, &newDatsetName, &newUseCase),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasetFromFileResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "file_path", fileName),
					resource.TestCheckResourceAttr(resourceName, "name", newDatsetName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// update file path triggers replace
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: datasetFromFileResourceConfig(fileName2, &newDatsetName, &newUseCase),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasetFromFileResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "file_path", fileName2),
					resource.TestCheckResourceAttr(resourceName, "name", newDatsetName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// update file contents triggers replace
			{
				PreConfig: func() {
					if err := os.WriteFile(fileName2, []byte("col11,col22\nnewVal1,newVal2"), 0644); err != nil {
						t.Fatal(err)
					}
				},
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: datasetFromFileResourceConfig(fileName2, &newDatsetName, &newUseCase),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasetFromFileResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "file_path", fileName2),
					resource.TestCheckResourceAttr(resourceName, "name", newDatsetName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestDatasetFromFileResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewDatasetFromFileResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func datasetFromFileResourceConfig(filePath string, name *string, useCaseID *string) string {
	nameStr := ""
	if name != nil {
		nameStr = fmt.Sprintf(`name = "%s"`, *name)
	}

	useCaseIDsStr := ""
	if useCaseID != nil {
		useCaseIDsStr = fmt.Sprintf(`use_case_ids = ["${datarobot_use_case.%s.id}"]`, *useCaseID)
	}

	return fmt.Sprintf(`
resource "datarobot_use_case" "test_datasource" {
	  name = "test"
}
resource "datarobot_use_case" "test_new_datasource" {
	  name = "test 2"
}

resource "datarobot_dataset_from_file" "test" {
	  file_path = "%s"
	  %s
	  %s
}
`, filePath, nameStr, useCaseIDsStr)
}

func checkDatasetFromFileResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_dataset_from_file.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_dataset_from_file.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = client.NewService(cl)

		traceAPICall("GetDataset")
		dataset, err := p.service.GetDataset(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if dataset.Name == rs.Primary.Attributes["name"] ||
			dataset.Name == strings.Split(rs.Primary.Attributes["file_path"], "/")[3] {
			return nil
		}

		return fmt.Errorf("Dataset not found")
	}
}
