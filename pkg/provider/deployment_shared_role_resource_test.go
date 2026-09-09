package provider

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestDeploymentSharedRoleResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewDeploymentSharedRoleResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

// sharedRoleFake stands in for the deployment's access control list, so that a
// read reflects what the preceding grants and revokes actually did rather than
// a canned response.
type sharedRoleFake struct {
	groupIDs map[string]string
	grants   map[string]string
}

func (f *sharedRoleFake) roles() []client.SharedRole {
	names := map[string]string{}
	for name, id := range f.groupIDs {
		names[id] = name
	}

	roles := []client.SharedRole{}
	for id, role := range f.grants {
		roles = append(roles, client.SharedRole{
			ID:                 id,
			Name:               names[id],
			ShareRecipientType: client.ShareRecipientTypeGroup,
			Role:               role,
		})
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i].ID < roles[j].ID })

	return roles
}

func TestUnitDeploymentSharedRoleResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	// globalTestCfg is shared across the package and the first mocked test to
	// run sets it, while t.Setenv restores the variable when that test ends. So
	// the two are checked separately, otherwise a later test finds the key set
	// but the environment empty and the provider refuses to configure.
	if globalTestCfg.ApiKey == "" {
		globalTestCfg.ApiKey = "fake"
	}
	if os.Getenv(DataRobotApiKeyEnvVar) == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	const deploymentID = "68a0000000000000000dep01"
	const firstGroupID = "68a0000000000000000grp01"
	const secondGroupID = "68a0000000000000000grp02"

	fake := &sharedRoleFake{
		groupIDs: map[string]string{"FirstGroup": firstGroupID, "SecondGroup": secondGroupID},
		grants:   map[string]string{},
	}

	mockService.EXPECT().ListDirectoryEntities(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *client.ListDirectoryEntitiesRequest) (*client.ListDirectoryEntitiesResponse, error) {
			id, ok := fake.groupIDs[req.Name]
			if !ok {
				return &client.ListDirectoryEntitiesResponse{}, nil
			}
			return &client.ListDirectoryEntitiesResponse{
				Count:      1,
				TotalCount: 1,
				Data:       []client.DirectoryEntity{{ID: id, Name: req.Name, EntityType: "group"}},
			}, nil
		}).AnyTimes()

	mockService.EXPECT().UpdateDeploymentSharedRoles(gomock.Any(), deploymentID, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, req *client.UpdateSharedRolesRequest) error {
			if req.Operation != client.SharedRolesOperationUpdate {
				return fmt.Errorf("expected the updateRoles operation, got %q", req.Operation)
			}
			for _, grant := range req.Roles {
				if grant.Role == client.SharedRoleNone {
					delete(fake.grants, grant.ID)
					continue
				}
				fake.grants[grant.ID] = grant.Role
			}
			return nil
		}).AnyTimes()

	mockService.EXPECT().ListDeploymentSharedRoles(gomock.Any(), deploymentID).DoAndReturn(
		func(_ context.Context, _ string) ([]client.SharedRole, error) {
			return fake.roles(), nil
		}).AnyTimes()

	resourceName := "datarobot_deployment_shared_role.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read. The role defaults to CONSUMER.
			{
				Config: deploymentSharedRoleResourceConfig(deploymentID, "FirstGroup", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "group_name", "FirstGroup"),
					resource.TestCheckResourceAttr(resourceName, "group_id", firstGroupID),
					resource.TestCheckResourceAttr(resourceName, "role", "CONSUMER"),
					resource.TestCheckResourceAttr(resourceName, "id", deploymentID+":"+firstGroupID),
					checkSharedRoleGranted(fake, firstGroupID, "CONSUMER"),
				),
			},
			// A changed role is an in place update that keeps the same grant.
			{
				Config: deploymentSharedRoleResourceConfig(deploymentID, "FirstGroup", "USER"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "group_id", firstGroupID),
					resource.TestCheckResourceAttr(resourceName, "role", "USER"),
					resource.TestCheckResourceAttr(resourceName, "id", deploymentID+":"+firstGroupID),
					checkSharedRoleGranted(fake, firstGroupID, "USER"),
				),
			},
			// A changed group replaces the resource. Before group_name required
			// replacement the plan kept the old composite id as a known value
			// while apply wrote a new one, which Terraform rejects as an
			// inconsistent result.
			{
				Config: deploymentSharedRoleResourceConfig(deploymentID, "SecondGroup", "USER"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "group_name", "SecondGroup"),
					resource.TestCheckResourceAttr(resourceName, "group_id", secondGroupID),
					resource.TestCheckResourceAttr(resourceName, "id", deploymentID+":"+secondGroupID),
					checkSharedRoleGranted(fake, secondGroupID, "USER"),
					checkSharedRoleRevoked(fake, firstGroupID),
				),
			},
			// Import by <deployment_id>:<group_id>.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     deploymentID + ":" + secondGroupID,
				ImportStateVerify: true,
			},
			// Destroy is tested automatically, and asserted below.
		},
	})

	if len(fake.grants) != 0 {
		t.Errorf("expected destroy to revoke every grant, %d left: %v", len(fake.grants), fake.grants)
	}
}

func checkSharedRoleGranted(fake *sharedRoleFake, groupID, role string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		got, ok := fake.grants[groupID]
		if !ok {
			return fmt.Errorf("expected group %s to hold a grant, the deployment has %v", groupID, fake.grants)
		}
		if got != role {
			return fmt.Errorf("expected group %s to hold %s, got %s", groupID, role, got)
		}
		return nil
	}
}

func checkSharedRoleRevoked(fake *sharedRoleFake, groupID string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if role, ok := fake.grants[groupID]; ok {
			return fmt.Errorf("expected group %s to have been revoked, it still holds %s", groupID, role)
		}
		return nil
	}
}

func deploymentSharedRoleResourceConfig(deploymentID, groupName, role string) string {
	roleLine := ""
	if role != "" {
		roleLine = fmt.Sprintf("\n\trole = \"%s\"", role)
	}

	return fmt.Sprintf(`
resource "datarobot_deployment_shared_role" "test" {
	deployment_id = "%s"
	group_name    = "%s"%s
}
`, deploymentID, groupName, roleLine)
}
