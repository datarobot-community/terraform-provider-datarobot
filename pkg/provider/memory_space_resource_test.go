package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
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
	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "ENABLE_AGENTIC_MEMORY_API").Return(true, nil)
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
	emptyStr := ""
	mockService.EXPECT().UpdateMemorySpace(gomock.Any(), id, &client.MemorySpaceRequest{
		Description:        &newDescription,
		LLMModelName:       &emptyStr,
		LLMBaseURL:         &emptyStr,
		CustomInstructions: &emptyStr,
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
		Description:        &emptyDesc,
		LLMModelName:       &emptyDesc,
		LLMBaseURL:         &emptyDesc,
		CustomInstructions: &emptyDesc,
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

func TestIntegrationMemorySpaceResourceNewFields(t *testing.T) {
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
	modelName := "gpt-4o"
	baseURL := "https://api.openai.com/v1"
	instructions := "Be concise."
	updatedInstructions := "Be very concise."

	// Create
	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "ENABLE_AGENTIC_MEMORY_API").Return(true, nil)
	mockService.EXPECT().CreateMemorySpace(gomock.Any(), &client.MemorySpaceRequest{
		LLMModelName:       &modelName,
		LLMBaseURL:         &baseURL,
		CustomInstructions: &instructions,
	}).Return(&client.MemorySpaceResponse{
		MemorySpaceID:      id,
		LLMModelName:       modelName,
		LLMBaseURL:         baseURL,
		CustomInstructions: instructions,
	}, nil)
	mockService.EXPECT().GetMemorySpace(gomock.Any(), id).Return(&client.MemorySpaceResponse{
		MemorySpaceID:      id,
		LLMModelName:       modelName,
		LLMBaseURL:         baseURL,
		CustomInstructions: instructions,
	}, nil)

	// Read (plan refresh before update)
	mockService.EXPECT().GetMemorySpace(gomock.Any(), id).Return(&client.MemorySpaceResponse{
		MemorySpaceID:      id,
		LLMModelName:       modelName,
		LLMBaseURL:         baseURL,
		CustomInstructions: instructions,
	}, nil)

	// Update
	emptyDesc := ""
	mockService.EXPECT().UpdateMemorySpace(gomock.Any(), id, &client.MemorySpaceRequest{
		Description:        &emptyDesc,
		LLMModelName:       &modelName,
		LLMBaseURL:         &baseURL,
		CustomInstructions: &updatedInstructions,
	}).Return(&client.MemorySpaceResponse{
		MemorySpaceID:      id,
		LLMModelName:       modelName,
		LLMBaseURL:         baseURL,
		CustomInstructions: updatedInstructions,
	}, nil)
	mockService.EXPECT().GetMemorySpace(gomock.Any(), id).Return(&client.MemorySpaceResponse{
		MemorySpaceID:      id,
		LLMModelName:       modelName,
		LLMBaseURL:         baseURL,
		CustomInstructions: updatedInstructions,
	}, nil)

	// Delete
	mockService.EXPECT().DeleteMemorySpace(gomock.Any(), id).Return(nil)

	resourceName := "datarobot_memory_space.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: memorySpaceResourceConfigWithNewFields(modelName, baseURL, instructions),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "llm_model_name", modelName),
					resource.TestCheckResourceAttr(resourceName, "llm_base_url", baseURL),
					resource.TestCheckResourceAttr(resourceName, "custom_instructions", instructions),
				),
			},
			// Update custom_instructions
			{
				Config: memorySpaceResourceConfigWithNewFields(modelName, baseURL, updatedInstructions),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "custom_instructions", updatedInstructions),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestIntegrationMemorySpaceResourceValidation(t *testing.T) {
	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	tests := []struct {
		name        string
		config      string
		expectError string
	}{
		{
			name:        "llm_model_name too long",
			config:      memorySpaceResourceConfigLLMModelName(strings.Repeat("a", 201)),
			expectError: "string length must be at most 200",
		},
		{
			name:        "llm_base_url empty",
			config:      memorySpaceResourceConfigLLMBaseURL(""),
			expectError: "string length must be between 1 and 2083",
		},
		{
			name:        "llm_base_url too long",
			config:      memorySpaceResourceConfigLLMBaseURL(strings.Repeat("a", 2084)),
			expectError: "string length must be between 1 and 2083",
		},
		{
			name:        "custom_instructions too long",
			config:      memorySpaceResourceConfigCustomInstructions(strings.Repeat("a", 10001)),
			expectError: "string length must be at most 10000",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      tc.config,
						ExpectError: regexp.MustCompile(tc.expectError),
					},
				},
			})
		})
	}
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

	mockService.EXPECT().IsFeatureFlagEnabled(gomock.Any(), "ENABLE_AGENTIC_MEMORY_API").Return(false, nil)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      memorySpaceResourceConfig("test-description"),
				ExpectError: regexp.MustCompile("ENABLE_AGENTIC_MEMORY_API feature flag is not enabled"),
			},
		},
	})
}

func testMemorySpaceResource(t *testing.T, description string, isMock bool) {
	resourceName := "datarobot_memory_space.test"
	var preCheck func()
	if !isMock {
		preCheck = func() { testAccFeatureFlagPreCheck(t, "ENABLE_AGENTIC_MEMORY_API") }
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

func memorySpaceResourceConfigWithNewFields(modelName, baseURL, instructions string) string {
	return fmt.Sprintf(`
resource "datarobot_memory_space" "test" {
  llm_model_name       = "%s"
  llm_base_url         = "%s"
  custom_instructions  = "%s"
}
`, modelName, baseURL, instructions)
}

func memorySpaceResourceConfigLLMModelName(name string) string {
	return fmt.Sprintf(`
resource "datarobot_memory_space" "test" {
  llm_model_name = %q
}
`, name)
}

func memorySpaceResourceConfigLLMBaseURL(url string) string {
	return fmt.Sprintf(`
resource "datarobot_memory_space" "test" {
  llm_base_url = %q
}
`, url)
}

func memorySpaceResourceConfigCustomInstructions(instructions string) string {
	return fmt.Sprintf(`
resource "datarobot_memory_space" "test" {
  custom_instructions = %q
}
`, instructions)
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
