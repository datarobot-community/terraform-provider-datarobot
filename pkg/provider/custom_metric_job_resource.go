package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	jobType = "hostedCustomMetric"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.ResourceWithImportState = &CustomMetricJobResource{}
var _ resource.ResourceWithModifyPlan = &CustomMetricJobResource{}

func NewCustomMetricJobResource() resource.Resource {
	return &CustomMetricJobResource{}
}

// VectorDatabaseResource defines the resource implementation.
type CustomMetricJobResource struct {
	provider *Provider
}

func (r *CustomMetricJobResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_metric_job"
}

func (r *CustomMetricJobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Custom Job",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Custom Metric Job.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Custom Metric Job.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The description of the Custom Metric Job.",
			},
			"environment_id": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "The ID of the environment to use with the Job.",
			},
			"environment_version_id": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "The ID of the environment version to use with the Job.",
			},
			"runtime_parameter_values": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Additional parameters to be injected into a Job at runtime.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The name of the runtime parameter.",
						},
						"type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The type of the runtime parameter.",
						},
						"value": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The value of the runtime parameter (type conversion is handled internally).",
						},
					},
				},
			},
			"folder_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The path to a folder containing files to be uploaded. Each file in the folder is uploaded under path relative to a folder path.",
			},
			"folder_path_hash": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The hash of the folder path contents.",
			},
			"files": schema.DynamicAttribute{
				Optional:            true,
				MarkdownDescription: "The list of tuples, where values in each tuple are the local filesystem path and the path the file should be placed in the Job. If list is of strings, then basenames will be used for tuples.",
			},
			"files_hashes": schema.ListAttribute{
				Computed:            true,
				MarkdownDescription: "The hash of file contents for each file in files.",
				ElementType:         types.StringType,
			},
			"egress_network_policy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The egress network policy for the Job.",
				Default:             stringdefault.StaticString("public"),
				Validators:          EgressNetworkPolicyValidators(),
			},
			"resource_bundle_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A single identifier that represents a bundle of resources: Memory, CPU, GPU, etc.",
			},
			"directionality": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("higherIsBetter"),
				MarkdownDescription: "The directionality of the Custom Metric.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"higherIsBetter",
						"lowerIsBetter",
					),
				},
			},
			"type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("gauge"),
				MarkdownDescription: "The aggregation type of the custom metric.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"gauge",
						"sum",
						"average",
						"categorical",
					),
				},
			},
			"time_step": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("hour"),
				MarkdownDescription: "Custom metric time bucket size.",
			},
			"units": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("y"),
				MarkdownDescription: "The units, or the y-axis label, of the given custom metric.",
			},
			"is_model_specific": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Determines whether the metric is related to the model or deployment.",
			},
		},
	}
}

func (r *CustomMetricJobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CustomMetricJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CustomMetricJobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	customMetricJob, err := r.provider.service.CreateCustomJob(ctx, &client.CreateCustomJobRequest{
		Name:                 data.Name.ValueString(),
		Description:          StringValuePointerOptional(data.Description),
		JobType:              jobType,
		EnvironmentID:        StringValuePointerOptional(data.EnvironmentID),
		EnvironmentVersionID: StringValuePointerOptional(data.EnvironmentVersionID),
		Resources: client.CustomJobResources{
			EgressNetworkPolicy: data.EgressNetworkPolicy.ValueString(),
			ResourceBundleID:    StringValuePointerOptional(data.ResourceBundleID),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Custom Job", err.Error())
		return
	}

	localFiles, err := prepareLocalFiles(data.FolderPath, data.Files)
	if err != nil {
		return
	}

	customMetricJob, err = r.provider.service.UpdateCustomJobFiles(ctx, customMetricJob.ID, localFiles)
	if err != nil {
		resp.Diagnostics.AddError("Error adding Custom Job files", err.Error())
		return
	}

	if IsKnown(data.RuntimeParameterValues) {
		runtimeParameterValues, err := convertRuntimeParameterValues(ctx, data.RuntimeParameterValues)
		if err != nil {
			resp.Diagnostics.AddError("Error reading runtime parameter values", err.Error())
			return
		}

		if _, err = r.provider.service.UpdateCustomJob(ctx, customMetricJob.ID, &client.UpdateCustomJobRequest{
			Name:                   customMetricJob.Name,
			RuntimeParameterValues: runtimeParameterValues,
		}); err != nil {
			resp.Diagnostics.AddError("Error adding runtime parameter values to Custom Job", err.Error())
			return
		}
	}
	data.ID = types.StringValue(customMetricJob.ID)
	data.EnvironmentID = types.StringValue(customMetricJob.EnvironmentID)
	data.EnvironmentVersionID = types.StringValue(customMetricJob.EnvironmentVersionID)

	traceAPICall("CreateHostedCustomMetricTemplate")
	_, err = r.provider.service.CreateHostedCustomMetricTemplate(ctx, customMetricJob.ID, &client.HostedCustomMetricTemplateRequest{
		Directionality:  data.Directionality.ValueString(),
		Type:            data.Type.ValueString(),
		Units:           data.Units.ValueString(),
		TimeStep:        data.TimeStep.ValueString(),
		IsModelSpecific: data.IsModelSpecific.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Hosted Custom Metric Template", err.Error())
		return
	}

	var diags diag.Diagnostics
	data.RuntimeParameterValues, diags = formatRuntimeParameterValues(
		ctx,
		customMetricJob.RuntimeParameters,
		data.RuntimeParameterValues)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomMetricJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CustomMetricJobResourceModel

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	id := data.ID.ValueString()

	traceAPICall("GetCustomJob")
	CustomMetricJob, err := r.provider.service.GetCustomJob(ctx, id)
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Custom Job not found",
				fmt.Sprintf("Custom Job with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Custom Job with ID %s", id),
				err.Error())
		}
		return
	}

	data.Name = types.StringValue(CustomMetricJob.Name)
	if CustomMetricJob.Description != "" {
		data.Description = types.StringValue(CustomMetricJob.Description)
	}
	data.EnvironmentID = types.StringValue(CustomMetricJob.EnvironmentID)
	data.EnvironmentVersionID = types.StringValue(CustomMetricJob.EnvironmentVersionID)
	data.RuntimeParameterValues, diags = formatRuntimeParameterValues(
		ctx,
		CustomMetricJob.RuntimeParameters,
		data.RuntimeParameterValues)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	data.EgressNetworkPolicy = types.StringValue(CustomMetricJob.Resources.EgressNetworkPolicy)

	traceAPICall("GetHostedCustomMetricTemplate")
	hostedCustomMetricTemplate, err := r.provider.service.GetHostedCustomMetricTemplate(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting Hosted Custom Metric Template", err.Error())
		return
	}
	data.Directionality = types.StringValue(hostedCustomMetricTemplate.Directionality)
	data.Type = types.StringValue(hostedCustomMetricTemplate.Type)
	data.Units = types.StringValue(hostedCustomMetricTemplate.Units)
	data.TimeStep = types.StringValue(hostedCustomMetricTemplate.TimeStep)
	data.IsModelSpecific = types.BoolValue(hostedCustomMetricTemplate.IsModelSpecific)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomMetricJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CustomMetricJobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// first update the Custom Job files
	localFiles, err := prepareLocalFiles(data.FolderPath, data.Files)
	if err != nil {
		return
	}

	_, err = r.provider.service.UpdateCustomJobFiles(ctx, data.ID.ValueString(), localFiles)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Custom Job files", err.Error())
		return
	}

	runtimeParameterValues, err := convertRuntimeParameterValues(ctx, data.RuntimeParameterValues)
	if err != nil {
		resp.Diagnostics.AddError("Error reading runtime parameter values", err.Error())
		return
	}

	// then update the rest of the Custom Job fields
	traceAPICall("UpdateCustomJob")
	_, err = r.provider.service.UpdateCustomJob(ctx, data.ID.ValueString(), &client.UpdateCustomJobRequest{
		Name:                   data.Name.ValueString(),
		Description:            StringValuePointerOptional(data.Description),
		EnvironmentID:          StringValuePointerOptional(data.EnvironmentID),
		EnvironmentVersionID:   StringValuePointerOptional(data.EnvironmentVersionID),
		RuntimeParameterValues: runtimeParameterValues,
		Resources: &client.CustomJobResources{
			EgressNetworkPolicy: data.EgressNetworkPolicy.ValueString(),
			ResourceBundleID:    StringValuePointerOptional(data.ResourceBundleID),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating Custom Job", err.Error())
		return
	}

	customMetrics, err := r.provider.service.ListCustomJobMetrics(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing custom metrics", err.Error())
		return
	}

	if len(customMetrics) < 1 {
		traceAPICall("UpdateHostedCustomMetricTemplate")
		_, err = r.provider.service.UpdateHostedCustomMetricTemplate(ctx, data.ID.ValueString(), &client.HostedCustomMetricTemplateRequest{
			Directionality:  data.Directionality.ValueString(),
			Type:            data.Type.ValueString(),
			Units:           data.Units.ValueString(),
			TimeStep:        data.TimeStep.ValueString(),
			IsModelSpecific: data.IsModelSpecific.ValueBool(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error updating Hosted Custom Metric Template", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomMetricJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CustomMetricJobResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteCustomJob")
	err := r.provider.service.DeleteCustomJob(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Custom Job", err.Error())
			return
		}
	}
}

func (r *CustomMetricJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r CustomMetricJobResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Resource is being destroyed
		return
	}

	var plan CustomMetricJobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// compute file content hashes
	filesHashes, err := computeFilesHashes(ctx, plan.Files)
	if err != nil {
		resp.Diagnostics.AddError("Error calculating files hashes", err.Error())
		return
	}
	plan.FilesHashes = filesHashes

	folderPathHash, err := computeFolderHash(plan.FolderPath)
	if err != nil {
		resp.Diagnostics.AddError("Error calculating folder path hash", err.Error())
		return
	}
	plan.FolderPathHash = folderPathHash

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)

	if req.State.Raw.IsNull() {
		// resource is being created
		return
	}

	var state CustomMetricJobResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !IsKnown(plan.EnvironmentID) {
		if plan.EnvironmentVersionID == state.EnvironmentVersionID {
			// use state environment id if environment version id is not changed
			plan.EnvironmentID = state.EnvironmentID
		}
	}

	if !IsKnown(plan.EnvironmentVersionID) {
		if plan.EnvironmentID == state.EnvironmentID {
			// use state environment version id if environment id is not changed
			plan.EnvironmentVersionID = state.EnvironmentVersionID
		}
	}

	if !IsKnown(plan.RuntimeParameterValues) {
		// use empty list if runtime parameter values are unknown
		plan.RuntimeParameterValues, _ = listValueFromRuntimParameters(ctx, []RuntimeParameterValue{})
	}

	customMetrics, err := r.provider.service.ListCustomJobMetrics(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing custom metrics", err.Error())
		return
	}
	if len(customMetrics) > 0 {
		if state.Directionality != plan.Directionality {
			addCannotChangeCustomJobAttributeError(resp, "directionality")
			return
		}
		if state.Units != plan.Units {
			addCannotChangeCustomJobAttributeError(resp, "units")
			return
		}
		if state.TimeStep != plan.TimeStep {
			addCannotChangeCustomJobAttributeError(resp, "time_step")
			return
		}
		if state.Type != plan.Type {
			addCannotChangeCustomJobAttributeError(resp, "type")
			return
		}
		if state.IsModelSpecific != plan.IsModelSpecific {
			addCannotChangeCustomJobAttributeError(resp, "is_model_specific")
			return
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r CustomMetricJobResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("environment_id"),
			path.MatchRoot("environment_version_id"),
		),
	}
}

func addCannotChangeCustomJobAttributeError(
	resp *resource.ModifyPlanResponse,
	attribute string,
) {
	resp.Diagnostics.AddError(
		"Custom Metric Job Update Error",
		fmt.Sprintf("%s cannot be changed if the custom job has an associated deployment.", attribute))
}
