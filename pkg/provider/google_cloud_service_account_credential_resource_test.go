package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const (
	gcpKeyJsonTemplate = `{
		"type": "service_account",
		"project_id": "example",
		"private_key_id": "1",
		"private_key": "-----BEGIN PRIVATE KEY-----%s\n-----END PRIVATE KEY-----\n",
		"client_email": "example",
		"client_id": "111",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/example",
		"universe_domain": "googleapis.com"
		}`
)

func TestAccGoogleCloudCredentialResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_google_cloud_credential.test"
	credentialName := uuid.NewString()
	gcpKeyFileName := "example.json"
	gcpKeyFileName2 := "example2.json"

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())
	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())

	if err := os.WriteFile(gcpKeyFileName, []byte(fmt.Sprintf(gcpKeyJsonTemplate, "file")), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(gcpKeyFileName)

	if err := os.WriteFile(gcpKeyFileName2, []byte(fmt.Sprintf(gcpKeyJsonTemplate, "file2")), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(gcpKeyFileName2)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: googleCloudCredentialResourceConfig(credentialName, false, &gcpKeyFileName),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkGoogleCloudCredentialResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", credentialName),
					resource.TestCheckResourceAttr(resourceName, "gcp_key_file", "example.json"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_key_file_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("gcp_key_file_hash"),
					),
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: googleCloudCredentialResourceConfig(credentialName+"_new", false, &gcpKeyFileName),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkGoogleCloudCredentialResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", credentialName+"_new"),
					resource.TestCheckResourceAttr(resourceName, "gcp_key_file", "example.json"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_key_file_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update gcp_key_file
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("gcp_key_file_hash"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: googleCloudCredentialResourceConfig(credentialName+"_new", false, &gcpKeyFileName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkGoogleCloudCredentialResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", credentialName+"_new"),
					resource.TestCheckResourceAttr(resourceName, "gcp_key_file", "example2.json"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_key_file_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update contents of gcp_key_file
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("gcp_key_file_hash"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				PreConfig: func() {
					if err := os.WriteFile(gcpKeyFileName2, []byte(fmt.Sprintf(gcpKeyJsonTemplate, "file2new")), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: googleCloudCredentialResourceConfig(credentialName+"_new", false, &gcpKeyFileName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkGoogleCloudCredentialResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", credentialName+"_new"),
					resource.TestCheckResourceAttr(resourceName, "gcp_key_file", "example2.json"),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_key_file_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Use gcp_key instead of gcp_key_file
			{
				Config: googleCloudCredentialResourceConfig(credentialName+"_new", true, nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkGoogleCloudCredentialResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "gcp_key"),
					resource.TestCheckNoResourceAttr(resourceName, "gcp_key_file"),
					resource.TestCheckNoResourceAttr(resourceName, "gcp_key_file_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func googleCloudCredentialResourceConfig(name string, gcpKey bool, gcpKeyFile *string) string {
	gcpKeyStr := ""
	if gcpKey {
		gcpKeyStr = fmt.Sprintf(`
		gcp_key = jsonencode(%s)
		`, fmt.Sprintf(gcpKeyJsonTemplate, "string"))
	}

	gcpKeyFileStr := ""
	if gcpKeyFile != nil {
		gcpKeyFileStr = fmt.Sprintf(`
		gcp_key_file = "%s"
		`, *gcpKeyFile)
	}

	return fmt.Sprintf(`
resource "datarobot_google_cloud_credential" "test" {
	  name = "%s"
	  %s
	  %s
}
`, name, gcpKeyStr, gcpKeyFileStr)
}

func checkGoogleCloudCredentialResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_google_cloud_credential.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_google_cloud_credential.test")
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
