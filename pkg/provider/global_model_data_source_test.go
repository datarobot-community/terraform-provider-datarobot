package provider

import (
	"context"
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

func TestAccGlobalModelDataSource(t *testing.T) {
	t.Parallel()

	testGlobalModelDataSource(t, false)
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

	globalModels, err := getGlobalModels()
	if err != nil {
		t.Fatal(err)
	}

	for _, globalModel := range globalModels {
		id := uuid.NewString()
		versionID := uuid.NewString()
		versionNum := 2

		for i := 0; i < 3; i++ {
			mockService.EXPECT().ListRegisteredModels(gomock.Any(), &client.ListRegisteredModelsRequest{
				IsGlobal: true,
				Search:   globalModel.Name,
			}).Return([]client.RegisteredModel{
				{
					ID:             id,
					Name:           globalModel.Name,
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

	testGlobalModelDataSource(t, true)
}

func testGlobalModelDataSource(t *testing.T, isMock bool) {
	dataSourceName := "data.datarobot_global_model.test"

	steps := []resource.TestStep{}

	globalModels, err := getGlobalModels()
	if err != nil {
		t.Fatal(err)
	}

	for _, globalModel := range globalModels {
		steps = append(steps, resource.TestStep{
			Config: globalModelDataSourceConfig(globalModel.Name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(dataSourceName, "name", globalModel.Name),
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

func getGlobalModels() ([]client.RegisteredModel, error) {
	p, ok := testAccProvider.(*Provider)
	if !ok {
		return nil, fmt.Errorf("Provider not found")
	}
	p.service = client.NewService(cl)

	traceAPICall("ListRegisteredModels")
	registeredModels, err := p.service.ListRegisteredModels(context.TODO(), &client.ListRegisteredModelsRequest{
		IsGlobal: true,
	})
	if err != nil {
		return nil, err
	}

	return registeredModels, nil
}
