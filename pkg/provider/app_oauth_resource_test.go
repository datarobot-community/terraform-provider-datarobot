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

func TestAccAppOAuthResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_app_oauth.test"
	oauthName := uuid.NewString()

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())
	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())
	authProviderType := "box"
	clientID := "client-id"
	clientSecret := "client-secret"
	newClientSecret := "new-client-secret"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: AppOAuthResourceConfig(oauthName, authProviderType, clientID, clientSecret),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAppOAuthResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", oauthName),
					resource.TestCheckResourceAttr(resourceName, "type", authProviderType),
					resource.TestCheckResourceAttr(resourceName, "client_id", clientID),
					resource.TestCheckResourceAttr(resourceName, "client_secret", clientSecret),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name
			{
				Config: AppOAuthResourceConfig(oauthName+"_new", authProviderType, clientID, clientSecret),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAppOAuthResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", oauthName+"_new"),
				),
			},
			// Update client_secret triggers replace
			{
				Config: AppOAuthResourceConfig(oauthName+"_new", authProviderType, clientID, newClientSecret),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAppOAuthResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "client_secret", newClientSecret),
				),
			},
			// Delete is tested automatically
		},
	})
}

func AppOAuthResourceConfig(name, oauthType, clientID, clientSecret string) string {
	return fmt.Sprintf(`
resource "datarobot_app_oauth" "test" {
	name          = "%s"
	type          = "%s"
	client_id     = "%s"
	client_secret = "%s"
}
`, name, oauthType, clientID, clientSecret)
}

func checkAppOAuthResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_app_oauth.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_app_oauth.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = client.NewService(cl)

		oauth, err := p.service.GetAppOAuthProvider(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if oauth.Name == rs.Primary.Attributes["name"] &&
			oauth.Type == rs.Primary.Attributes["type"] {
			return nil
		}

		return fmt.Errorf("AppOAuth resource not found")
	}
}
