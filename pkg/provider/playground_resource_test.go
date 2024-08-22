package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPlaygroundResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_playground.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: playgroundResourceConfig("example_name", "example_description", "test_use_case"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPlaygroundResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, description
			{
				Config: playgroundResourceConfig("new_example_name", "new_example_description", "test_use_case"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPlaygroundResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update use case
			{
				Config: playgroundResourceConfig("new_example_name", "new_example_description", "new_test_use_case"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPlaygroundResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func playgroundResourceConfig(name, description, use_case_resource_name string) string {
	return fmt.Sprintf(`
resource "datarobot_use_case" "%s" {
	name = "%s"
	description = "%s"
}

resource "datarobot_playground" "test" {
	  name = "%s"
	  description = "%s"
	  use_case_id = "${datarobot_use_case.%s.id}"
}
`, use_case_resource_name, name, description, name, description, use_case_resource_name)
}

func checkPlaygroundResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetPlayground")
		playground, err := p.service.GetPlayground(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if playground.Name == rs.Primary.Attributes["name"] &&
			playground.Description == rs.Primary.Attributes["description"] &&
			playground.UseCaseID == rs.Primary.Attributes["use_case_id"] {
			return nil
		}

		return fmt.Errorf("Playground not found")
	}
}
