package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/common"
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &QAApplicationResource{}
var _ resource.ResourceWithImportState = &QAApplicationResource{}

func NewQAApplicationResource() resource.Resource {
	return &QAApplicationResource{}
}

type QAApplicationResource struct {
	service client.Service
}

func (r *QAApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_qa_application"
}

func (r *QAApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Q&A Application",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Q&A Application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Q&A Application Source.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_version_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The version ID of the Q&A Application Source.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The URL of the Q&A Application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Q&A Application.",
			},
			"deployment_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The deployment ID of the Q&A Application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"external_access_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether external access is enabled for the Q&A Application.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"external_access_recipients": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "The list of external email addresses that have access to the Q&A Application.",
				ElementType:         types.StringType,
			},
			"allow_auto_stopping": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether auto stopping is allowed for the Q&A Application.",
			},
		},
	}
}

func (r *QAApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil { return }
	accessor, ok := req.ProviderData.(common.ServiceAccessor)
	if !ok {
			resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
			return
	}
	r.service = accessor.GetService()
}

func (r *QAApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.QAApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("CreateQAApplication")
	createResp, err := r.service.CreateQAApplication(ctx, &client.CreateQAApplicationRequest{
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

	common.TraceAPICall("UpdateApplication")
	_, err = r.service.UpdateApplication(ctx,
		createResp.ID,
		&client.UpdateApplicationRequest{
			Name:                     data.Name.ValueString(),
			ExternalAccessEnabled:    common.IsKnown(data.ExternalAccessEnabled) && data.ExternalAccessEnabled.ValueBool(),
			ExternalAccessRecipients: recipients,
			AllowAutoStopping:        data.AllowAutoStopping.ValueBool(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Application not found",
				fmt.Sprintf("Application with ID %s is not found. Removing from state.", createResp.ID))
			resp.State.RemoveResource(ctx)
		} else {
			errMessage := common.CheckApplicationNameAlreadyExists(err, data.Name.ValueString())
			resp.Diagnostics.AddError("Error adding details to Q&A Application", errMessage)
		}
		return
	}

	application, err := common.WaitForApplicationToBeReady(ctx, r.service, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Application not ready", err.Error())
		return
	}
	data.ID = types.StringValue(application.ID)
	data.SourceID = types.StringValue(application.CustomApplicationSourceID)
	data.SourceVersionID = types.StringValue(application.CustomApplicationSourceVersionID)
	data.ApplicationUrl = types.StringValue(application.ApplicationUrl)
	data.ExternalAccessEnabled = types.BoolValue(application.ExternalAccessEnabled)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *QAApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.QAApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	common.TraceAPICall("GetApplication")
	application, err := r.service.GetApplication(ctx, data.ID.ValueString())
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
	data.AllowAutoStopping = types.BoolValue(application.AllowAutoStopping)
	data.SourceID = types.StringValue(application.CustomApplicationSourceID)
	data.SourceVersionID = types.StringValue(application.CustomApplicationSourceVersionID)

	common.TraceAPICall("GetApplicationSourceVersion")
	applicationSourceVersion, err := r.service.GetApplicationSourceVersion(
		ctx,
		application.CustomApplicationSourceID,
		application.CustomApplicationSourceVersionID)
	if err != nil {
		resp.Diagnostics.AddError("Error getting Application Source version", err.Error())
		return
	}

	if applicationSourceVersion.RuntimeParameters != nil {
		for _, runtimeParameter := range applicationSourceVersion.RuntimeParameters {
			if runtimeParameter.FieldName == "DEPLOYMENT_ID" {
				if runtimeParameter.CurrentValue == nil {
					data.DeploymentID = types.StringNull()
				}
				if currentValue, ok := runtimeParameter.CurrentValue.(string); ok {
					data.DeploymentID = types.StringValue(currentValue)
				} else {
					resp.Diagnostics.AddError("Invalid Deployment ID", fmt.Sprintf("Deployment ID is not a string: %v", runtimeParameter.CurrentValue))
					return
				}
				break
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *QAApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.QAApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state models.QAApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	recipients := make([]string, len(plan.ExternalAccessRecipients))
	for i, recipient := range plan.ExternalAccessRecipients {
		recipients[i] = recipient.ValueString()
	}

	updateRequest := &client.UpdateApplicationRequest{
		ExternalAccessEnabled:    common.IsKnown(plan.ExternalAccessEnabled) && plan.ExternalAccessEnabled.ValueBool(),
		ExternalAccessRecipients: recipients,
		AllowAutoStopping:        plan.AllowAutoStopping.ValueBool(),
	}

	if state.Name.ValueString() != plan.Name.ValueString() {
		updateRequest.Name = plan.Name.ValueString()
	}

	common.TraceAPICall("UpdateApplication")
	_, err := r.service.UpdateApplication(ctx,
		plan.ID.ValueString(),
		updateRequest)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Application not found",
				fmt.Sprintf("Application with ID %s is not found. Removing from state.", plan.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			errMessage := common.CheckApplicationNameAlreadyExists(err, plan.Name.ValueString())
			resp.Diagnostics.AddError("Error updating Application", errMessage)
		}
		return
	}

	_, err = common.WaitForApplicationToBeReady(ctx, r.service, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Application not ready", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *QAApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.QAApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("DeleteApplication")
	err := r.service.DeleteApplication(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Application", err.Error())
			return
		}
	}

	common.TraceAPICall("DeleteApplicationSource")
	err = r.service.DeleteApplicationSource(ctx, data.SourceID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Application Source", err.Error())
			return
		}
	}
}

func (r *QAApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
