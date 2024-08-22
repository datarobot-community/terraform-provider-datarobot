package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/omnistrate/terraform-provider-datarobot/internal/client"
)

func TestAccApiTokenCredentialResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_api_token_credential.test"
	credentialName := uuid.NewString()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: apiTokenCredentialResourceConfig(credentialName, "example_description", "example_api_token"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApiTokenCredentialResourceExists(resourceName),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", credentialName),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "api_token", "example_api_token"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, description, and api_token
			{
				Config: apiTokenCredentialResourceConfig(credentialName+"_new", "new_example_description", "new_example_api_token"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApiTokenCredentialResourceExists(resourceName),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", credentialName+"_new"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "api_token", "new_example_api_token"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func apiTokenCredentialResourceConfig(name, description, apiToken string) string {
	return fmt.Sprintf(`
resource "datarobot_api_token_credential" "test" {
	  name = "%s"
	  description = "%s"
	  api_token = "%s"
}
`, name, description, apiToken)
}

func checkApiTokenCredentialResourceExists(resourceName string) resource.TestCheckFunc {
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

		return fmt.Errorf("API Token Credential not found")
	}
}
