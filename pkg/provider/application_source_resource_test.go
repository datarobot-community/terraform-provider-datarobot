package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccApplicationSourceResource(t *testing.T) {
	t.Parallel()
	testApplicationSourceResource(t, false)
}

func TestIntegrationApplicationSourceResource(t *testing.T) {
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
	id := uuid.NewString()
	name := uuid.NewString()
	versionID := uuid.NewString()

	// Create
	mockService.EXPECT().ListExecutionEnvironments(gomock.Any()).Return(
		&client.ListExecutionEnvironmentsResponse{
			Data: []client.ExecutionEnvironment{
				client.ExecutionEnvironment{
					ID:   envID,
					Name: baseEnvironmentName,
				},
			},
		}, nil)

	mockService.EXPECT().CreateApplicationSource(gomock.Any()).Return(&client.ApplicationSource{
		ID:   id,
		Name: name,
	}, nil)
	mockService.EXPECT().CreateApplicationSourceVersion(gomock.Any(), id, &client.CreateApplicationSourceVersionRequest{
		Label:             "v1",
		BaseEnvironmentID: envID,
	}).Return(&client.ApplicationSourceVersion{
		ID: versionID,
	}, nil)
	mockService.EXPECT().UpdateApplicationSourceVersionFiles(gomock.Any(), id, versionID, gomock.Any()).Return(
		&client.ApplicationSourceVersion{
			ID: versionID,
		}, nil)

	// Test check
	mockService.EXPECT().GetApplicationSource(gomock.Any(), id).Return(&client.ApplicationSource{
		ID:   id,
		Name: name,
	}, nil)
	mockService.EXPECT().GetApplicationSourceVersion(gomock.Any(), id, versionID).Return(&client.ApplicationSourceVersion{
		ID:   versionID,
		Items: []client.FileItem{
			client.FileItem{
				FileName: "start-app.sh",
			},
		},
	}, nil)

	// Read
	mockService.EXPECT().GetApplicationSource(gomock.Any(), id).Return(&client.ApplicationSource{
		ID:   id,
		Name: name,
	}, nil)

	// Update
	mockService.EXPECT().GetApplicationSource(gomock.Any(), id).Return(&client.ApplicationSource{
		ID:   id,
		Name: name,
	}, nil)
	mockService.EXPECT().UpdateApplicationSource(gomock.Any(), id, &client.UpdateApplicationSourceRequest{
		Name: "new_example_name",
	}).Return(nil, nil)
	mockService.EXPECT().GetApplicationSourceVersion(gomock.Any(), id, versionID).Return(
		&client.ApplicationSourceVersion{
			ID: versionID,
			Items: []client.FileItem{
				client.FileItem{
					FileName: "streamlit-app.py",
				},
			},
		}, nil)
	mockService.EXPECT().UpdateApplicationSourceVersionFiles(gomock.Any(), id, versionID, gomock.Any()).Return(
		&client.ApplicationSourceVersion{
			ID: versionID,
		}, nil)

	// Test check
	mockService.EXPECT().GetApplicationSource(gomock.Any(), id).Return(&client.ApplicationSource{
		ID:   id,
		Name: "new_example_name",
	}, nil)
	mockService.EXPECT().GetApplicationSourceVersion(gomock.Any(), id, versionID).Return(&client.ApplicationSourceVersion{
		ID:   versionID,
		Items: []client.FileItem{
			client.FileItem{
				FileName: "streamlit-app.py",
			},
		},
	}, nil)

	// Delete
	mockService.EXPECT().GetApplicationSource(gomock.Any(), id).Return(&client.ApplicationSource{
		ID:   id,
		Name: "new_example_name",
	}, nil)
	mockService.EXPECT().DeleteApplicationSource(gomock.Any(), id).Return(nil)

	testApplicationSourceResource(t, true)
}

func testApplicationSourceResource(t *testing.T, isMock bool) {
	resourceName := "datarobot_application_source.test"

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
				Config: applicationSourceResourceConfig("", "start-app.sh"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceResourceExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttr(resourceName, "local_files.0", "start-app.sh"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update name and local file
			{
				Config: applicationSourceResourceConfig("new_example_name", "streamlit-app.py"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkApplicationSourceResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "local_files.0", "streamlit-app.py"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func applicationSourceResourceConfig(name string, localFile string) string {
	if name == "" {
		return fmt.Sprintf(`
resource "datarobot_application_source" "test" {
	local_files = ["%s"]
  }
`, localFile)	
	}

	return fmt.Sprintf(`
resource "datarobot_application_source" "test" {
	name = "%s"
	local_files = ["%s"]
  }
`, name, localFile)
}

func checkApplicationSourceResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetApplicationSourceInTest")
		applicationSource, err := p.service.GetApplicationSource(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}


		traceAPICall("GetApplicationSourceVersionInTest")
		applicationSourceVersion, err := p.service.GetApplicationSourceVersion(context.TODO(), rs.Primary.ID, rs.Primary.Attributes["version_id"] )
		if err != nil {
			return err
		}
		
		if applicationSource.Name == rs.Primary.Attributes["name"] && 
			applicationSourceVersion.Items[0].FileName == rs.Primary.Attributes["local_files.0"] {
			return nil
		}

		return fmt.Errorf("Application Source not found")
	}
}