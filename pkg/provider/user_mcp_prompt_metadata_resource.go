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
var _ resource.Resource = &UserMCPPromptMetadataResource{}
var _ resource.ResourceWithImportState = &UserMCPPromptMetadataResource{}

func NewUserMCPPromptMetadataResource() resource.Resource {
	return &UserMCPPromptMetadataResource{}
}

// UserMCPPromptMetadataResource defines the resource implementation.
type UserMCPPromptMetadataResource struct {
	provider *Provider
}

func (r *UserMCPPromptMetadataResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_mcp_prompt_metadata"
}

func (r *UserMCPPromptMetadataResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "User MCP prompt metadata. This resource creates a prompt metadata entry for a given MCP server version using the User MCP public API.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the User MCP prompt metadata.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the MCP prompt.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the MCP prompt.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mcp_server_version_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the MCP server version this prompt belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the MCP prompt is created.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "The id of the user who created the MCP prompt.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_name": schema.StringAttribute{
				MarkdownDescription: "The name of the user who created the MCP prompt",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *UserMCPPromptMetadataResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserMCPPromptMetadataResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserMCPPromptMetadataResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateUserMCPPromptMetadata")
	request_payload := &client.UserMCPPromptMetadataRequest{
		Name: data.Name.ValueString(),
		Type: data.Type.ValueString(),
	}
	mcp_server_version_id := data.MCPServerVersionID.ValueString()

	createResp, err := r.provider.service.CreateUserMCPPromptMetadata(ctx, mcp_server_version_id, request_payload)

	if err != nil {
		resp.Diagnostics.AddError("Error creating User MCP prompt metadata", err.Error())
		return
	}
	data.ID = types.StringValue(createResp.ID)
	data.Name = types.StringValue(createResp.Name)
	data.Type = types.StringValue(createResp.Type)
	data.CreatedAt = types.StringValue(createResp.CreatedAt)
	data.UserId = types.StringValue(createResp.UserId)
	data.UserName = types.StringValue(createResp.UserName)
	data.MCPServerVersionID = types.StringValue(createResp.MCPServerVersionID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *UserMCPPromptMetadataResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserMCPPromptMetadataResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("GetUserMCPPromptMetadata is not supported")

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *UserMCPPromptMetadataResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserMCPPromptMetadataResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteUserMCPPromptMetadata is not supported")
}

func (r *UserMCPPromptMetadataResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserMCPPromptMetadataResourceModel

	// Resource is immutable; no API update. Echo state back so Terraform sees no change.
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("UpdateUserMCPPromptMetadata is not supported")

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *UserMCPPromptMetadataResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
