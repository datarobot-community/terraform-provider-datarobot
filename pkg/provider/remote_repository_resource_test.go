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
				Config: remoteRepositoryResourceConfig("example_name", "", "/example/location", nil, nil, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRemoteRepositoryResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckNoResourceAttr(resourceName, "description"),
					resource.TestCheckResourceAttr(resourceName, "location", "/example/location"),
					resource.TestCheckResourceAttr(resourceName, "source_type", "github"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, description, and location
			{
				Config: remoteRepositoryResourceConfig("new_example_name", "", "/example/new-location", nil, nil, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRemoteRepositoryResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckNoResourceAttr(resourceName, "description"),
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
				Config: remoteRepositoryResourceConfig("example_name", "example_description", "/example/location", &personal_access_token, nil, nil),
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
				Config: remoteRepositoryResourceConfig("new_example_name", "new_example_description", "/example/new-location", &updated_personal_access_token, nil, nil),
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

func TestAccS3RemoteRepositoryResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_remote_repository.test_s3"
	aws_access_key_id := "example_aws_access_key_id"
	aws_secret_access_key := "example_aws_secret_access_key"
	updated_aws_access_key_id := "updated_example_aws_access_key_id"
	updated_aws_secret_access_key := "updated_example_aws_secret_access_key"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: remoteRepositoryResourceConfig("example_name", "example_description", "example-location", nil, &aws_access_key_id, &aws_secret_access_key),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRemoteRepositoryResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "location", "example-location"),
					resource.TestCheckResourceAttr(resourceName, "source_type", "s3"),
					resource.TestCheckResourceAttr(resourceName, "aws_access_key_id", aws_access_key_id),
					resource.TestCheckResourceAttr(resourceName, "aws_secret_access_key", aws_secret_access_key),
					resource.TestCheckNoResourceAttr(resourceName, "aws_session_token"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, description, location, and aws credentials
			{
				Config: remoteRepositoryResourceConfig("new_example_name", "new_example_description", "example-new-location", nil, &updated_aws_access_key_id, &updated_aws_secret_access_key),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRemoteRepositoryResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "location", "example-new-location"),
					resource.TestCheckResourceAttr(resourceName, "source_type", "s3"),
					resource.TestCheckResourceAttr(resourceName, "aws_access_key_id", updated_aws_access_key_id),
					resource.TestCheckResourceAttr(resourceName, "aws_secret_access_key", updated_aws_secret_access_key),
					resource.TestCheckNoResourceAttr(resourceName, "aws_session_token"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func remoteRepositoryResourceConfig(name, description, location string, personalAccessToken, awsAccessKeyID, awsSecretAccessKey *string) string {
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

	if awsAccessKeyID != nil {
		return fmt.Sprintf(`
resource "datarobot_remote_repository" "test_s3" {
	name = "%s"
	description = "%s"
	location = "%s"
	source_type = "s3"
	aws_access_key_id = "%s"
	aws_secret_access_key = "%s"
}
`, name, description, location, *awsAccessKeyID, *awsSecretAccessKey)
	}

	return fmt.Sprintf(`
resource "datarobot_remote_repository" "test" {
	  name = "%s"
	  location = "%s"
	  source_type = "github"
}
`, name, location)
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

			// Check for credentials
			if remoteRepository.CredentialID != "" {
				if remoteRepository.SourceType == "s3" {
					if rs.Primary.Attributes["aws_access_key_id"] != "" && rs.Primary.Attributes["aws_secret_access_key"] != "" {
						return nil
					}
				} else if rs.Primary.Attributes["personal_access_token"] != "" {
					return nil
				}
			} else {
				return nil
			}
		}

		return fmt.Errorf("Remote Repository not found")
	}
}
