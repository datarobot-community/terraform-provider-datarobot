package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBasicCredentialResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_basic_credential.test"
	credential_name := uuid.NewString()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: basicCredentialResourceConfig(credential_name, "example_description", "example_user", "example_password"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkBasicCredentialResourceExists(resourceName),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", credential_name),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "user", "example_user"),
					resource.TestCheckResourceAttr(resourceName, "password", "example_password"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, description, user, and password
			{
				Config: basicCredentialResourceConfig(credential_name+"_new", "new_example_description", "new_example_user", "new_example_password"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkBasicCredentialResourceExists(resourceName),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", credential_name+"_new"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "user", "new_example_user"),
					resource.TestCheckResourceAttr(resourceName, "password", "new_example_password"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func basicCredentialResourceConfig(name, description, user, password string) string {
	return fmt.Sprintf(`
resource "datarobot_basic_credential" "test" {
	  name = "%s"
	  description = "%s"
	  user = "%s"
	  password = "%s"
}
`, name, description, user, password)
}

func checkBasicCredentialResourceExists(resourceName string) resource.TestCheckFunc {
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

		return fmt.Errorf("Basic Credential not found")
	}
}
