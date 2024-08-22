package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/omnistrate/terraform-provider-datarobot/internal/client"
	mock_client "github.com/omnistrate/terraform-provider-datarobot/mock"
)

func TestAccUseCaseResource(t *testing.T) {
	t.Parallel()
	testUseCaseResource(t, uuid.NewString(), uuid.NewString(), false)
}

func TestIntegrationUseCaseResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if os.Getenv(DataRobotApiKeyEnvVar) == "" {
		os.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	id := uuid.NewString()
	name := uuid.NewString()
	description := uuid.NewString()

	// Create
	mockService.EXPECT().CreateUseCase(gomock.Any(), &client.UseCaseRequest{
		Name:        name,
		Description: description,
	}).Return(&client.CreateUseCaseResponse{
		ID: id,
	}, nil)
	mockService.EXPECT().GetUseCase(gomock.Any(), id).Return(&client.UseCaseResponse{
		ID:          id,
		Name:        name,
		Description: description,
	}, nil)

	// Read
	mockService.EXPECT().GetUseCase(gomock.Any(), id).Return(&client.UseCaseResponse{
		ID:          id,
		Name:        name,
		Description: description,
	}, nil)

	// Update
	mockService.EXPECT().GetUseCase(gomock.Any(), id).Return(&client.UseCaseResponse{
		ID:          id,
		Name:        name,
		Description: description,
	}, nil)
	mockService.EXPECT().UpdateUseCase(gomock.Any(), id, &client.UseCaseRequest{
		Name:        "new_" + name,
		Description: "new_" + description,
	}).Return(&client.UseCaseResponse{
		ID:          id,
		Name:        "new_" + name,
		Description: "new_" + description,
	}, nil)
	mockService.EXPECT().GetUseCase(gomock.Any(), id).Return(&client.UseCaseResponse{
		ID:          id,
		Name:        "new_" + name,
		Description: "new_" + description,
	}, nil)

	// Delete
	mockService.EXPECT().GetUseCase(gomock.Any(), id).Return(&client.UseCaseResponse{
		ID:          id,
		Name:        "new_" + name,
		Description: "new_" + description,
	}, nil)
	mockService.EXPECT().DeleteUseCase(gomock.Any(), id).Return(nil)

	testUseCaseResource(t, name, description, true)
}

func testUseCaseResource(t *testing.T, name, description string, isMock bool) {
	resourceName := "datarobot_use_case.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest: isMock,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: useCaseResourceConfig(name, description),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkUseCaseResourceExists(resourceName),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name and description
			{
				Config: useCaseResourceConfig("new_"+name, "new_"+description),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkUseCaseResourceExists(resourceName),
					// Verify name and description
					resource.TestCheckResourceAttr(resourceName, "name", "new_"+name),
					resource.TestCheckResourceAttr(resourceName, "description", "new_"+description),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func useCaseResourceConfig(name, description string) string {
	return fmt.Sprintf(`
resource "datarobot_use_case" "test" {
	  name = "%s"
	  description = "%s"
}
`, name, description)
}

func checkUseCaseResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetUseCase")
		useCase, err := p.service.GetUseCase(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if useCase.Name == rs.Primary.Attributes["name"] &&
			useCase.Description == rs.Primary.Attributes["description"] {
			return nil
		}

		return fmt.Errorf("Use case not found")
	}
}
