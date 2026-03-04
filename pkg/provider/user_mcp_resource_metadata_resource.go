package provider

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &UserMCPResourceMetadataResource{}
var _ resource.ResourceWithImportState = &UserMCPResourceMetadataResource{}

func NewUserMCPResourceMetadataResource() resource.Resource {
	return &UserMCPResourceMetadataResource{}
}

// UserMCPResourceMetadataResource defines the resource implementation.
type UserMCPResourceMetadataResource struct {
	provider *Provider
}

func (r *UserMCPResourceMetadataResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_mcp_resource_metadata"
}

func (r *UserMCPResourceMetadataResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "User MCP resource metadata. This resource creates a resource metadata entry for a given MCP server version using the User MCP public API.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the User MCP resource metadata.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the MCP resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the MCP resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"uri": schema.StringAttribute{
				MarkdownDescription: "The URI of the MCP resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mcp_server_version_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the MCP server version this resource belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the MCP resource is created.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "The id of the user who created the MCP resource.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_name": schema.StringAttribute{
				MarkdownDescription: "The name of the user who created the MCP resource",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *UserMCPResourceMetadataResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	var ok bool
	if r.provider, ok = req.ProviderData.(*Provider); !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected  %T, got: %T. Please report this issue to the provider developers.", Provider{}, req.ProviderData),
		)
	}
}

func (r *UserMCPResourceMetadataResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserMCPResourceMetadataResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateUserMCPResourceMetadata")
	request_payload := &client.UserMCPResourceMetadataRequest{
		Name: data.Name.ValueString(),
		Type: data.Type.ValueString(),
		Uri:  data.Uri.ValueString(),
	}
	mcp_server_version_id := data.MCPServerVersionID.ValueString()

	createResp, err := r.provider.service.CreateUserMCPResourceMetadata(ctx, mcp_server_version_id, request_payload)

	if err != nil {
		resp.Diagnostics.AddError("Error creating User MCP resource metadata", err.Error())
		return
	}
	data.ID = types.StringValue(createResp.ID)
	data.Name = types.StringValue(createResp.Name)
	data.Type = types.StringValue(createResp.Type)
	data.Uri = types.StringValue(createResp.Uri)
	data.CreatedAt = types.StringValue(createResp.CreatedAt)
	data.UserId = types.StringValue(createResp.UserId)
	data.UserName = types.StringValue(createResp.UserName)
	data.MCPServerVersionID = types.StringValue(createResp.MCPServerVersionID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *UserMCPResourceMetadataResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserMCPResourceMetadataResourceModel
	NoOpRead(ctx, req, resp, userMCPReadNoOpMsg, &data)
}

func (r *UserMCPResourceMetadataResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserMCPResourceMetadataResourceModel
	NoOpDelete(ctx, req, resp, userMCPDeleteNoOpMsg, &data)
}

func (r *UserMCPResourceMetadataResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserMCPResourceMetadataResourceModel
	NoOpUpdate(ctx, req, resp, userMCPUpdateNoOpMsg, &data)
}

func (r *UserMCPResourceMetadataResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
