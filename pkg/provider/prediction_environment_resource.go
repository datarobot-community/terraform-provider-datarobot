package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	batchJobPriorityLow    int = 0
	batchJobPriorityMedium int = 1
	batchJobPriorityHigh   int = 2

	managedBySelfManaged     string = "selfManaged"
	managedByManagementAgent string = "managementAgent"
	managedByDatarobot       string = "datarobot"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PredictionEnvironmentResource{}
var _ resource.ResourceWithImportState = &PredictionEnvironmentResource{}

func NewPredictionEnvironmentResource() resource.Resource {
	return &PredictionEnvironmentResource{}
}

// VectorDatabaseResource defines the resource implementation.
type PredictionEnvironmentResource struct {
	provider *Provider
}

func (r *PredictionEnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prediction_environment"
}

func (r *PredictionEnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "prediction environment",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Prediction Environment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Prediction Environment.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the Prediction Environment.",
				Optional:            true,
			},
			"platform": schema.StringAttribute{
				MarkdownDescription: "The platform for the Prediction Environment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"batch_jobs_max_concurrent": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "The maximum number of concurrent batch prediction jobs.",
			},
			"batch_jobs_priority": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf("low", "medium", "high"),
				},
				MarkdownDescription: "The importance of batch jobs.",
			},
			"supported_model_formats": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "The list of supported model formats.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"managed_by": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(managedBySelfManaged),
				MarkdownDescription: "Determines if the prediction environment should be managed by the management agent, datarobot, or self-managed. Self-managed by default.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"credential_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the credential associated with the data connection. Only applicable for external prediction environments managed by DataRobot.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"datastore_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the data store connection configuration. Only applicable for external prediction environments managed by DataRobot.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *PredictionEnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PredictionEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PredictionEnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreatePredictionEnvironment")
	createResp, err := r.provider.service.CreatePredictionEnvironment(ctx, buildPredictionEnvironmentRequest(data))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Prediction Environment", err.Error())
		return
	}
	data.ID = types.StringValue(createResp.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *PredictionEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PredictionEnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetPredictionEnvironment")
	predictionEnvironment, err := r.provider.service.GetPredictionEnvironment(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Prediction Environment not found",
				fmt.Sprintf("Prediction Environment with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Prediction Environment with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}

	data.Name = types.StringValue(predictionEnvironment.Name)
	data.Platform = types.StringValue(predictionEnvironment.Platform)
	if predictionEnvironment.Description != "" {
		data.Description = types.StringValue(predictionEnvironment.Description)
	}
	data.ManagedBy = types.StringValue(predictionEnvironment.ManagedBy)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PredictionEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PredictionEnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("UpdatePredictionEnvironment")
	_, err := r.provider.service.UpdatePredictionEnvironment(ctx, data.ID.ValueString(), buildPredictionEnvironmentRequest(data))
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Prediction Environment not found",
				fmt.Sprintf("Prediction Environment with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error updating Prediction Environment", err.Error())
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *PredictionEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PredictionEnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeletePredictionEnvironment")
	err := r.provider.service.DeletePredictionEnvironment(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Prediction Environment", err.Error())
			return
		}
	}
}

func (r *PredictionEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildPredictionEnvironmentRequest(data PredictionEnvironmentResourceModel) *client.PredictionEnvironmentRequest {
	request := &client.PredictionEnvironmentRequest{
		Name:                       data.Name.ValueString(),
		Description:                data.Description.ValueString(),
		Platform:                   data.Platform.ValueString(),
		ManagedBy:                  data.ManagedBy.ValueString(),
		IsManagedByManagementAgent: isManagedByManagementAgent(data.ManagedBy.ValueString()),
	}

	if len(data.SupportedModelFormats) > 0 {
		supportedModelFormats := make([]string, len(data.SupportedModelFormats))
		for i, v := range data.SupportedModelFormats {
			supportedModelFormats[i] = v.ValueString()
		}
		request.SupportedModelFormats = supportedModelFormats
	}
	if IsKnown(data.BatchJobsPriority) {
		request.Priority = priorityStringToEnum(data.BatchJobsPriority.ValueString())
	}
	if IsKnown(data.BatchJobsMaxConcurrent) {
		batchJobsMaxConcurrent := data.BatchJobsMaxConcurrent.ValueInt64()
		request.MaxConcurrentBatchPredictionsJob = &batchJobsMaxConcurrent
	}
	if IsKnown(data.CredentialID) {
		request.CredentialID = data.CredentialID.ValueStringPointer()
	}
	if IsKnown(data.DatastoreID) {
		request.DatastoreID = data.DatastoreID.ValueStringPointer()
	}

	return request
}

func priorityStringToEnum(priority string) *int {
	var priorityValue int
	switch priority {
	case "high":
		priorityValue = batchJobPriorityHigh
	case "medium":
		priorityValue = batchJobPriorityMedium
	default:
		priorityValue = batchJobPriorityLow
	}
	return &priorityValue
}

func isManagedByManagementAgent(managedBy string) bool {
	return managedBy == managedByManagementAgent || managedBy == managedByDatarobot
}
