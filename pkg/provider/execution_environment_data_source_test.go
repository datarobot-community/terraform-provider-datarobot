package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccExecutionEnvironmentDataSource(t *testing.T) {
	t.Parallel()

	testExecutionEnvironmentDataSource(t, false)
}

func TestIntegrationExecutionEnvironmentDataSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	executionEnvironments, err := getExecutionEnvironments()
	if err != nil {
		t.Fatal(err)
	}

	for _, executionEnvironment := range executionEnvironments {
		id := uuid.NewString()
		versionID := uuid.NewString()
		description := uuid.NewString()
		programmingLanguage := "Python"

		for i := 0; i < 3; i++ {
			mockService.EXPECT().ListExecutionEnvironments(gomock.Any()).Return([]client.ExecutionEnvironment{
				{
					ID:                  id,
					Name:                executionEnvironment.Name,
					Description:         description,
					ProgrammingLanguage: programmingLanguage,
					LatestVersion: client.ExecutionEnvironmentVersion{
						ID: versionID,
					},
				},
			}, nil)
		}

		// Mock for lookup by ID (will be used in test steps)
		mockService.EXPECT().GetExecutionEnvironment(gomock.Any(), id).Return(&client.ExecutionEnvironment{
			ID:                  id,
			Name:                executionEnvironment.Name,
			Description:         description,
			ProgrammingLanguage: programmingLanguage,
			LatestVersion: client.ExecutionEnvironmentVersion{
				ID: versionID,
			},
		}, nil).AnyTimes()

		// Mock for lookup by ID with specific version
		specificVersionID := uuid.NewString()
		mockService.EXPECT().GetExecutionEnvironmentVersion(gomock.Any(), id, specificVersionID).Return(&client.ExecutionEnvironmentVersion{
			ID:          specificVersionID,
			Description: "specific version",
		}, nil).AnyTimes()
	}

	mockService.EXPECT().ListExecutionEnvironments(gomock.Any()).Return([]client.ExecutionEnvironment{}, nil)

	testExecutionEnvironmentDataSource(t, true)
}

func testExecutionEnvironmentDataSource(t *testing.T, isMock bool) {
	dataSourceName := "data.datarobot_execution_environment.test"

	steps := []resource.TestStep{}

	executionEnvironments, err := getExecutionEnvironments()
	if err != nil {
		t.Fatal(err)
	}

	for _, executionEnvironment := range executionEnvironments {
		steps = append(steps, resource.TestStep{
			Config: executionEnvironmentDataSourceConfigByName(executionEnvironment.Name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(dataSourceName, "name", executionEnvironment.Name),
				resource.TestCheckResourceAttrSet(dataSourceName, "id"),
				resource.TestCheckResourceAttrSet(dataSourceName, "programming_language"),
				resource.TestCheckResourceAttrSet(dataSourceName, "version_id"),
			),
		})
	}

	// Test lookup by ID (only for non-mock tests where we can get real IDs)
	if !isMock && len(executionEnvironments) > 0 {
		firstEnv := executionEnvironments[0]
		// Get the actual ID from the first environment
		p, ok := testAccProvider.(*Provider)
		if ok {
			p.service = client.NewService(cl)
			traceAPICall("ListExecutionEnvironments")
			envs, err := p.service.ListExecutionEnvironments(context.TODO())
			if err == nil {
				for _, env := range envs {
					if env.Name == firstEnv.Name {
						steps = append(steps, resource.TestStep{
							Config: executionEnvironmentDataSourceConfigByID(env.ID),
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr(dataSourceName, "id", env.ID),
								resource.TestCheckResourceAttr(dataSourceName, "name", env.Name),
								resource.TestCheckResourceAttrSet(dataSourceName, "programming_language"),
								resource.TestCheckResourceAttrSet(dataSourceName, "version_id"),
							),
						})
						// Test lookup by ID with specific version_id
						if env.LatestVersion.ID != "" {
							steps = append(steps, resource.TestStep{
								Config: executionEnvironmentDataSourceConfigWithVersion(env.ID, env.LatestVersion.ID),
								Check: resource.ComposeAggregateTestCheckFunc(
									resource.TestCheckResourceAttr(dataSourceName, "id", env.ID),
									resource.TestCheckResourceAttr(dataSourceName, "name", env.Name),
									resource.TestCheckResourceAttr(dataSourceName, "version_id", env.LatestVersion.ID),
									resource.TestCheckResourceAttrSet(dataSourceName, "programming_language"),
								),
							})
						}
						break
					}
				}
			}
		}
	}

	// Test error case: invalid name
	steps = append(steps, resource.TestStep{
		Config:      executionEnvironmentDataSourceConfigByName("invalid"),
		ExpectError: regexp.MustCompile("Execution Environment not found"),
	})

	// Test error case: missing both id and name
	steps = append(steps, resource.TestStep{
		Config:      executionEnvironmentDataSourceConfigEmpty(),
		ExpectError: regexp.MustCompile("Missing required attributes"),
	})

	resource.Test(t, resource.TestCase{
		IsUnitTest: isMock,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    steps,
	})
}

func executionEnvironmentDataSourceConfigByName(name string) string {
	return fmt.Sprintf(`
data "datarobot_execution_environment" "test" {
	name = "%s"
}
`, name)
}

func executionEnvironmentDataSourceConfigByID(id string) string {
	return fmt.Sprintf(`
data "datarobot_execution_environment" "test" {
	id = "%s"
}
`, id)
}

func executionEnvironmentDataSourceConfigWithVersion(id, versionID string) string {
	return fmt.Sprintf(`
data "datarobot_execution_environment" "test" {
	id = "%s"
	version_id = "%s"
}
`, id, versionID)
}

func executionEnvironmentDataSourceConfigEmpty() string {
	return `
data "datarobot_execution_environment" "test" {
}
`
}

func getExecutionEnvironments() ([]client.ExecutionEnvironment, error) {
	p, ok := testAccProvider.(*Provider)
	if !ok {
		return nil, fmt.Errorf("Provider not found")
	}
	p.service = client.NewService(cl)

	traceAPICall("ListExecutionEnvironments")
	executionEnvironments, err := p.service.ListExecutionEnvironments(context.TODO())
	if err != nil {
		return nil, err
	}

	publicExecutionEnvironments := []client.ExecutionEnvironment{}
	for _, executionEnvironment := range executionEnvironments {
		if executionEnvironment.IsPublic && executionEnvironment.LatestVersion.ID != "" {
			publicExecutionEnvironments = append(publicExecutionEnvironments, executionEnvironment)
		}
	}

	return publicExecutionEnvironments, nil
}
