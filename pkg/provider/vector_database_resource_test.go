package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccVectorDatabaseResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_vector_database.test"

	name := "vector_database " + nameSalt
	newName := "new_vector_database " + nameSalt

	chunkSize := 500
	newChunkSize := 510

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: vectorDatabaseResourceConfig(
					name,
					chunkSize),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkVectorDatabaseResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "chunking_parameters.chunk_size", "500"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name
			{
				Config: vectorDatabaseResourceConfig(
					newName,
					chunkSize),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkVectorDatabaseResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "chunking_parameters.chunk_size", "500"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update chunking parameters creates new version
			{
				Config: vectorDatabaseResourceConfig(
					newName,
					newChunkSize),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkVectorDatabaseResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "chunking_parameters.chunk_size", "510"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func vectorDatabaseResourceConfig(
	name string,
	chunkSize int,
) string {
	return fmt.Sprintf(`
resource "datarobot_use_case" "test_vector_database" {
	name = "test"
	description = "test"
}

resource "datarobot_dataset_from_file" "test_vector_database" {
	file_path = "../../test/datarobot_english_documentation_docsassist.zip"
	use_case_ids = ["${datarobot_use_case.test_vector_database.id}"]
}
resource "datarobot_vector_database" "test" {
	  name = "%s"
	  dataset_id = "${datarobot_dataset_from_file.test_vector_database.id}"
	  use_case_id = "${datarobot_use_case.test_vector_database.id}"
	  chunking_parameters = {
		chunk_size = %d
		separators = ["\n"]
	  }
}
`, name, chunkSize)
}

func checkVectorDatabaseResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetVectorDatabase")
		vectorDatabase, err := p.service.GetVectorDatabase(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if vectorDatabase.Name == rs.Primary.Attributes["name"] &&
			vectorDatabase.DatasetID == rs.Primary.Attributes["dataset_id"] &&
			vectorDatabase.UseCaseID == rs.Primary.Attributes["use_case_id"] {
			return nil
		}

		return fmt.Errorf("Vector Database not found")
	}
}
