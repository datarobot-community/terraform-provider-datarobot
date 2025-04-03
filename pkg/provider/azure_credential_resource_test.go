package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccAzureCredentialResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_azure_credential.test"
	credentialName := uuid.NewString()

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())
	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: azureCredentialResourceConfig(
					credentialName,
					"example_description",
					"example_azure_connection_string"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAzureCredentialResourceExists(resourceName),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", credentialName),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "azure_connection_string", "example_azure_connection_string"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name and description
			{
				Config: azureCredentialResourceConfig(
					credentialName+"_new",
					"new_example_description",
					"example_azure_connection_string"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAzureCredentialResourceExists(resourceName),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", credentialName+"_new"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "azure_connection_string", "example_azure_connection_string"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update connection string trigger replace
			{
				Config: azureCredentialResourceConfig(
					credentialName+"_new",
					"new_example_description",
					"new_example_azure_connection_string"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAzureCredentialResourceExists(resourceName),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", credentialName+"_new"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "azure_connection_string", "new_example_azure_connection_string"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func azureCredentialResourceConfig(
	name,
	description,
	azureConnectionString string,
) string {
	return fmt.Sprintf(`
resource "datarobot_azure_credential" "test" {
	  name = "%s"
	  description = "%s"
	  azure_connection_string = "%s"
}
`, name, description, azureConnectionString)
}

func checkAzureCredentialResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetCredential")
		credential, err := p.service.GetCredential(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if credential.Name == rs.Primary.Attributes["name"] &&
			credential.Description == rs.Primary.Attributes["description"] {
			return nil
		}

		return fmt.Errorf("Azure Credential not found")
	}
}
