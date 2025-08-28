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
	resourceName := "datarobot_playground.test_acc_playground_resource"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read with default playground_type (should be "rag")
			{
				Config: playgroundResourceConfig("example_name", "example_description", "test_use_case", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPlaygroundResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "playground_type", "rag"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, description
			{
				Config: playgroundResourceConfig("new_example_name", "new_example_description", "test_use_case", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPlaygroundResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "playground_type", "rag"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update use case
			{
				Config: playgroundResourceConfig("new_example_name", "new_example_description", "new_test_use_case", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPlaygroundResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "playground_type", "rag"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccPlaygroundResourceWithExplicitType(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_playground.test_acc_playground_resource_with_explicit_type"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read with explicit playground_type "agentic"
			{
				Config: playgroundResourceConfig("example_name", "example_description", "test_use_case", "agentic"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPlaygroundResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "playground_type", "agentic"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update to explicit playground_type "rag"
			{
				Config: playgroundResourceConfig("example_name", "example_description", "test_use_case", "rag"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPlaygroundResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "playground_type", "rag"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func playgroundResourceConfig(name, description, use_case_resource_name, playground_type string) string {
	playgroundTypeConfig := ""
	if playground_type != "" {
		playgroundTypeConfig = fmt.Sprintf(`  playground_type = "%s"`, playground_type)
	}

	return fmt.Sprintf(`
resource "datarobot_use_case" "%s" {
	name = "%s"
	description = "%s"
}

resource "datarobot_playground" "test" {
	  name = "%s"
	  description = "%s"
	  use_case_id = "${datarobot_use_case.%s.id}"
%s
}
`, use_case_resource_name, name, description, name, description, use_case_resource_name, playgroundTypeConfig)
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

		// Check if playground_type is set in the resource attributes
		expectedPlaygroundType := "rag" // default value
		if playgroundType, ok := rs.Primary.Attributes["playground_type"]; ok && playgroundType != "" {
			expectedPlaygroundType = playgroundType
		}

		if playground.Name == rs.Primary.Attributes["name"] &&
			playground.Description == rs.Primary.Attributes["description"] &&
			playground.UseCaseID == rs.Primary.Attributes["use_case_id"] &&
			playground.PlaygroundType == expectedPlaygroundType {
			return nil
		}

		return fmt.Errorf("Playground not found or attributes don't match")
	}
}
