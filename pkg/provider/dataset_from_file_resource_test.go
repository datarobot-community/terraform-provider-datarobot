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
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: datasetFromFileResourceConfig("../../test/datarobot_english_documentation_docsassist.zip"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasetFromFileResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "source_file", "../../test/datarobot_english_documentation_docsassist.zip"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_id"),
				),
			},
			// TODO: update source_file and use_case_id

			// Delete is tested automatically
		},
	})
}

func datasetFromFileResourceConfig(source_file string) string {
	return fmt.Sprintf(`
resource "datarobot_use_case" "test_datasource" {
	  name = "test"
	  description = "test"
}

resource "datarobot_dataset_from_file" "test" {
	  source_file = "%s"
	  use_case_id = "${datarobot_use_case.test_datasource.id}"
}
`, source_file)
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

		if dataset.Name == strings.Split(rs.Primary.Attributes["source_file"], "/")[3] {
			return nil
		}

		return fmt.Errorf("Dataset not found")
	}
}
