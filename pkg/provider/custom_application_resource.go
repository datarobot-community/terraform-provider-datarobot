package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
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
var _ resource.Resource = &CustomApplicationResource{}
var _ resource.ResourceWithImportState = &CustomApplicationResource{}

func NewCustomApplicationResource() resource.Resource {
	return &CustomApplicationResource{}
}

type CustomApplicationResource struct {
	provider *Provider
}

func (r *CustomApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_application"
}

func (r *CustomApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Custom Application",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Custom Application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Custom Application Source.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_version_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The version ID of the Custom Application Source.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The name of the Custom Application.",
			},
			"application_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The URL of the Custom Application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"external_access_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether external access is enabled for the Custom Application.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"external_access_recipients": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "The list of external email addresses that have access to the Custom Application.",
				ElementType:         types.StringType,
			},
			"allow_auto_stopping": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether auto stopping is allowed for the Custom Application.",
			},
			"use_case_ids": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "The list of Use Case IDs to add the Custom Application to.",
				ElementType:         types.StringType,
			},
			"resources": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The resources for the Custom Application.",
				Attributes: map[string]schema.Attribute{
					"replicas": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "The replicas for the Custom Application.",
					},
					"resource_label": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The resource label for the Custom Application.",
					},
					"session_affinity": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "The session affinity for the Custom Application.",
					},
				},
			},
		},
	}
}

func (r *CustomApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CustomApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CustomApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateCustomApplication")
	application, err := r.provider.service.CreateCustomApplication(ctx, &client.CreateCustomApplicationRequest{
		ApplicationSourceVersionID: data.SourceVersionID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Custom Application", err.Error())
		return
	}

	enableExternalAccess := IsKnown(data.ExternalAccessEnabled) && data.ExternalAccessEnabled.ValueBool()

	hasResources := data.Resources != nil && (!data.Resources.Replicas.IsNull() || !data.Resources.ResourceLabel.IsNull() || !data.Resources.SessionAffinity.IsNull())

	if IsKnown(data.Name) || enableExternalAccess || !data.AllowAutoStopping.ValueBool() || hasResources {
		recipients := make([]string, len(data.ExternalAccessRecipients))
		for i, recipient := range data.ExternalAccessRecipients {
			recipients[i] = recipient.ValueString()
		}

		updateRequest := &client.UpdateApplicationRequest{
			ExternalAccessEnabled:    enableExternalAccess,
			ExternalAccessRecipients: recipients,
			AllowAutoStopping:        data.AllowAutoStopping.ValueBool(),
		}

		if IsKnown(data.Name) {
			updateRequest.Name = data.Name.ValueString()
		}

		if hasResources {
			updateRequest.Resources = &client.ApplicationResources{
				Replicas:        Int64ValuePointerOptional(data.Resources.Replicas),
				ResourceLabel:   StringValuePointerOptional(data.Resources.ResourceLabel),
				SessionAffinity: BoolValuePointerOptional(data.Resources.SessionAffinity),
			}
		}

		traceAPICall("UpdateCustomApplication")
		_, err = r.provider.service.UpdateApplication(ctx, application.ID, updateRequest)
		if err != nil {
			errMessage := checkApplicationNameAlreadyExists(err, data.Name.ValueString())
			resp.Diagnostics.AddError("Error adding details to Custom Application", errMessage)
			return
		}
	}

	application, err = waitForApplicationToBeReady(ctx, r.provider.service, application.ID)
	if err != nil {
		resp.Diagnostics.AddError("Custom Application is not ready", err.Error())
		return
	}
	data.ID = types.StringValue(application.ID)
	data.Name = types.StringValue(application.Name)
	data.SourceID = types.StringValue(application.CustomApplicationSourceID)
	data.ApplicationUrl = types.StringValue(application.ApplicationUrl)
	data.ExternalAccessEnabled = types.BoolValue(application.ExternalAccessEnabled)

	for _, useCaseID := range data.UseCaseIDs {
		traceAPICall("AddCustomApplicationToUseCase")
		if err = addEntityToUseCase(
			ctx,
			r.provider.service,
			useCaseID.ValueString(),
			"customApplication",
			application.ID,
		); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error adding Custom Application to Use Case %s", useCaseID), err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CustomApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetCustomApplication")
	application, err := r.provider.service.GetApplication(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Custom Application not found",
				fmt.Sprintf("Custom Application with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Custom Application with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.Name = types.StringValue(application.Name)
	data.ApplicationUrl = types.StringValue(application.ApplicationUrl)
	data.SourceID = types.StringValue(application.CustomApplicationSourceID)
	data.SourceVersionID = types.StringValue(application.CustomApplicationSourceVersionID)
	data.ExternalAccessEnabled = types.BoolValue(application.ExternalAccessEnabled)
	data.AllowAutoStopping = types.BoolValue(application.AllowAutoStopping)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CustomApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CustomApplicationResourceModel

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
		AllowAutoStopping:        plan.AllowAutoStopping.ValueBool(),
	}

	if state.Name.ValueString() != plan.Name.ValueString() {
		updateRequest.Name = plan.Name.ValueString()
	}

	if state.SourceVersionID.ValueString() != plan.SourceVersionID.ValueString() {
		updateRequest.CustomApplicationSourceVersionID = plan.SourceVersionID.ValueString()
	}

	// The API Currently does not support updating Resources
	// if the resources are not set in the plan, we should not send them to the API
	// hasResources := plan.Resources != nil && (!plan.Resources.Replicas.IsNull() || !plan.Resources.ResourceLabel.IsNull() || !plan.Resources.SessionAffinity.IsNull())
	// if hasResources {
	// 	updateRequest.Resources = &client.ApplicationResources{
	// 		Replicas:        Int64ValuePointerOptional(plan.Resources.Replicas),
	// 		ResourceLabel:   StringValuePointerOptional(plan.Resources.ResourceLabel),
	// 		SessionAffinity: BoolValuePointerOptional(plan.Resources.SessionAffinity),
	// 	}
	// }

	traceAPICall("UpdateCustomApplication")
	_, err := r.provider.service.UpdateApplication(ctx,
		plan.ID.ValueString(),
		updateRequest)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Custom Application not found",
				fmt.Sprintf("Custom Application with ID %s is not found. Removing from state.", plan.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			errMessage := checkApplicationNameAlreadyExists(err, plan.Name.ValueString())
			resp.Diagnostics.AddError("Error updating Custom Application", errMessage)
		}
		return
	}

	application, err := waitForApplicationToBeReady(ctx, r.provider.service, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Custom Application is not ready", err.Error())
		return
	}
	plan.SourceID = types.StringValue(application.CustomApplicationSourceID)

	if err = updateUseCasesForEntity(
		ctx,
		r.provider.service,
		"customApplication",
		application.ID,
		state.UseCaseIDs,
		plan.UseCaseIDs,
	); err != nil {
		resp.Diagnostics.AddError("Error updating Use Cases for Custom Application", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *CustomApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CustomApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteCustomApplication")
	err := r.provider.service.DeleteApplication(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Custom Application", err.Error())
			return
		}
	}
}

func (r *CustomApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
