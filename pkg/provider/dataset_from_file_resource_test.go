package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDatasetFromFileResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_dataset_from_file.test"

	datasetName := "example_dataset"
	newDatsetName := "new_example_dataset"

	useCase := "test_datasource"
	newUseCase := "test_new_datasource"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: datasetFromFileResourceConfig("../../test/datarobot_english_documentation_docsassist.zip", &datasetName, &useCase),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasetFromFileResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "file_path", "../../test/datarobot_english_documentation_docsassist.zip"),
					resource.TestCheckResourceAttr(resourceName, "name", datasetName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// update name and use case IDs
			{
				Config: datasetFromFileResourceConfig("../../test/datarobot_english_documentation_docsassist.zip", &newDatsetName, &newUseCase),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasetFromFileResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "file_path", "../../test/datarobot_english_documentation_docsassist.zip"),
					resource.TestCheckResourceAttr(resourceName, "name", newDatsetName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// Delete is tested automatically
		},
	})
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

func checkDatasetFromFileResourceExists(resourceName string) resource.TestCheckFunc {
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
