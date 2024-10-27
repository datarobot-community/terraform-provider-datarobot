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

var globalModelNames = []string{
	"[Hugging Face] Zero-shot Classifier",
	"[Hugging Face] Toxicity Classifier",
	"[Hugging Face] Sentiment Classifier",
	"[Hugging Face] Emotions Classifier",
	"[DataRobot] Dummy Binary Classification",
}

func TestAccGlobalModelDataSource(t *testing.T) {
	t.Parallel()

	testGlobalModelDataSource(t, globalModelNames, false)
}

func TestIntegrationGlobalModelDataSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if os.Getenv(DataRobotApiKeyEnvVar) == "" {
		os.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	for _, name := range globalModelNames {
		id := uuid.NewString()
		versionID := uuid.NewString()
		versionNum := 2

		for i := 0; i < 3; i++ {
			mockService.EXPECT().ListRegisteredModels(gomock.Any(), &client.ListRegisteredModelsRequest{
				IsGlobal: true,
				Search:   name,
			}).Return([]client.RegisteredModel{
				{
					ID:             id,
					Name:           name,
					LastVersionNum: versionNum,
					IsGlobal:       true,
				},
			}, nil)
			mockService.EXPECT().GetLatestRegisteredModelVersion(gomock.Any(), id).Return(
				&client.RegisteredModelVersion{
					ID:                     versionID,
					RegisteredModelVersion: versionNum,
				}, nil)
		}
	}

	mockService.EXPECT().ListRegisteredModels(gomock.Any(), &client.ListRegisteredModelsRequest{
		IsGlobal: true,
		Search:   "invalid",
	}).Return([]client.RegisteredModel{}, nil)

	testGlobalModelDataSource(t, globalModelNames, true)
}

func testGlobalModelDataSource(t *testing.T, names []string, isMock bool) {
	dataSourceName := "data.datarobot_global_model.test"

	steps := []resource.TestStep{}

	for _, name := range names {
		steps = append(steps, resource.TestStep{
			Config: globalModelDataSourceConfig(name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(dataSourceName, "name", name),
				resource.TestCheckResourceAttrSet(dataSourceName, "id"),
				resource.TestCheckResourceAttrSet(dataSourceName, "version_id"),
			),
		})
	}

	steps = append(steps, resource.TestStep{
		Config:      globalModelDataSourceConfig("invalid"),
		ExpectError: regexp.MustCompile("Global Model not found"),
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

func globalModelDataSourceConfig(name string) string {
	return fmt.Sprintf(`
data "datarobot_global_model" "test" {
	  name = "%s"
}
`, name)
}
