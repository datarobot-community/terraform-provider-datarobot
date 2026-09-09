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
	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestGroupDataSourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwdatasource.SchemaRequest{}
	schemaResponse := &fwdatasource.SchemaResponse{}

	NewGroupDataSource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func TestUnitGroupDataSource(t *testing.T) {
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

	const groupID = "687b1c6f4e1a2b0001c0ffee"

	mockService.EXPECT().ListDirectoryEntities(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *client.ListDirectoryEntitiesRequest) (*client.ListDirectoryEntitiesResponse, error) {
			if req.EntityType != "group" {
				return nil, fmt.Errorf("expected the search to be scoped to groups, got %q", req.EntityType)
			}

			switch req.Name {
			case "FinanceTeam":
				description := "Finance"
				memberCount := 12
				return &client.ListDirectoryEntitiesResponse{
					Count:      1,
					TotalCount: 1,
					Data: []client.DirectoryEntity{{
						ID:                 groupID,
						Name:               req.Name,
						EntityType:         "group",
						Description:        &description,
						ProvisioningSource: "scim",
						MemberCount:        &memberCount,
					}},
				}, nil
			case "Ambiguous":
				return &client.ListDirectoryEntitiesResponse{
					Count:      2,
					TotalCount: 2,
					Data: []client.DirectoryEntity{
						{ID: "grp-1", Name: req.Name, EntityType: "group"},
						{ID: "grp-2", Name: req.Name, EntityType: "group"},
					},
				}, nil
			default:
				// The API returns an empty page rather than a 404 for a name
				// that matches nothing, usually a casing mistake.
				return &client.ListDirectoryEntitiesResponse{}, nil
			}
		}).AnyTimes()

	dataSourceName := "data.datarobot_group.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: groupDataSourceConfig("FinanceTeam"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", groupID),
					resource.TestCheckResourceAttr(dataSourceName, "name", "FinanceTeam"),
					resource.TestCheckResourceAttr(dataSourceName, "description", "Finance"),
					resource.TestCheckResourceAttr(dataSourceName, "provisioning_source", "scim"),
				),
			},
			{
				// A name that matches nothing has to say so rather than
				// resolving to an empty ID.
				Config:      groupDataSourceConfig("financeteam"),
				ExpectError: regexp.MustCompile(`no group named "financeteam" was found`),
			},
			{
				// Two groups can share a name, and picking one silently would
				// grant the wrong people access.
				Config:      groupDataSourceConfig("Ambiguous"),
				ExpectError: regexp.MustCompile(`expected exactly one group named "Ambiguous", found 2`),
			},
		},
	})
}

func groupDataSourceConfig(name string) string {
	return fmt.Sprintf(`
data "datarobot_group" "test" {
	name = "%s"
}
`, name)
}
