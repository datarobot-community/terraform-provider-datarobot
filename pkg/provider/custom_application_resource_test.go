package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const (
	resourceName = "datarobot_custom_application.test"
)

func TestAccCustomApplicationResource(t *testing.T) {
	t.Parallel()

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())
	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())

	newName := "new_custom_application " + nameSalt

	useCaseResourceName := "test_custom_application"
	useCaseResourceName2 := "test_new_custom_application"

	folderPath := "custom_application"
	err := os.Mkdir(folderPath, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(folderPath)

	startAppScript := `#!/usr/bin/env bash

echo "Starting App"

streamlit run streamlit-app.py
`

	appCode := `import streamlit as st
from datarobot import Client
from datarobot.client import set_client


def start_streamlit():
	set_client(Client())

	st.title("Example Custom Application")

if __name__ == "__main__":
	start_streamlit()
	`

	err = os.WriteFile(folderPath+"/start-app.sh", []byte(startAppScript), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(folderPath+"/streamlit-app.py", []byte(appCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: customApplicationResourceConfig(
					"",
					1,
					false,
					[]string{},
					false,
					&useCaseResourceName),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomApplicationResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
					resource.TestCheckResourceAttr(resourceName, "external_access_enabled", "false"),
					resource.TestCheckNoResourceAttr(resourceName, "external_access_recipients"),
					resource.TestCheckResourceAttr(resourceName, "allow_auto_stopping", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// Update name, external access, and use case id
			{
				Config: customApplicationResourceConfig(
					newName,
					1,
					true,
					[]string{"test@test.com"},
					true,
					&useCaseResourceName2),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("source_version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("application_url"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomApplicationResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
					resource.TestCheckResourceAttr(resourceName, "external_access_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "external_access_recipients.0", "test@test.com"),
					resource.TestCheckResourceAttr(resourceName, "allow_auto_stopping", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// Update Application Source version and remove use case
			{
				Config: customApplicationResourceConfig(
					newName,
					2,
					true,
					[]string{"test2@test.com"},
					true,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("source_version_id"),
					),
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("application_url"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomApplicationResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
					resource.TestCheckResourceAttr(resourceName, "external_access_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "external_access_recipients.0", "test2@test.com"),
					resource.TestCheckResourceAttr(resourceName, "allow_auto_stopping", "true"),
					resource.TestCheckNoResourceAttr(resourceName, "use_case_ids.0"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func customApplicationResourceConfig(
	name string,
	applicationSourceReplicas int,
	externalAccess bool,
	externalAccessRecipients []string,
	allowAutoStopping bool,
	useCaseResourceName *string,
) string {
	recipients := ""
	if len(externalAccessRecipients) > 0 {
		recipients = fmt.Sprintf(`
		external_access_recipients = %q
		`, externalAccessRecipients)
	}

	nameStr := ""
	if name != "" {
		nameStr = fmt.Sprintf(`
		name = "%s"
		`, name)
	}

	useCaseIDsStr := ""
	if useCaseResourceName != nil {
		useCaseIDsStr = fmt.Sprintf(`use_case_ids = ["${datarobot_use_case.%s.id}"]`, *useCaseResourceName)
	}

	// resourcesBlock is not used, remove it
	return fmt.Sprintf(`
resource "datarobot_use_case" "test_custom_application" {
       name = "test custom application"
}
resource "datarobot_use_case" "test_new_custom_application" {
       name = "test new custom application"
}

resource "datarobot_application_source" "test" {
       base_environment_id = "6542cd582a9d3d51bf4ac71e"
       folder_path = "custom_application"
       files = [
               {
                       source = "custom_application/start-app.sh",
                       destination = "start-app.sh"
               },
               {
                       source = "custom_application/streamlit-app.py",
                       destination = "streamlit-app.py"
               }
       ]
       resources = {
               replicas = %d
               resource_label = "test-label"
               session_affinity = true
       }
}

resource "datarobot_custom_application" "test" {
       source_version_id = "${datarobot_application_source.test.version_id}"
       external_access_enabled = %t
       allow_auto_stopping = %t
       resources = {
           replicas = %d
           resource_label = "test-label"
           session_affinity = true
       }
       %s
       %s
       %s
}
`, applicationSourceReplicas, externalAccess, allowAutoStopping, applicationSourceReplicas, recipients, nameStr, useCaseIDsStr)
}

func checkCustomApplicationResourceExists() resource.TestCheckFunc {
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
		p.service = NewService(cl)

		traceAPICall("GetApplicationInTest")
		application, err := p.service.GetApplication(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if application.Name == rs.Primary.Attributes["name"] &&
			application.ApplicationUrl == rs.Primary.Attributes["application_url"] &&
			application.CustomApplicationSourceID == rs.Primary.Attributes["source_id"] &&
			application.CustomApplicationSourceVersionID == rs.Primary.Attributes["source_version_id"] {
			b, err := strconv.ParseBool(rs.Primary.Attributes["allow_auto_stopping"])
			if err == nil {
				if application.AllowAutoStopping != b {
					return fmt.Errorf("AllowAutoStopping is %t but should be %t", application.AllowAutoStopping, b)
				}
			}

			b, err = strconv.ParseBool(rs.Primary.Attributes["external_access_enabled"])
			if err == nil {
				if application.ExternalAccessEnabled == b {
					if len(application.ExternalAccessRecipients) > 0 {
						if application.ExternalAccessRecipients[0] != rs.Primary.Attributes["external_access_recipients.0"] {
							return fmt.Errorf("ExternalAccessRecipient is %s but should be %s", application.ExternalAccessRecipients[0], rs.Primary.Attributes["external_access_recipients.0"])
						}
					}
				}
			}

			// Check resources block
			if rs.Primary.Attributes["resources.replicas"] != "1" {
				return fmt.Errorf("resources.replicas is %s but should be 1", rs.Primary.Attributes["resources.replicas"])
			}
			if rs.Primary.Attributes["resources.resource_label"] != "test-label" {
				return fmt.Errorf("resources.resource_label is %s but should be test-label", rs.Primary.Attributes["resources.resource_label"])
			}
			if rs.Primary.Attributes["resources.session_affinity"] != "true" {
				return fmt.Errorf("resources.session_affinity is %s but should be true", rs.Primary.Attributes["resources.session_affinity"])
			}

			return nil
		}

		return fmt.Errorf("Custom Application not found")
	}
}
