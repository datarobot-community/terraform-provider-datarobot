package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRemoteRepositoryResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_remote_repository.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: remoteRepositoryResourceConfig("example_name", "example_description", "/example/location", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRemoteRepositoryResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "location", "/example/location"),
					resource.TestCheckResourceAttr(resourceName, "source_type", "github"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, description, and location
			{
				Config: remoteRepositoryResourceConfig("new_example_name", "new_example_description", "/example/new-location", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRemoteRepositoryResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "location", "/example/new-location"),
					resource.TestCheckResourceAttr(resourceName, "source_type", "github"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestAccRemoteRepositoryResourceWithCredential(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_remote_repository.test_credential"
	personal_access_token := "example_personal_access_token"
	updated_personal_access_token := "updated_example_personal_access_token"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: remoteRepositoryResourceConfig("example_name", "example_description", "/example/location", &personal_access_token),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRemoteRepositoryResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "location", "/example/location"),
					resource.TestCheckResourceAttr(resourceName, "source_type", "github"),
					resource.TestCheckResourceAttr(resourceName, "personal_access_token", personal_access_token),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, description, location, and personal_access_token
			{
				Config: remoteRepositoryResourceConfig("new_example_name", "new_example_description", "/example/new-location", &updated_personal_access_token),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRemoteRepositoryResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "location", "/example/new-location"),
					resource.TestCheckResourceAttr(resourceName, "source_type", "github"),
					resource.TestCheckResourceAttr(resourceName, "personal_access_token", updated_personal_access_token),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func remoteRepositoryResourceConfig(name, description, location string, personalAccessToken *string) string {
	if personalAccessToken != nil {
		return fmt.Sprintf(`
resource "datarobot_remote_repository" "test_credential" {
	name = "%s"
	description = "%s"
	location = "%s"
	source_type = "github"
	personal_access_token = "%s"
}
`, name, description, location, *personalAccessToken)
	}

	return fmt.Sprintf(`
resource "datarobot_remote_repository" "test" {
	  name = "%s"
	  description = "%s"
	  location = "%s"
	  source_type = "github"
}
`, name, description, location)
}

func checkRemoteRepositoryResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetRemoteRepository")
		remoteRepository, err := p.service.GetRemoteRepository(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if remoteRepository.Name == rs.Primary.Attributes["name"] &&
			remoteRepository.Description == rs.Primary.Attributes["description"] &&
			remoteRepository.Location == rs.Primary.Attributes["location"] &&
			remoteRepository.SourceType == rs.Primary.Attributes["source_type"] {

			// Check for personal_access_token
			if remoteRepository.CredentialID != "" && rs.Primary.Attributes["personal_access_token"] == "" {
				return fmt.Errorf("personal_access_token not found in attributes")
			}

			if remoteRepository.CredentialID == "" && rs.Primary.Attributes["personal_access_token"] != "" {
				return fmt.Errorf("personal_access_token found in attributes but not in remote repository")
			}

			return nil
		}

		return fmt.Errorf("Remote Repository not found")
	}
}
