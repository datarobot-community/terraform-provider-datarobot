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

func TestAccAwsCredentialResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_aws_credential.test"

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())
	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())

	credentialName := uuid.NewString()
	newCredentialName := uuid.NewString()

	description := "example_description"
	newDescription := "new_example_description"

	accessKeyID := "example_access_key_id"
	newAccessKeyID := "new_example_access_key_id"

	secretAccessKey := "example_secret_access_key"
	newSecretAccessKey := "new_example_secret_access_key"

	sessionToken := "example_session_token"
	newSessionToken := "new_example_session_token"

	configID := "example_config_id"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: awsCredentialResourceConfig(
					credentialName,
					description,
					accessKeyID,
					secretAccessKey,
					sessionToken,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAwsCredentialResourceExists(),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", credentialName),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "aws_access_key_id", accessKeyID),
					resource.TestCheckResourceAttr(resourceName, "aws_secret_access_key", secretAccessKey),
					resource.TestCheckResourceAttr(resourceName, "aws_session_token", sessionToken),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, description
			{
				Config: awsCredentialResourceConfig(
					newCredentialName,
					newDescription,
					accessKeyID,
					secretAccessKey,
					sessionToken,
					nil),
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
					checkAwsCredentialResourceExists(),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", newCredentialName),
					resource.TestCheckResourceAttr(resourceName, "description", newDescription),
					resource.TestCheckResourceAttr(resourceName, "aws_access_key_id", accessKeyID),
					resource.TestCheckResourceAttr(resourceName, "aws_secret_access_key", secretAccessKey),
					resource.TestCheckResourceAttr(resourceName, "aws_session_token", sessionToken),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update access key triggers replace
			{
				Config: awsCredentialResourceConfig(
					newCredentialName,
					newDescription,
					newAccessKeyID,
					newSecretAccessKey,
					newSessionToken,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAwsCredentialResourceExists(),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", newCredentialName),
					resource.TestCheckResourceAttr(resourceName, "description", newDescription),
					resource.TestCheckResourceAttr(resourceName, "aws_access_key_id", newAccessKeyID),
					resource.TestCheckResourceAttr(resourceName, "aws_secret_access_key", newSecretAccessKey),
					resource.TestCheckResourceAttr(resourceName, "aws_session_token", newSessionToken),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update config id triggers replace
			{
				Config: awsCredentialResourceConfig(
					newCredentialName,
					newDescription,
					newAccessKeyID,
					newSecretAccessKey,
					newSessionToken,
					&configID),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAwsCredentialResourceExists(),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", newCredentialName),
					resource.TestCheckResourceAttr(resourceName, "description", newDescription),
					resource.TestCheckResourceAttr(resourceName, "config_id", configID),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func awsCredentialResourceConfig(
	name,
	description,
	accessKeyId,
	secretAccessKey,
	sessionToken string,
	configID *string,
) string {
	if configID != nil {
		return fmt.Sprintf(`
resource "datarobot_aws_credential" "test" {
	name = "%s"
	description = "%s"
	config_id = "%s"
}
`, name, description, *configID)

	}

	return fmt.Sprintf(`
resource "datarobot_aws_credential" "test" {
	  name = "%s"
	  description = "%s"
	  aws_access_key_id = "%s"
	  aws_secret_access_key = "%s"
	  aws_session_token = "%s"
}
`, name, description, accessKeyId, secretAccessKey, sessionToken)
}

func checkAwsCredentialResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_aws_credential.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_aws_credential.test")
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

		return fmt.Errorf("AWS Credential not found")
	}
}
