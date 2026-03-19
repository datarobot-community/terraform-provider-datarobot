package provider

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccUserMcpToolMetadataResource(t *testing.T) {
	t.Parallel()

	toolResourceId := "test_mpc_tool_metadata"
	resourceName := "datarobot_user_mcp_tool_metadata." + toolResourceId
	toolName := "test-tool-" + uuid.NewString()[:8]
	toolType := "userTool"
	baseEnvironmentID := testGenAIBaseEnvID

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccFeatureFlagPreCheck(t, "ENABLE_MCP_TOOLS_GALLERY_SUPPORT")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with only required attributes; verify state (Read is not implemented; state comes from Create response)
			{
				Config: userMcpToolMetadataAndCustomModelVersionConfig(toolResourceId, toolName, toolType, baseEnvironmentID),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkUserMcpToolMetadataResourceInState(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", toolName),
					resource.TestCheckResourceAttr(resourceName, "type", toolType),
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

func userMcpToolMetadataAndCustomModelVersionConfig(toolResourceId, toolName, toolType, baseEnvironmentID string) string {
	return fmt.Sprintf(`
resource "datarobot_custom_model" "mcp_sever_version" {
  name                  = "mcp-for-user-tool-metadata-test"
  target_type            = "MCP"
  target_name            = "test_target"
  language               = "python"
  base_environment_id    = "%s"
}

resource "datarobot_user_mcp_tool_metadata" "%s" {
  mcp_server_version_id = datarobot_custom_model.mcp_sever_version.version_id
  name                  = "%s"
  type                  = "%s"
}
`, baseEnvironmentID, toolResourceId, toolName, toolType)
}

// checkUserMcpToolMetadataResourceInState verifies the resource exists in state and has an ID.
// The User MCP Tool Metadata API does not expose a Get by ID in this provider, so we only check state.
func checkUserMcpToolMetadataResourceInState(resourceName string) resource.TestCheckFunc {
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
