package provider

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccUserMcpResourceMetadataResource(t *testing.T) {
	t.Parallel()

	resourceResourceId := "test_mpc_resource_metadata"
	resourceName := "datarobot_user_mcp_resource_metadata." + resourceResourceId
	mcpResourceName := "test-resource-" + uuid.NewString()[:8]
	mcpResourceType := "userResource"
	mcpResourceUri := "uri://sasada"
	baseEnvironmentID := testGenAIBaseEnvID

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccFeatureFlagPreCheck(t, "ENABLE_MCP_TOOLS_GALLERY_SUPPORT")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with only required attributes; verify state (Read is not implemented; state comes from Create response)
			{
				Config: userMcpresourceMetadataAndCustomModelVersionConfig(resourceResourceId, mcpResourceName, mcpResourceType, mcpResourceUri, baseEnvironmentID),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkUserMcpresourceMetadataResourceInState(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", mcpResourceName),
					resource.TestCheckResourceAttr(resourceName, "type", mcpResourceType),
					resource.TestCheckResourceAttr(resourceName, "uri", mcpResourceUri),
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

func userMcpresourceMetadataAndCustomModelVersionConfig(resourceResourceId, mcpResourceName, mcpResourceType, mcpResourceUri, baseEnvironmentID string) string {
	return fmt.Sprintf(`
resource "datarobot_custom_model" "mcp_sever_version" {
  name                  = "mcp-for-user-resource-metadata-test"
  target_type            = "MCP"
  target_name            = "test_target"
  language               = "python"
  base_environment_id    = "%s"
}

resource "datarobot_user_mcp_resource_metadata" "%s" {
  mcp_server_version_id = datarobot_custom_model.mcp_sever_version.version_id
  name                  = "%s"
  type                  = "%s"
  uri                   = "%s"
}
`, baseEnvironmentID, resourceResourceId, mcpResourceName, mcpResourceType, mcpResourceUri)
}

// checkUserMcpresourceMetadataResourceInState verifies the resource exists in state and has an ID.
// The User MCP resource Metadata API does not expose a Get by ID in this provider, so we only check state.
func checkUserMcpresourceMetadataResourceInState(resourceName string) resource.TestCheckFunc {
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
