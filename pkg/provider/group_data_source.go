package provider

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &GroupDataSource{}

type GroupDataSource struct {
	provider *Provider
}

func NewGroupDataSource() datasource.DataSource {
	return &GroupDataSource{}
}

func (d *GroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *GroupDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a directory group by name and returns its ID, so that group memberships and shares can be expressed by name rather than by hardcoded ID.\n\n" +
			"Names are matched in full and case sensitively, so the name must be given exactly as it appears in the identity provider. " +
			"Results are scoped to the caller's organization.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the group, matched in full and case sensitively.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the group.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the group, or null if none is set.",
			},
			"provisioning_source": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "How the group is provisioned: `scim` when synced from an external identity provider, `local` when created in DataRobot.",
			},
		},
	}
}

func (d *GroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	var ok bool
	if d.provider, ok = req.ProviderData.(*Provider); !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected  %T, got: %T. Please report this issue to the provider developers.", Provider{}, req.ProviderData),
		)
	}
}

func (d *GroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config GroupDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := config.Name.ValueString()

	traceAPICall("ListDirectoryEntities")
	group, err := resolveGroupByName(ctx, d.provider.service, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to look up group", err.Error())
		return
	}

	config.ID = types.StringValue(group.ID)
	config.ProvisioningSource = types.StringValue(group.ProvisioningSource)
	if group.Description != nil {
		config.Description = types.StringValue(*group.Description)
	} else {
		config.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// resolveGroupByName resolves a group name to exactly one directory entity.
//
// The Directory Search API returns every match with a total count rather than
// erroring when a name is ambiguous, so the count is asserted here. Without
// that check an ambiguous name would silently resolve to whichever match
// happened to sort first.
func resolveGroupByName(ctx context.Context, svc client.Service, name string) (*client.DirectoryEntity, error) {
	resp, err := svc.ListDirectoryEntities(ctx, &client.ListDirectoryEntitiesRequest{
		EntityType: "group",
		Name:       name,
	})
	if err != nil {
		return nil, err
	}

	switch {
	case resp.TotalCount == 0:
		return nil, fmt.Errorf(
			"no group named %q was found. Names are matched in full and case sensitively, so check the capitalisation matches the identity provider",
			name)
	case resp.TotalCount > 1:
		return nil, fmt.Errorf(
			"expected exactly one group named %q, found %d. Group names are unique case sensitively, so more than one match means the name is ambiguous",
			name, resp.TotalCount)
	case len(resp.Data) == 0:
		return nil, fmt.Errorf("group %q reported a total count of 1 but returned no entities", name)
	}

	return &resp.Data[0], nil
}
