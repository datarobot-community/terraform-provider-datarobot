package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &DeploymentSharedRoleResource{}
var _ resource.ResourceWithImportState = &DeploymentSharedRoleResource{}

// defaultDeploymentShareRole is the read only tier for a deployment. Consumers
// can make predictions against it. USER is the tier that also makes the
// deployment visible.
const defaultDeploymentShareRole = "CONSUMER"

func NewDeploymentSharedRoleResource() resource.Resource {
	return &DeploymentSharedRoleResource{}
}

type DeploymentSharedRoleResource struct {
	provider *Provider
}

func (r *DeploymentSharedRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_shared_role"
}

func (r *DeploymentSharedRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Grants a directory group a role on a deployment, addressing the group by name rather than by ID.\n\n" +
			"Pipelines know a group by the name it has in the identity provider, but the sharing API needs the internal group ID. " +
			"This resource resolves the one to the other, so no group ID has to be hardcoded.\n\n" +
			"Destroying the resource revokes the grant. Changing `group_name` revokes the previous group before granting the new one.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the grant, in the form `<deployment_id>:<group_id>`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployment_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the deployment to share.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the group to grant the role to, matched in full and case sensitively.",
			},
			"group_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID the group name resolved to.",
			},
			"role": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(defaultDeploymentShareRole),
				MarkdownDescription: "The role to grant. One of `OWNER`, `USER` or `CONSUMER`. Defaults to `CONSUMER`, which allows predictions but does not make the deployment visible; use `USER` if the group needs to see it.",
			},
		},
	}
}

func (r *DeploymentSharedRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DeploymentSharedRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeploymentSharedRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("ListDirectoryEntities")
	group, err := resolveGroupByName(ctx, r.provider.service, data.GroupName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to look up group", err.Error())
		return
	}

	traceAPICall("UpdateDeploymentSharedRoles")
	if err := r.setRole(ctx, data.DeploymentID.ValueString(), group.ID, data.Role.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error sharing deployment with group", err.Error())
		return
	}

	data.GroupID = types.StringValue(group.ID)
	data.ID = types.StringValue(sharedRoleID(data.DeploymentID.ValueString(), group.ID))

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentSharedRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeploymentSharedRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("ListDeploymentSharedRoles")
	sharedRoles, err := r.provider.service.ListDeploymentSharedRoles(ctx, data.DeploymentID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Deployment not found",
				fmt.Sprintf("Deployment with ID %s is not found. Removing from state.", data.DeploymentID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting shared roles for deployment %s", data.DeploymentID.ValueString()),
				err.Error())
		}
		return
	}

	groupID := data.GroupID.ValueString()
	for _, sharedRole := range sharedRoles.Data {
		if sharedRole.ShareRecipientType == client.ShareRecipientTypeGroup && sharedRole.ID == groupID {
			data.Role = types.StringValue(sharedRole.Role)
			if sharedRole.Name != "" {
				data.GroupName = types.StringValue(sharedRole.Name)
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	// The grant was revoked outside Terraform.
	resp.Diagnostics.AddWarning(
		"Shared role not found",
		fmt.Sprintf("Group %s no longer has a role on deployment %s. Removing from state.", groupID, data.DeploymentID.ValueString()))
	resp.State.RemoveResource(ctx)
}

func (r *DeploymentSharedRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeploymentSharedRoleResourceModel
	var state DeploymentSharedRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deploymentID := plan.DeploymentID.ValueString()

	traceAPICall("ListDirectoryEntities")
	group, err := resolveGroupByName(ctx, r.provider.service, plan.GroupName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to look up group", err.Error())
		return
	}

	// Revoke the previous group first when the name now resolves elsewhere.
	// Without this, repointing group_name would leave the old grant in place.
	previousGroupID := state.GroupID.ValueString()
	if previousGroupID != "" && previousGroupID != group.ID {
		traceAPICall("UpdateDeploymentSharedRoles")
		if err := r.setRole(ctx, deploymentID, previousGroupID, client.SharedRoleNone); err != nil {
			resp.Diagnostics.AddError("Error revoking the previous group's role", err.Error())
			return
		}
	}

	traceAPICall("UpdateDeploymentSharedRoles")
	if err := r.setRole(ctx, deploymentID, group.ID, plan.Role.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error updating the group's role on the deployment", err.Error())
		return
	}

	plan.GroupID = types.StringValue(group.ID)
	plan.ID = types.StringValue(sharedRoleID(deploymentID, group.ID))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *DeploymentSharedRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeploymentSharedRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("UpdateDeploymentSharedRoles")
	err := r.setRole(ctx, data.DeploymentID.ValueString(), data.GroupID.ValueString(), client.SharedRoleNone)
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			// The deployment is already gone, so the grant is moot.
			return
		}
		resp.Diagnostics.AddError("Error revoking the group's role on the deployment", err.Error())
	}
}

func (r *DeploymentSharedRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	deploymentID, groupID, ok := strings.Cut(req.ID, ":")
	if !ok || deploymentID == "" || groupID == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected an identifier of the form <deployment_id>:<group_id>, got %q.", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deployment_id"), deploymentID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), groupID)...)
}

// setRole grants or revokes a single group's role on a deployment. Passing
// client.SharedRoleNone revokes. updateRoles has set rather than append
// semantics, so re-sending an unchanged role is a no-op.
func (r *DeploymentSharedRoleResource) setRole(ctx context.Context, deploymentID, groupID, role string) error {
	return r.provider.service.UpdateDeploymentSharedRoles(ctx, deploymentID, &client.UpdateSharedRolesRequest{
		Operation: client.SharedRolesOperationUpdate,
		Roles: []client.SharedRoleGrant{
			{
				ShareRecipientType: client.ShareRecipientTypeGroup,
				ID:                 groupID,
				Role:               role,
			},
		},
	})
}

func sharedRoleID(deploymentID, groupID string) string {
	return deploymentID + ":" + groupID
}
