package application

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/common"
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AppOAuthResource{}
var _ resource.ResourceWithImportState = &AppOAuthResource{}

func NewAppOAuthResource() resource.Resource {
	return &AppOAuthResource{}
}

type AppOAuthResource struct {
	service client.Service
}

func (r *AppOAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_oauth"
}

func (r *AppOAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Resource for managing OAuth providers in DataRobot. This resource allows you to create, read, update, and delete OAuth provider configurations.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the OAuth provider.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the OAuth provider.",
			},
			"org_id": schema.StringAttribute{
				Computed:    true,
				Description: "Organization ID associated with the OAuth provider.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Type of the OAuth provider, e.g., 'google', 'box', etc.",
			},

			"client_id": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "OAuth client ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"client_secret": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "OAuth client secret.",
			},
			"secure_config_id": schema.StringAttribute{
				Computed:    true,
				Description: "Secure config ID for the OAuth provider.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the OAuth provider.",
			},
		},
	}
}

func (r *AppOAuthResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
		if req.ProviderData == nil { return }
	accessor, ok := req.ProviderData.(common.ServiceAccessor)
	if !ok {
			resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
			return
	}
	r.service = accessor.GetService()
}

func (r *AppOAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.AppOAuthResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	request := &client.CreateAppOAuthProviderRequest{
		Name:         data.Name.ValueString(),
		Type:         data.Type.ValueString(),
		ClientID:     data.ClientID.ValueString(),
		ClientSecret: data.ClientSecret.ValueString(),
	}

	common.TraceAPICall("CreateAppOAuthProvider")
	createResp, err := r.service.CreateAppOAuthProvider(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating OAuth provider",
			fmt.Sprintf("Could not create OAuth provider: %s", err),
		)
		return
	}
	data.ID = types.StringValue(createResp.ID)
	data.SecureConfigID = types.StringValue(createResp.SecureConfigID)
	data.OrgID = types.StringValue(createResp.OrgID)
	data.Status = types.StringValue(createResp.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppOAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.AppOAuthResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	common.TraceAPICall("GetAppOAuthProvider")
	getResp, err := r.service.GetAppOAuthProvider(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"OAuth provider not found",
				fmt.Sprintf("OAuth provider with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting OAuth provider with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.ID = types.StringValue(getResp.ID)
	data.Name = types.StringValue(getResp.Name)
	data.OrgID = types.StringValue(getResp.OrgID)
	data.Type = types.StringValue(getResp.Type)
	data.ClientID = types.StringValue(getResp.ClientID)
	data.SecureConfigID = types.StringValue(getResp.SecureConfigID)
	data.Status = types.StringValue(getResp.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *AppOAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.AppOAuthResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state models.AppOAuthResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.IsNull() {
		resp.Diagnostics.AddError(
			"Invalid Resource ID",
			"Cannot update OAuth provider without a valid ID.",
		)
		return
	}

	request := &client.UpdateAppOAuthProviderRequest{}
	// Only update fields that have changed
	if plan.Name != state.Name && !plan.Name.IsNull() {
		request.Name = plan.Name.ValueString()
	}
	if plan.ClientSecret != state.ClientSecret && !plan.ClientSecret.IsNull() {
		request.ClientSecret = plan.ClientSecret.ValueString()
	}

	if request.Name == "" && request.ClientSecret == "" {
		resp.Diagnostics.AddWarning(
			"Invalid Update Request",
			"At least one of 'name' or 'client_secret' must be provided for update.",
		)
		return
	}
	common.TraceAPICall("UpdateAppOAuthProvider")
	updateResp, err := r.service.UpdateAppOAuthProvider(ctx, state.ID.ValueString(), request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating OAuth provider",
			fmt.Sprintf("Could not update OAuth provider with ID %s: %s", plan.ID.ValueString(), err),
		)
		return
	}
	plan.ID = types.StringValue(updateResp.ID)
	plan.Name = types.StringValue(updateResp.Name)
	plan.OrgID = types.StringValue(updateResp.OrgID)
	plan.Type = types.StringValue(updateResp.Type)
	plan.ClientID = types.StringValue(updateResp.ClientID)
	plan.SecureConfigID = types.StringValue(updateResp.SecureConfigID)
	plan.Status = types.StringValue(updateResp.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppOAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.AppOAuthResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	common.TraceAPICall("DeleteAppOAuthProvider")
	err := r.service.DeleteAppOAuthProvider(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting OAuth provider",
			fmt.Sprintf("Could not delete OAuth provider with ID %s: %s", data.ID.ValueString(), err),
		)
		return
	}
}

func (r *AppOAuthResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
