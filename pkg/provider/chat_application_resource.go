package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ChatApplicationResource{}
var _ resource.ResourceWithImportState = &ChatApplicationResource{}

func NewChatApplicationResource() resource.Resource {
	return &ChatApplicationResource{}
}

type ChatApplicationResource struct {
	provider *Provider
}

func (r *ChatApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_chat_application"
}

func (r *ChatApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Chat Application",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Chat Application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The version ID of the Chat Application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The URL of the Chat Application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Chat Application.",
			},
			"deployment_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The deployment ID of the Chat Application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"external_access_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether external access is enabled for the Chat Application.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"external_access_recipients": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "The list of external email addresses that have access to the Chat Application.",
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *ChatApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *ChatApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ChatApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateChatApplication")
	createResp, err := r.provider.service.CreateChatApplication(ctx, &client.CreateChatApplicationRequest{
		DeploymentID: data.DeploymentID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Application", err.Error())
		return
	}

	recipients := make([]string, len(data.ExternalAccessRecipients))
	for i, recipient := range data.ExternalAccessRecipients {
		recipients[i] = recipient.ValueString()
	}

	traceAPICall("UpdateChatApplication")
	_, err = r.provider.service.UpdateApplication(ctx,
		createResp.ID,
		&client.UpdateApplicationRequest{
			Name:                     data.Name.ValueString(),
			ExternalAccessEnabled:    IsKnown(data.ExternalAccessEnabled) && data.ExternalAccessEnabled.ValueBool(),
			ExternalAccessRecipients: recipients,
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Application not found",
				fmt.Sprintf("Application with ID %s is not found. Removing from state.", createResp.ID))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error updating Application", err.Error())
		}
		return
	}

	application, err := waitForApplicationToBeReady(ctx, r.provider.service, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Application not ready", err.Error())
		return
	}
	data.ID = types.StringValue(application.ID)
	data.ApplicationUrl = types.StringValue(application.ApplicationUrl)
	data.ExternalAccessEnabled = types.BoolValue(application.ExternalAccessEnabled)

	traceAPICall("GetChatApplicationSource")
	applicationSource, err := r.provider.service.GetApplicationSource(ctx, application.CustomApplicationSourceID)
	if err != nil {
		resp.Diagnostics.AddError("Error getting Application Source info", err.Error())
		return
	}
	data.VersionID = types.StringValue(applicationSource.LatestVersion.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ChatApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ChatApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetChatApplication")
	application, err := r.provider.service.GetApplication(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Application not found",
				fmt.Sprintf("Application with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Application with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.Name = types.StringValue(application.Name)
	data.ApplicationUrl = types.StringValue(application.ApplicationUrl)
	data.ExternalAccessEnabled = types.BoolValue(application.ExternalAccessEnabled)

	traceAPICall("GetChatApplicationSource")
	applicationSource, err := r.provider.service.GetApplicationSource(ctx, application.CustomApplicationSourceID)
	if err != nil {
		resp.Diagnostics.AddError("Error getting Application Source info", err.Error())
		return
	}
	data.VersionID = types.StringValue(applicationSource.LatestVersion.ID)

	if applicationSource.LatestVersion.RuntimeParameters != nil {
		for _, runtimeParameter := range applicationSource.LatestVersion.RuntimeParameters {
			if runtimeParameter.FieldName == "DEPLOYMENT_ID" {
				data.DeploymentID = types.StringValue(runtimeParameter.CurrentValue.(string))
				break
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ChatApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ChatApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ChatApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	recipients := make([]string, len(plan.ExternalAccessRecipients))
	for i, recipient := range plan.ExternalAccessRecipients {
		recipients[i] = recipient.ValueString()
	}

	updateRequest := &client.UpdateApplicationRequest{
		ExternalAccessEnabled:    IsKnown(plan.ExternalAccessEnabled) && plan.ExternalAccessEnabled.ValueBool(),
		ExternalAccessRecipients: recipients,
	}

	if state.Name.ValueString() != plan.Name.ValueString() {
		updateRequest.Name = plan.Name.ValueString()
	}

	traceAPICall("UpdateChatApplication")
	_, err := r.provider.service.UpdateApplication(ctx,
		plan.ID.ValueString(),
		updateRequest)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Application not found",
				fmt.Sprintf("Application with ID %s is not found. Removing from state.", plan.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error updating Application", err.Error())
		}
		return
	}

	application, err := waitForApplicationToBeReady(ctx, r.provider.service, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Application not ready", err.Error())
		return
	}

	traceAPICall("GetChatApplicationSource")
	applicationSource, err := r.provider.service.GetApplicationSource(ctx, application.CustomApplicationSourceID)
	if err != nil {
		resp.Diagnostics.AddError("Error getting Application Source info", err.Error())
		return
	}
	plan.VersionID = types.StringValue(applicationSource.LatestVersion.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ChatApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ChatApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteChatApplication")
	err := r.provider.service.DeleteApplication(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Application", err.Error())
			return
		}
	}
}

func (r *ChatApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
