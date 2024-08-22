package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/omnistrate/terraform-provider-datarobot/internal/client"
)

func TestAccPredictionEnvironmentResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_prediction_environment.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: predictionEnvironmentResourceConfig("example_name", "example_description", "aws"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPredictionEnvironmentResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "platform", "aws"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, description, and platform
			{
				Config: predictionEnvironmentResourceConfig("new_example_name", "new_example_description", "gcp"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPredictionEnvironmentResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "platform", "gcp"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func predictionEnvironmentResourceConfig(name, description, platform string) string {
	return fmt.Sprintf(`
resource "datarobot_prediction_environment" "test" {
	  name = "%s"
	  description = "%s"
	  platform = "%s"
}
`, name, description, platform)
}

func checkPredictionEnvironmentResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetPredictionEnvironment")
		predictionEnvironment, err := p.service.GetPredictionEnvironment(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if predictionEnvironment.Name == rs.Primary.Attributes["name"] &&
			predictionEnvironment.Description == rs.Primary.Attributes["description"] &&
			predictionEnvironment.Platform == rs.Primary.Attributes["platform"] {
			return nil
		}

		return fmt.Errorf("Prediction Environment not found")
	}
}
