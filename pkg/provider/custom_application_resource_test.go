package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccCustomApplicationResource(t *testing.T) {
	t.Parallel()
	testCustomApplicationResource(t, false)
}

func TestIntegrationCustomApplicationResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if os.Getenv(DataRobotApiKeyEnvVar) == "" {
		os.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	envID := uuid.NewString()
	sourceID := uuid.NewString()
	sourceVersionID := uuid.NewString()
	sourceVersionID2 := uuid.NewString()

	id := uuid.NewString()
	id2 := uuid.NewString()
	name := uuid.NewString()
	applicationUrl := uuid.NewString()

	// Create Application Source
	mockService.EXPECT().ListExecutionEnvironments(gomock.Any()).Return(
		&client.ListExecutionEnvironmentsResponse{
			Data: []client.ExecutionEnvironment{
				{
					ID:   envID,
					Name: baseEnvironmentName,
				},
			},
		}, nil)

	mockService.EXPECT().CreateApplicationSource(gomock.Any()).Return(&client.ApplicationSource{
		ID:   sourceID,
		Name: name,
	}, nil)
	mockService.EXPECT().CreateApplicationSourceVersion(gomock.Any(), sourceID, &client.CreateApplicationSourceVersionRequest{
		Label:             "v1",
		BaseEnvironmentID: envID,
	}).Return(&client.ApplicationSourceVersion{
		ID: sourceVersionID,
	}, nil)
	mockService.EXPECT().UpdateApplicationSourceVersionFiles(gomock.Any(), sourceID, sourceVersionID, gomock.Any()).Return(
		&client.ApplicationSourceVersion{
			ID: sourceVersionID,
		}, nil)

	// Create
	mockService.EXPECT().CreateApplicationFromSource(gomock.Any(), &client.CreateApplicationFromSourceRequest{
		ApplicationSourceVersionID: sourceVersionID,
	}).Return(&client.Application{
		ID:   id,
		Name: name,
	}, nil)
	mockService.EXPECT().IsApplicationReady(gomock.Any(), id).Return(true, nil)
	mockService.EXPECT().GetApplication(gomock.Any(), id).Return(&client.Application{
		ID:                               id,
		Name:                             name,
		ApplicationUrl:                   applicationUrl,
		CustomApplicationSourceID:        sourceID,
		CustomApplicationSourceVersionID: sourceVersionID,
	}, nil)

	// Test check
	mockService.EXPECT().GetApplication(gomock.Any(), id).Return(&client.Application{
		ID:                               id,
		Name:                             name,
		ApplicationUrl:                   applicationUrl,
		CustomApplicationSourceID:        sourceID,
		CustomApplicationSourceVersionID: sourceVersionID,
	}, nil)

	// Read
	mockService.EXPECT().GetApplicationSource(gomock.Any(), sourceID).Return(&client.ApplicationSource{
		ID:   sourceID,
		Name: name,
	}, nil)
	mockService.EXPECT().GetApplication(gomock.Any(), id).Return(&client.Application{
		ID:                               id,
		Name:                             name,
		ApplicationUrl:                   applicationUrl,
		CustomApplicationSourceID:        sourceID,
		CustomApplicationSourceVersionID: sourceVersionID,
	}, nil)

	// Update name
	mockService.EXPECT().GetApplicationSource(gomock.Any(), sourceID).Return(&client.ApplicationSource{
		ID:   sourceID,
		Name: name,
	}, nil)
	mockService.EXPECT().GetApplication(gomock.Any(), id).Return(&client.Application{
		ID:                               id,
		Name:                             name,
		ApplicationUrl:                   applicationUrl,
		CustomApplicationSourceID:        sourceID,
		CustomApplicationSourceVersionID: sourceVersionID,
	}, nil)
	mockService.EXPECT().UpdateApplication(gomock.Any(), id, &client.UpdateApplicationRequest{
		Name:                     "new_example_name",
		ExternalAccessEnabled:    true,
		ExternalAccessRecipients: []string{"test@test.com"},
	}).Return(&client.Application{
		ID:   id,
		Name: "new_example_name",
	}, nil)

	// Test check
	mockService.EXPECT().GetApplication(gomock.Any(), id).Return(&client.Application{
		ID:                               id,
		Name:                             "new_example_name",
		ApplicationUrl:                   applicationUrl,
		CustomApplicationSourceID:        sourceID,
		CustomApplicationSourceVersionID: sourceVersionID,
		ExternalAccessEnabled:            true,
	}, nil)

	// Update application source version
	mockService.EXPECT().GetApplicationSource(gomock.Any(), sourceID).Return(&client.ApplicationSource{
		ID:   sourceID,
		Name: name,
	}, nil)
	mockService.EXPECT().GetApplication(gomock.Any(), id).Return(&client.Application{
		ID:                               id,
		Name:                             "new_example_name",
		ApplicationUrl:                   applicationUrl,
		CustomApplicationSourceID:        sourceID,
		CustomApplicationSourceVersionID: sourceVersionID,
		ExternalAccessEnabled:            true,
	}, nil)
	mockService.EXPECT().DeleteApplication(gomock.Any(), id).Return(nil)
	mockService.EXPECT().DeleteApplicationSource(gomock.Any(), sourceID).Return(nil)
	mockService.EXPECT().ListExecutionEnvironments(gomock.Any()).Return(
		&client.ListExecutionEnvironmentsResponse{
			Data: []client.ExecutionEnvironment{
				{
					ID:   envID,
					Name: baseEnvironmentName,
				},
			},
		}, nil)
	mockService.EXPECT().CreateApplicationSource(gomock.Any()).Return(&client.ApplicationSource{
		ID:   sourceID,
		Name: name,
	}, nil)
	mockService.EXPECT().CreateApplicationSourceVersion(gomock.Any(), sourceID, &client.CreateApplicationSourceVersionRequest{
		Label:             "v1",
		BaseEnvironmentID: envID,
	}).Return(&client.ApplicationSourceVersion{
		ID: sourceVersionID2,
	}, nil)
	mockService.EXPECT().UpdateApplicationSourceVersionFiles(gomock.Any(), sourceID, sourceVersionID2, gomock.Any()).Return(
		&client.ApplicationSourceVersion{
			ID: sourceVersionID,
		}, nil)

	mockService.EXPECT().CreateApplicationFromSource(gomock.Any(), &client.CreateApplicationFromSourceRequest{
		ApplicationSourceVersionID: sourceVersionID2,
	}).Return(&client.Application{
		ID:   id2,
		Name: "new_example_name",
	}, nil)
	mockService.EXPECT().UpdateApplication(gomock.Any(), id2, &client.UpdateApplicationRequest{
		Name:                     "new_example_name",
		ExternalAccessEnabled:    true,
		ExternalAccessRecipients: []string{"test2@test.com"},
	}).Return(&client.Application{
		ID:   id2,
		Name: "new_example_name",
	}, nil)
	mockService.EXPECT().GetApplication(gomock.Any(), id2).Return(&client.Application{
		ID:                               id2,
		Name:                             "new_example_name",
		ApplicationUrl:                   applicationUrl,
		CustomApplicationSourceID:        sourceID,
		CustomApplicationSourceVersionID: sourceVersionID2,
		ExternalAccessEnabled:            true,
	}, nil)
	mockService.EXPECT().GetApplicationSource(gomock.Any(), sourceID).Return(&client.ApplicationSource{
		ID:   sourceID,
		Name: name,
	}, nil)
	mockService.EXPECT().IsApplicationReady(gomock.Any(), id2).Return(true, nil)
	mockService.EXPECT().GetApplication(gomock.Any(), id).Return(&client.Application{
		ID:                               id,
		Name:                             "new_example_name",
		ApplicationUrl:                   applicationUrl,
		CustomApplicationSourceID:        sourceID,
		CustomApplicationSourceVersionID: sourceVersionID,
		ExternalAccessEnabled:            true,
	}, nil)

	// Test check
	mockService.EXPECT().GetApplication(gomock.Any(), id2).Return(&client.Application{
		ID:                               id2,
		Name:                             "new_example_name",
		ApplicationUrl:                   applicationUrl,
		CustomApplicationSourceID:        sourceID,
		CustomApplicationSourceVersionID: sourceVersionID2,
		ExternalAccessEnabled:            true,
	}, nil)

	// Delete
	mockService.EXPECT().GetApplicationSource(gomock.Any(), sourceID).Return(&client.ApplicationSource{
		ID:   sourceID,
		Name: name,
	}, nil)
	mockService.EXPECT().GetApplication(gomock.Any(), id2).Return(&client.Application{
		ID:                               id2,
		Name:                             "new_example_name",
		ApplicationUrl:                   applicationUrl,
		CustomApplicationSourceID:        sourceID,
		CustomApplicationSourceVersionID: sourceVersionID2,
		ExternalAccessEnabled:            true,
	}, nil)
	mockService.EXPECT().DeleteApplication(gomock.Any(), id2).Return(nil)
	mockService.EXPECT().DeleteApplicationSource(gomock.Any(), sourceID).Return(nil)

	testCustomApplicationResource(t, true)
}

func testCustomApplicationResource(t *testing.T, isMock bool) {
	resourceName := "datarobot_custom_application.test"
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

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

	err := os.WriteFile("start-app.sh", []byte(startAppScript), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("start-app.sh")

	err = os.WriteFile("streamlit-app.py", []byte(appCode), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("streamlit-app.py")

	resource.Test(t, resource.TestCase{
		IsUnitTest: isMock,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: customApplicationResourceConfig("", "test", false, []string{}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomApplicationResourceExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
					resource.TestCheckResourceAttr(resourceName, "external_access_enabled", "false"),
					resource.TestCheckNoResourceAttr(resourceName, "external_access_recipients"),
				),
			},
			// Update name and external access
			{
				Config: customApplicationResourceConfig("new_example_name", "test", true, []string{"test@test.com"}),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("source_version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("application_url"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomApplicationResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
					resource.TestCheckResourceAttr(resourceName, "external_access_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "external_access_recipients.0", "test@test.com"),
				),
			},
			// Update Application source version
			{
				Config: customApplicationResourceConfig("new_example_name", "new_test", true, []string{"test2@test.com"}),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("source_version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("application_url"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomApplicationResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
					resource.TestCheckResourceAttr(resourceName, "external_access_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "external_access_recipients.0", "test2@test.com"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func customApplicationResourceConfig(
	name string,
	applicationSourceResourceName string,
	externalAccess bool,
	externalAccessRecipients []string,
) string {
	recipients := ""
	if len(externalAccessRecipients) > 0 {
		recipients = fmt.Sprintf(`
		external_access_recipients = %q
		`, externalAccessRecipients)
	}

	if name == "" {
		return fmt.Sprintf(`
resource "datarobot_application_source" "%s" {
	local_files = ["start-app.sh", "streamlit-app.py"]
}

resource "datarobot_custom_application" "test" {
	source_version_id = "${datarobot_application_source.%s.version_id}"
	external_access_enabled = %t
	%s
}`, applicationSourceResourceName, applicationSourceResourceName, externalAccess, recipients)
	}

	return fmt.Sprintf(`
resource "datarobot_application_source" "%s" {
	local_files = ["start-app.sh", "streamlit-app.py"]
}

resource "datarobot_custom_application" "test" {
	name = "%s"
	source_version_id = "${datarobot_application_source.%s.version_id}"
	external_access_enabled = %t
	%s
}
`, applicationSourceResourceName, name, applicationSourceResourceName, externalAccess, recipients)
}

func checkCustomApplicationResourceExists(resourceName string) resource.TestCheckFunc {
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
			b, err := strconv.ParseBool(rs.Primary.Attributes["external_access_enabled"])
			if err == nil {
				if application.ExternalAccessEnabled == b {
					if len(application.ExternalAccessRecipients) > 0 {
						if application.ExternalAccessRecipients[0] == rs.Primary.Attributes["external_access_recipients.0"] {
							return nil
						}
					} else if rs.Primary.Attributes["external_access_recipients.0"] == "" {
						return nil
					}
				}
			}
			return nil
		}

		return fmt.Errorf("Custom Application not found")
	}
}
