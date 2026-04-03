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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMemorySpaceResource(t *testing.T) {
	t.Parallel()
	testMemorySpaceResource(t, uuid.NewString(), false)
}

func TestIntegrationMemorySpaceResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	id := uuid.NewString()
	description := uuid.NewString()
	newDescription := "new_" + description

	// Create
	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "AGENTIC_MEMORY_API").Return(true, nil)
	mockService.EXPECT().CreateMemorySpace(gomock.Any(), &client.MemorySpaceRequest{
		Description: &description,
	}).Return(&client.MemorySpaceResponse{
		MemorySpaceID: id,
		Description:   description,
	}, nil)
	mockService.EXPECT().GetMemorySpace(gomock.Any(), id).Return(&client.MemorySpaceResponse{
		MemorySpaceID: id,
		Description:   description,
	}, nil)

	// Read
	mockService.EXPECT().GetMemorySpace(gomock.Any(), id).Return(&client.MemorySpaceResponse{
		MemorySpaceID: id,
		Description:   description,
	}, nil)

	// Update
	mockService.EXPECT().GetMemorySpace(gomock.Any(), id).Return(&client.MemorySpaceResponse{
		MemorySpaceID: id,
		Description:   description,
	}, nil)
	mockService.EXPECT().UpdateMemorySpace(gomock.Any(), id, &client.MemorySpaceRequest{
		Description: &newDescription,
	}).Return(&client.MemorySpaceResponse{
		MemorySpaceID: id,
		Description:   newDescription,
	}, nil)
	mockService.EXPECT().GetMemorySpace(gomock.Any(), id).Return(&client.MemorySpaceResponse{
		MemorySpaceID: id,
		Description:   newDescription,
	}, nil)

	// Remove description
	emptyDesc := ""
	mockService.EXPECT().GetMemorySpace(gomock.Any(), id).Return(&client.MemorySpaceResponse{
		MemorySpaceID: id,
		Description:   newDescription,
	}, nil)
	mockService.EXPECT().GetMemorySpace(gomock.Any(), id).Return(&client.MemorySpaceResponse{
		MemorySpaceID: id,
		Description:   newDescription,
	}, nil)
	mockService.EXPECT().UpdateMemorySpace(gomock.Any(), id, &client.MemorySpaceRequest{
		Description: &emptyDesc,
	}).Return(&client.MemorySpaceResponse{
		MemorySpaceID: id,
		Description:   "",
	}, nil)
	mockService.EXPECT().GetMemorySpace(gomock.Any(), id).Return(&client.MemorySpaceResponse{
		MemorySpaceID: id,
		Description:   "",
	}, nil)

	// Delete
	mockService.EXPECT().DeleteMemorySpace(gomock.Any(), id).Return(nil)

	testMemorySpaceResource(t, description, true)
}

func TestIntegrationMemorySpaceResourceFeatureFlagDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "AGENTIC_MEMORY_API").Return(false, nil)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      memorySpaceResourceConfig("test-description"),
				ExpectError: regexp.MustCompile("AGENTIC_MEMORY_API feature flag is not enabled"),
			},
		},
	})
}

func testMemorySpaceResource(t *testing.T, description string, isMock bool) {
	resourceName := "datarobot_memory_space.test"
	var preCheck func()
	if !isMock {
		preCheck = func() { testAccFeatureFlagPreCheck(t, "AGENTIC_MEMORY_API") }
	}
	resource.Test(t, resource.TestCase{
		IsUnitTest:               isMock,
		PreCheck:                 preCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: memorySpaceResourceConfig(description),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkMemorySpaceResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update description
			{
				Config: memorySpaceResourceConfig("new_" + description),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkMemorySpaceResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "description", "new_"+description),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Remove description
			{
				Config: memorySpaceResourceConfigNoDescription(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckNoResourceAttr(resourceName, "description"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func memorySpaceResourceConfigNoDescription() string {
	return `
resource "datarobot_memory_space" "test" {
}
`
}

func memorySpaceResourceConfig(description string) string {
	return fmt.Sprintf(`
resource "datarobot_memory_space" "test" {
	  description = "%s"
}
`, description)
}

func checkMemorySpaceResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetMemorySpace")
		memorySpace, err := p.service.GetMemorySpace(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if memorySpace.Description == rs.Primary.Attributes["description"] {
			return nil
		}

		return fmt.Errorf("Memory Space not found")
	}
}
