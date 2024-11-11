package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var executionEnvironmentNames = []string{
	"[DataRobot][NVIDIA] Python 3.11 GenAI",
	"[GenAI] vLLM Inference Server",
	"PyONNX",
}

func TestAccExecutionEnvironmentDataSource(t *testing.T) {
	t.Parallel()

	testExecutionEnvironmentDataSource(t, executionEnvironmentNames, false)
}

func TestIntegrationExecutionEnvironmentDataSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if os.Getenv(DataRobotApiKeyEnvVar) == "" {
		os.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	for _, name := range executionEnvironmentNames {
		id := uuid.NewString()
		versionID := uuid.NewString()
		description := uuid.NewString()
		programmingLanguage := "Python"

		for i := 0; i < 3; i++ {
			mockService.EXPECT().ListExecutionEnvironments(gomock.Any()).Return([]client.ExecutionEnvironment{
				{
					ID:                  id,
					Name:                name,
					Description:         description,
					ProgrammingLanguage: programmingLanguage,
					LatestVersion: client.ExecutionEnvironmentVersion{
						ID: versionID,
					},
				},
			}, nil)
		}
	}

	mockService.EXPECT().ListExecutionEnvironments(gomock.Any()).Return([]client.ExecutionEnvironment{}, nil)

	testExecutionEnvironmentDataSource(t, executionEnvironmentNames, true)
}

func testExecutionEnvironmentDataSource(t *testing.T, names []string, isMock bool) {
	dataSourceName := "data.datarobot_execution_environment.test"

	steps := []resource.TestStep{}

	for _, name := range names {
		steps = append(steps, resource.TestStep{
			Config: executionEnvironmentDataSourceConfig(name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(dataSourceName, "name", name),
				resource.TestCheckResourceAttrSet(dataSourceName, "id"),
				resource.TestCheckResourceAttrSet(dataSourceName, "description"),
				resource.TestCheckResourceAttrSet(dataSourceName, "programming_language"),
				resource.TestCheckResourceAttrSet(dataSourceName, "version_id"),
			),
		})
	}

	steps = append(steps, resource.TestStep{
		Config:      executionEnvironmentDataSourceConfig("invalid"),
		ExpectError: regexp.MustCompile("Execution Environment not found"),
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

func executionEnvironmentDataSourceConfig(name string) string {
	return fmt.Sprintf(`
data "datarobot_execution_environment" "test" {
	  name = "%s"
}
`, name)
}
