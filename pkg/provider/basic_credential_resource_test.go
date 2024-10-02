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

func TestAccBasicCredentialResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_basic_credential.test"
	credential_name := uuid.NewString()

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
				Config: basicCredentialResourceConfig(credential_name, "example_description", "example_user", "example_password"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkBasicCredentialResourceExists(),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", credential_name),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "user", "example_user"),
					resource.TestCheckResourceAttr(resourceName, "password", "example_password"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, description
			{
				Config: basicCredentialResourceConfig(credential_name+"_new", "new_example_description", "example_user", "example_password"),
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
					checkBasicCredentialResourceExists(),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", credential_name+"_new"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "user", "example_user"),
					resource.TestCheckResourceAttr(resourceName, "password", "example_password"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update user triggers replace
			{
				Config: basicCredentialResourceConfig(credential_name+"_new", "new_example_description", "new_example_user", "example_password"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkBasicCredentialResourceExists(),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", credential_name+"_new"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "user", "new_example_user"),
					resource.TestCheckResourceAttr(resourceName, "password", "example_password"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update password triggers replace
			{
				Config: basicCredentialResourceConfig(credential_name+"_new", "new_example_description", "new_example_user", "new_example_password"),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkBasicCredentialResourceExists(),
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

func checkBasicCredentialResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_basic_credential.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_basic_credential.test")
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
