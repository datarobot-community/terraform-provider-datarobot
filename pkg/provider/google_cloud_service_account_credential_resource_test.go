package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/omnistrate/terraform-provider-datarobot/internal/client"
)

func TestAccGoogleCloudCredentialResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_google_cloud_credential.test"
	credentialName := uuid.NewString()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: googleCloudCredentialResourceConfig(credentialName, "example.json"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkGoogleCloudCredentialResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", credentialName),
					resource.TestCheckResourceAttr(resourceName, "source_file", "example.json"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name and source_file
			{
				Config: googleCloudCredentialResourceConfig(credentialName+"_new", "example2.json"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkGoogleCloudCredentialResourceExists(resourceName),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", credentialName+"_new"),
					resource.TestCheckResourceAttr(resourceName, "source_file", "example2.json"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})

	err := os.Remove("example.json")
	if err != nil {
		panic(err)
	}
	err = os.Remove("example2.json")
	if err != nil {
		panic(err)
	}
}

func googleCloudCredentialResourceConfig(name, source_file string) string {
	json := `{
		"type": "service_account",
		"project_id": "example",
		"private_key_id": "1",
		"private_key": "-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----\n",
		"client_email": "example",
		"client_id": "111",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/example",
		"universe_domain": "googleapis.com"
		}`

	err := os.WriteFile(source_file, []byte(json), 0644)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf(`
resource "datarobot_google_cloud_credential" "test" {
	  name = "%s"
	  source_file = "%s"
}
`, name, source_file)
}

func checkGoogleCloudCredentialResourceExists(resourceName string) resource.TestCheckFunc {
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

		if credential.Name == rs.Primary.Attributes["name"] {
			return nil
		}

		return fmt.Errorf("Google Cloud Credential not found")
	}
}
