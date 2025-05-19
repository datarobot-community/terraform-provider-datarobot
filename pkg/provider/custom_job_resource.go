package provider

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	retrainingJobType   = "retraining"
	defaultJobType      = "default"
	notificationJobType = "notification"

	deploymentParamName         = "DEPLOYMENT"
	retrainingPolicyIDParamName = "RETRAINING_POLICY_ID"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.ResourceWithImportState = &CustomJobResource{}
var _ resource.ResourceWithModifyPlan = &CustomJobResource{}

func NewCustomJobResource() resource.Resource {
	return &CustomJobResource{}
}

// VectorDatabaseResource defines the resource implementation.
type CustomJobResource struct {
	provider *Provider
}

func (r *CustomJobResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_job"
}

func (r *CustomJobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Custom Job",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Custom Job.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Custom Job.",
			},
			"job_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
				MarkdownDescription: "The type of the Custom Job.",
				Validators:          CustomJobTypeValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The description of the Custom Job.",
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
			"files": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "List of files to upload, each with a source (local path) and destination (path in job).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"source": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Local filesystem path.",
						},
						"destination": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Path in the job.",
						},
					},
				},
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
			"schedule": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The schedule configuration for the custom job.",
				Attributes: map[string]schema.Attribute{
					"minute": schema.ListAttribute{
						Required:    true,
						Description: "Minutes of the day when the job will run.",
						ElementType: types.StringType,
					},
					"hour": schema.ListAttribute{
						Required:    true,
						Description: "Hours of the day when the job will run.",
						ElementType: types.StringType,
					},
					"month": schema.ListAttribute{
						Required:    true,
						Description: "Months of the year when the job will run.",
						ElementType: types.StringType,
					},
					"day_of_month": schema.ListAttribute{
						Required:    true,
						Description: "Days of the month when the job will run.",
						ElementType: types.StringType,
					},
					"day_of_week": schema.ListAttribute{
						Required:    true,
						Description: "Days of the week when the job will run.",
						ElementType: types.StringType,
					},
				},
			},
			"schedule_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The ID of the schedule associated with the custom job.",
			},
		},
	}
}

func (r *CustomJobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CustomJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CustomJobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	customJob, err := r.provider.service.CreateCustomJob(ctx, &client.CreateCustomJobRequest{
		Name:                 data.Name.ValueString(),
		Description:          StringValuePointerOptional(data.Description),
		JobType:              data.JobType.ValueString(),
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
		resp.Diagnostics.AddError("Error preparing local files", err.Error())
		return
	}

	customJob, err = r.provider.service.UpdateCustomJobFiles(ctx, customJob.ID, localFiles)
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

		if _, err = r.provider.service.UpdateCustomJob(ctx, customJob.ID, &client.UpdateCustomJobRequest{
			Name:                   customJob.Name,
			RuntimeParameterValues: runtimeParameterValues,
		}); err != nil {
			resp.Diagnostics.AddError("Error adding runtime parameter values to Custom Job", err.Error())
			return
		}
	}

	if data.Schedule != nil {
		var schedule client.Schedule
		if schedule, err = convertSchedule(*data.Schedule); err != nil {
			resp.Diagnostics.AddError("Error converting schedule", err.Error())
			return
		}

		scheduleRequest := client.CreateaCustomJobScheduleRequest{
			Schedule: schedule,
		}

		scheduleResponse, err := r.provider.service.CreateCustomJobSchedule(ctx, customJob.ID, scheduleRequest)
		if err != nil {
			resp.Diagnostics.AddError("Error creating Custom Job Schedule", err.Error())
			return
		}

		data.ScheduleID = types.StringValue(scheduleResponse.ID)

	} else {
		data.ScheduleID = types.StringNull()
	}

	data.ID = types.StringValue(customJob.ID)
	data.EnvironmentID = types.StringValue(customJob.EnvironmentID)
	data.EnvironmentVersionID = types.StringValue(customJob.EnvironmentVersionID)
	var diags diag.Diagnostics
	data.RuntimeParameterValues, diags = checkAndFormatRuntimeParameterValues(
		ctx,
		customJob.RuntimeParameters,
		data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CustomJobResourceModel

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
	customJob, err := r.provider.service.GetCustomJob(ctx, id)
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

	data.Name = types.StringValue(customJob.Name)
	if customJob.Description != "" {
		data.Description = types.StringValue(customJob.Description)
	}
	// Fetch the schedule
	schedules, err := r.provider.service.ListCustomJobSchedules(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Custom Job Schedules", err.Error())
		return
	}
	if len(schedules) > 0 {
		schedule := schedules[0] // Assuming one schedule per job, as it's not allowed to have multiple schedules
		convertedSchedule, err := convertScheduleFromAPI(schedule.Schedule)
		if err != nil {
			resp.Diagnostics.AddError("Error converting schedule", err.Error())
			return
		}
		data.Schedule = &convertedSchedule

		if data.ScheduleID.IsNull() || data.ScheduleID.ValueString() != schedule.ID {
			data.ScheduleID = types.StringValue(schedule.ID)
		}
	} else {
		data.ScheduleID = types.StringNull()
	}
	data.EnvironmentID = types.StringValue(customJob.EnvironmentID)
	data.EnvironmentVersionID = types.StringValue(customJob.EnvironmentVersionID)
	data.JobType = types.StringValue(customJob.JobType)
	data.RuntimeParameterValues, diags = checkAndFormatRuntimeParameterValues(
		ctx,
		customJob.RuntimeParameters,
		data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	data.EgressNetworkPolicy = types.StringValue(customJob.Resources.EgressNetworkPolicy)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state CustomJobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !reflect.DeepEqual(plan.FilesHashes, state.FilesHashes) ||
		plan.FolderPathHash != state.FolderPathHash {
		localFiles, err := prepareLocalFiles(plan.FolderPath, plan.Files)
		if err != nil {
			return
		}

		_, err = r.provider.service.UpdateCustomJobFiles(ctx, plan.ID.ValueString(), localFiles)
		if err != nil {
			resp.Diagnostics.AddError("Error updating Custom Job files", err.Error())
			return
		}
	}

	runtimeParameterValues, err := convertRuntimeParameterValues(ctx, plan.RuntimeParameterValues)
	if err != nil {
		resp.Diagnostics.AddError("Error reading runtime parameter values", err.Error())
		return
	}

	// then update the rest of the Custom Job fields
	traceAPICall("UpdateCustomJob")
	_, err = r.provider.service.UpdateCustomJob(ctx, plan.ID.ValueString(), &client.UpdateCustomJobRequest{
		Name:                   plan.Name.ValueString(),
		Description:            StringValuePointerOptional(plan.Description),
		EnvironmentID:          StringValuePointerOptional(plan.EnvironmentID),
		EnvironmentVersionID:   StringValuePointerOptional(plan.EnvironmentVersionID),
		RuntimeParameterValues: runtimeParameterValues,
		Resources: &client.CustomJobResources{
			EgressNetworkPolicy: plan.EgressNetworkPolicy.ValueString(),
			ResourceBundleID:    StringValuePointerOptional(plan.ResourceBundleID),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating Custom Job", err.Error())
		return
	}

	// Update or create the schedule if provided
	if plan.Schedule != nil {
		scheduleRequest := client.CreateaCustomJobScheduleRequest{
			Schedule: client.Schedule{
				Minute:     convertStringSlice(plan.Schedule.Minute),
				Hour:       convertStringSlice(plan.Schedule.Hour),
				Month:      convertStringSlice(plan.Schedule.Month),
				DayOfMonth: convertStringSlice(plan.Schedule.DayOfMonth),
				DayOfWeek:  convertStringSlice(plan.Schedule.DayOfWeek),
			},
		}

		if state.ScheduleID.IsNull() || state.ScheduleID.ValueString() == "" {
			// Create a new schedule if ScheduleID does not exist
			scheduleResponse, err := r.provider.service.CreateCustomJobSchedule(ctx, plan.ID.ValueString(), scheduleRequest)
			if err != nil {
				resp.Diagnostics.AddError("Error creating Custom Job Schedule", err.Error())
				return
			}
			plan.ScheduleID = types.StringValue(scheduleResponse.ID)
		} else {
			// Update the existing schedule
			_, err := r.provider.service.UpdateCustomJobSchedule(ctx, plan.ID.ValueString(), state.ScheduleID.ValueString(), scheduleRequest)
			if err != nil {
				resp.Diagnostics.AddError("Error updating Custom Job Schedule", err.Error())
				return
			}
		}
	} else {
		plan.ScheduleID = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *CustomJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CustomJobResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Delete the schedule
	if data.Schedule != nil {
		err := r.provider.service.DeleteCustomJobSchedule(ctx, data.ID.ValueString(), data.ScheduleID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error deleting Custom Job Schedule", err.Error())
			return
		}
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

func (r *CustomJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r CustomJobResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Resource is being destroyed
		return
	}

	var plan CustomJobResourceModel

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

	var state CustomJobResourceModel

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

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r CustomJobResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data CustomJobResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.JobType.ValueString() == retrainingJobType {
		verifyMetadataForRetrainingJob(data, resp)
	}
}

func verifyMetadataForRetrainingJob(data CustomJobResourceModel, resp *resource.ValidateConfigResponse) {
	localFiles, err := prepareLocalFiles(data.FolderPath, data.Files)
	if err != nil {
		return
	}

	for _, localFile := range localFiles {
		if localFile.Name == "metadata.yaml" {
			content := string(localFile.Content)
			if !strings.Contains(content, fmt.Sprintf("fieldName: %s", deploymentParamName)) ||
				!strings.Contains(content, fmt.Sprintf("fieldName: %s", retrainingPolicyIDParamName)) {
				resp.Diagnostics.AddError(
					"Invalid files for Retraining Job",
					fmt.Sprintf("Retraining Job requires a metadata.yaml that contains %s and %s runtimeParameterDefinitions", deploymentParamName, retrainingPolicyIDParamName))
				return

			}
			return
		}
	}
}

func checkAndFormatRuntimeParameterValues(ctx context.Context, customJobRuntimeParameters []client.RuntimeParameter, data CustomJobResourceModel) (basetypes.ListValue, diag.Diagnostics) {
	if data.JobType.ValueString() == retrainingJobType {
		return formatRuntimeParameterValuesForRetrainingJob(
			ctx,
			customJobRuntimeParameters,
			data.RuntimeParameterValues)
	}

	return formatRuntimeParameterValues(
		ctx,
		customJobRuntimeParameters,
		data.RuntimeParameterValues)
}

func (r CustomJobResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("environment_id"),
			path.MatchRoot("environment_version_id"),
		),
	}
}

func CustomJobTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			defaultJobType,
			retrainingJobType,
			notificationJobType,
		),
	}
}

func convertStringSlice(input []types.String) []any {
	var result []any
	for _, v := range input {
		result = append(result, v)
	}
	return result
}
