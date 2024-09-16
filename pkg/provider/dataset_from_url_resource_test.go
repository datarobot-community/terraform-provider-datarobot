package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDatasetFromURLResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_dataset_from_url.test"

	// TODO: Update the URL to a valid URL
	url := "url"

	datasetName := "example_dataset"
	newDatsetName := "new_example_dataset"

	useCase := "test_datasource_from_url"
	newUseCase := "test_new_datasource_from_url"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: datasetFromURLResourceConfig(url, &datasetName, &useCase),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasetFromURLResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "url", url),
					resource.TestCheckResourceAttr(resourceName, "name", datasetName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// update name and use case IDs
			{
				Config: datasetFromURLResourceConfig(url, &newDatsetName, &newUseCase),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasetFromURLResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "url", url),
					resource.TestCheckResourceAttr(resourceName, "name", newDatsetName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestDatasetFromURLResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewDatasetFromURLResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func datasetFromURLResourceConfig(url string, name *string, useCaseID *string) string {
	nameStr := ""
	if name != nil {
		nameStr = fmt.Sprintf(`name = "%s"`, *name)
	}

	useCaseIDsStr := ""
	if useCaseID != nil {
		useCaseIDsStr = fmt.Sprintf(`use_case_ids = ["${datarobot_use_case.%s.id}"]`, *useCaseID)
	}

	return fmt.Sprintf(`
resource "datarobot_use_case" "test_datasource_from_url" {
	  name = "test"
}
resource "datarobot_use_case" "test_new_datasource_from_url" {
	  name = "test 2"
}

resource "datarobot_dataset_from_url" "test" {
	  url = "%s"
	  %s
	  %s
}
`, url, nameStr, useCaseIDsStr)
}

func checkDatasetFromURLResourceExists(resourceName string) resource.TestCheckFunc {
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

		if dataset.Name == rs.Primary.Attributes["name"] {
			return nil
		}

		return fmt.Errorf("Dataset not found")
	}
}
