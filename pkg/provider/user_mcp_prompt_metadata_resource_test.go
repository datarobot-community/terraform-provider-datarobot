package provider

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccUserMcpPromptMetadataResource(t *testing.T) {
	t.Parallel()

	promptResourceId := "test_mpc_prompt_metadata"
	resourceName := "datarobot_user_mcp_prompt_metadata." + promptResourceId
	promptName := "test-prompt-" + uuid.NewString()[:8]
	promptType := "userPromptTemplate"
	baseEnvironmentID := testGenAIBaseEnvID

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with only required attributes; verify state (Read is not implemented; state comes from Create response)
			{
				Config: userMcpPromptMetadataAndCustomModelVersionConfig(promptResourceId, promptName, promptType, baseEnvironmentID),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkUserMcpPromptMetadataResourceInState(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", promptName),
					resource.TestCheckResourceAttr(resourceName, "type", promptType),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "mcp_server_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "user_id"),
					resource.TestCheckResourceAttrSet(resourceName, "user_name"),
				),
			},
		},
	})
}

func userMcpPromptMetadataAndCustomModelVersionConfig(promptResourceId, promptName, promptType, baseEnvironmentID string) string {
	return fmt.Sprintf(`
resource "datarobot_custom_model" "mcp_sever_version" {
  name                  = "mcp-for-user-prompt-metadata-test"
  target_type            = "MCP"
  target_name            = "test_target"
  language               = "python"
  base_environment_id    = "%s"
}

resource "datarobot_user_mcp_prompt_metadata" "%s" {
  mcp_server_version_id = datarobot_custom_model.mcp_sever_version.version_id
  name                  = "%s"
  type                  = "%s"
}
`, baseEnvironmentID, promptResourceId, promptName, promptType)
}

// checkUserMcpPromptMetadataResourceInState verifies the resource exists in state and has an ID.
// The User MCP Prompt Metadata API does not expose a Get by ID in this provider, so we only check state.
func checkUserMcpPromptMetadataResourceInState(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource %s has no ID", resourceName)
		}
		return nil
	}
}
