// filepath: /Users/volodymyr.onofriichuk/Projects/terraform-provider-datarobot/pkg/provider/custom_job_schedule_resource.go
package provider

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.ResourceWithImportState = &CustomJobScheduleResource{}

func NewCustomJobScheduleResource() resource.Resource {
	return &CustomJobScheduleResource{}
}

// CustomJobScheduleResource defines the resource implementation.
type CustomJobScheduleResource struct {
	provider *Provider
}

func (r *CustomJobScheduleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_job_schedule"
}

func (r *CustomJobScheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Custom Job Schedule",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique ID of the custom job schedule.",
			},
			"custom_job_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the custom job to which this schedule belongs.",
			},
			"schedule": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The schedule configuration for the custom job.",
				Attributes: map[string]schema.Attribute{
					"minute": schema.StringAttribute{
						Optional:    true,
						Description: "The minute field of the schedule (e.g., '0', '*', or '*/15').",
					},
					"hour": schema.StringAttribute{
						Optional:    true,
						Description: "The hour field of the schedule (e.g., '0', '*', or '9-17').",
					},
					"month": schema.StringAttribute{
						Optional:    true,
						Description: "The month field of the schedule (e.g., '1', '*', or '1,6,12').",
					},
					"day_of_month": schema.StringAttribute{
						Optional:    true,
						Description: "The day of the month field of the schedule (e.g., '1', '*', or '1-15').",
					},
					"day_of_week": schema.StringAttribute{
						Optional:    true,
						Description: "The day of the week field of the schedule (e.g., '0', '*', or '1-5').",
					},
				},
			},
			"parameter_overrides": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Overrides for runtime parameters when the job is executed.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"field_name": schema.StringAttribute{
							Required:    true,
							Description: "The name of the parameter to override.",
						},
						"type": schema.StringAttribute{
							Optional:    true,
							Description: "The type of the parameter (e.g., 'string', 'int').",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The value of the parameter to override.",
						},
					},
				},
			},
			"scheduled_job_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the scheduled job created by this schedule.",
			},
		},
	}
}

func (r *CustomJobScheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var ok bool
	if r.provider, ok = req.ProviderData.(*Provider); !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected %T, got: %T. Please report this issue to the provider developers.", Provider{}, req.ProviderData),
		)
	}
}

func (r *CustomJobScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CustomJobScheduleModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map the Terraform model to the API request
	apiRequest := client.CreateaCustomJobScheduleRequest{
		Schedule: client.Schedule{
			Minute:     convertStringSlice(plan.Schedule.Minute),
			Hour:       convertStringSlice(plan.Schedule.Hour),
			Month:      convertStringSlice(plan.Schedule.Month),
			DayOfMonth: convertStringSlice(plan.Schedule.DayOfMonth),
			DayOfWeek:  convertStringSlice(plan.Schedule.DayOfWeek),
		},
	}

	// Add parameter overrides if provided
	if len(plan.ParameterOverrides) > 0 {
		var overrides []client.RuntimeParameterValueRequest
		for _, override := range plan.ParameterOverrides {
			overrides = append(overrides, client.RuntimeParameterValueRequest{
				FieldName: override.FieldName.ValueString(),
				Type:      override.Type.ValueString(),
				Value:     stringToPointerAny(override.Value.ValueString()),
			})
		}
		apiRequest.ParameterOverrides = &overrides
	}
	// Call the API to create the schedule
	traceAPICall("CreateCustomJobSchedule")
	apiResponse, err := r.provider.service.CreateCustomJobSchedule(ctx, plan.CustomJobID.ValueString(), apiRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating custom job schedule",
			"Could not create custom job schedule: "+err.Error(),
		)
		return
	}

	// Populate the state with the API response
	plan.ID = types.StringValue(apiResponse.ID)
	plan.ScheduledJobID = types.StringValue(apiResponse.ScheduledJobID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *CustomJobScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CustomJobScheduleModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch all schedules for the custom job
	traceAPICall("ListCustomJobSchedules")
	schedules, err := r.provider.service.ListCustomJobSchedules(ctx, state.CustomJobID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom job schedules",
			"Could not list custom job schedules: "+err.Error(),
		)
		return
	}

	// Find the schedule with the matching ID
	var foundSchedule *client.CustomJobScheduleResponse
	for _, schedule := range schedules {
		if schedule.ID == state.ID.ValueString() {
			foundSchedule = &schedule
			break
		}
	}

	if foundSchedule == nil {
		resp.Diagnostics.AddError(
			"Custom job schedule not found",
			"Could not find custom job schedule with ID: "+state.ID.ValueString(),
		)
		return
	}

	// Update the state with the found schedule
	state.ScheduledJobID = types.StringValue(foundSchedule.ScheduledJobID)
	state.Schedule = &Schedule{
		Minute:     convertToStringTypeSlice(foundSchedule.Schedule.Minute),
		Hour:       convertToStringTypeSlice(foundSchedule.Schedule.Hour),
		Month:      convertToStringTypeSlice(foundSchedule.Schedule.Month),
		DayOfMonth: convertToStringTypeSlice(foundSchedule.Schedule.DayOfMonth),
		DayOfWeek:  convertToStringTypeSlice(foundSchedule.Schedule.DayOfWeek),
	}
	if foundSchedule.ParameterOverrides != nil {
		var overrides []RuntimeParameterValueRequestModel
		for _, override := range *foundSchedule.ParameterOverrides {
			overrides = append(overrides, RuntimeParameterValueRequestModel{
				FieldName: types.StringValue(override.FieldName),
				Type:      types.StringValue(override.Type),
				Value:     types.StringValue(fmt.Sprintf("%v", override.Value)),
			})
		}
		state.ParameterOverrides = overrides
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *CustomJobScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CustomJobScheduleModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map the Terraform model to the API request
	apiRequest := client.CreateaCustomJobScheduleRequest{
		Schedule: client.Schedule{
			Minute:     convertStringSlice(plan.Schedule.Minute),
			Hour:       convertStringSlice(plan.Schedule.Hour),
			Month:      convertStringSlice(plan.Schedule.Month),
			DayOfMonth: convertStringSlice(plan.Schedule.DayOfMonth),
			DayOfWeek:  convertStringSlice(plan.Schedule.DayOfWeek),
		},
	}

	// Add parameter overrides if provided
	if len(plan.ParameterOverrides) > 0 {
		var overrides []client.RuntimeParameterValueRequest
		for _, override := range plan.ParameterOverrides {

			overrides = append(overrides, client.RuntimeParameterValueRequest{
				FieldName: override.FieldName.ValueString(),
				Type:      override.Type.ValueString(),
				Value:     stringToPointerAny(override.Value.ValueString()),
			})
		}
		apiRequest.ParameterOverrides = &overrides
	}

	// Call the API to update the schedule
	traceAPICall("UpdateCustomJobSchedule")
	apiResponse, err := r.provider.service.UpdateCustomJobSchedule(ctx, plan.CustomJobID.ValueString(), plan.ID.ValueString(), apiRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating custom job schedule",
			"Could not update custom job schedule: "+err.Error(),
		)
		return
	}

	// Update the state with the API response
	plan.ScheduledJobID = types.StringValue(apiResponse.ScheduledJobID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *CustomJobScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CustomJobScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteCustomJobSchedule")
	err := r.provider.service.DeleteCustomJobSchedule(ctx, state.CustomJobID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting custom job schedule",
			"Could not delete custom job schedule: "+err.Error(),
		)
		return
	}
}
func (r *CustomJobScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func convertStringSlice(input []types.String) []any {
	var result []any
	for _, v := range input {
		result = append(result, v.ValueString())
	}
	return result
}

func convertToStringTypeSlice(input any) []types.String {
	var result []types.String
	if inputSlice, ok := input.([]any); ok {
		for _, v := range inputSlice {
			result = append(result, types.StringValue(v.(string)))
		}
	}
	return result
}

func stringToPointerAny(value string) *any {
	var v any = value
	return &v
}
