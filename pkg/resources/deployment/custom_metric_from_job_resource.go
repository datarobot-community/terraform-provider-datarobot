package provider

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CustomMetricFromJobResource{}
var _ resource.ResourceWithImportState = &CustomMetricFromJobResource{}

func NewCustomMetricFromJobResource() resource.Resource {
	return &CustomMetricFromJobResource{}
}

// VectorDatabaseResource defines the resource implementation.
type CustomMetricFromJobResource struct {
	service client.Service
}

func (r *CustomMetricFromJobResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_metric_from_job"
}

func (r *CustomMetricFromJobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Custom Metric From Job",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the Custom Metric.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"custom_job_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the Custom Job.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"deployment_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the Deployment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the metric.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Description of the metric.",
			},
			"baseline_value": schema.Float64Attribute{
				Optional:            true,
				MarkdownDescription: "Baseline value for the metric.",
			},
			"timestamp": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Timestamp spoofing when reading values from file, like dataset. By default, we replicate pd.to_datetime formatting behaviour.",
				Attributes: map[string]schema.Attribute{
					"column_name": schema.StringAttribute{
						Optional:    true,
						Description: "Column name.",
					},
					"time_format": schema.StringAttribute{
						Optional:    true,
						Description: "Format.",
						Validators:  common.TimeFormatValidators(),
					},
				},
			},
			"value": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Value source when reading values from columnar dataset like a file.",
				Attributes: map[string]schema.Attribute{
					"column_name": schema.StringAttribute{
						Optional:    true,
						Description: "Column name.",
					},
				},
			},
			"batch": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Batch ID source when reading values from columnar dataset like a file.",
				Attributes: map[string]schema.Attribute{
					"column_name": schema.StringAttribute{
						Optional:    true,
						Description: "Column name.",
					},
				},
			},
			"sample_count": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Points to a weight column if users provide pre-aggregated metric values. Used with columnar datasets.",
				Attributes: map[string]schema.Attribute{
					"column_name": schema.StringAttribute{
						Required:    true,
						Description: "Column name.",
					},
				},
			},
			"schedule": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Defines at what intervals the metric job should run.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"minute": schema.ListAttribute{
						Required:    true,
						Description: "Minutes of the day when the metric job will run.",
						ElementType: types.StringType,
					},
					"hour": schema.ListAttribute{
						Required:    true,
						Description: "Hours of the day when the metric job will run.",
						ElementType: types.StringType,
					},
					"month": schema.ListAttribute{
						Required:    true,
						Description: "Months of the year when the metric job will run.",
						ElementType: types.StringType,
					},
					"day_of_month": schema.ListAttribute{
						Required:    true,
						Description: "Days of the month when the metric job will run.",
						ElementType: types.StringType,
					},
					"day_of_week": schema.ListAttribute{
						Required:    true,
						Description: "Days of the week when the metric job will run.",
						ElementType: types.StringType,
					},
				},
			},
			"parameter_overrides": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Additional parameters to be injected into the Metric Job at runtime.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
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
		},
	}
}

func (r *CustomMetricFromJobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil { return }
	accessor, ok := req.ProviderData.(common.ServiceAccessor)
	if !ok {
			resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
			return
	}
	r.service = accessor.GetService()
}

func (r *CustomMetricFromJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.CustomMetricFromJobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := &client.CreateCustomMetricFromJobRequest{
		CustomJobID: data.CustomJobID.ValueString(),
		Name:        data.Name.ValueString(),
		Description: common.StringValuePointerOptional(data.Description),
	}

	if common.IsKnown(data.BaselineValue) {
		request.BaselineValues = []client.MetricBaselineValue{
			{
				Value: data.BaselineValue.ValueFloat64(),
			},
		}
	}

	if data.Timestamp != nil {
		request.Timestamp = &client.MetricTimestampSpoofing{
			ColumnName: common.StringValuePointerOptional(data.Timestamp.ColumnName),
			TimeFormat: common.StringValuePointerOptional(data.Timestamp.TimeFormat),
		}
	}

	if data.SampleCount != nil {
		request.SampleCount = &client.ColumnNameValue{
			ColumnName: data.SampleCount.ColumnName.ValueString(),
		}
	}

	if data.Value != nil {
		request.Value = &client.ColumnNameValue{
			ColumnName: data.Value.ColumnName.ValueString(),
		}
	}

	if data.Batch != nil {
		request.Batch = &client.ColumnNameValue{
			ColumnName: data.Batch.ColumnName.ValueString(),
		}
	}

	if data.Schedule != nil {
		schedule, err := common.ConvertSchedule(*data.Schedule)
		if err != nil {
			return
		}
		request.Schedule = &schedule
	}

	if common.IsKnown(data.ParameterOverrides) {
		parameterOverrides, err := common.ConvertRuntimeParameterValuesToList(ctx, data.ParameterOverrides)
		if err != nil {
			resp.Diagnostics.AddError("Error reading parameter overrides", err.Error())
			return
		}
		request.ParameterOverrides = parameterOverrides
	}

	customMetric, err := r.service.CreateCustomMetricFromJob(ctx, data.DeploymentID.ValueString(), request)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Custom Metric", err.Error())
		return
	}
	data.ID = types.StringValue(customMetric.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomMetricFromJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.CustomMetricFromJobResourceModel

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	id := data.ID.ValueString()

	common.TraceAPICall("GetCustomMetric")
	customMetric, err := r.service.GetCustomMetric(ctx, data.DeploymentID.ValueString(), id)
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Custom Metric not found",
				fmt.Sprintf("Custom Metric with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Custom Metric with ID %s", id),
				err.Error())
		}
		return
	}

	data.Name = types.StringValue(customMetric.Name)
	if customMetric.Description != "" {
		data.Description = types.StringValue(customMetric.Description)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomMetricFromJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data models.CustomMetricFromJobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := &client.UpdateCustomMetricRequest{
		Name:        common.StringValuePointerOptional(data.Name),
		Description: common.StringValuePointerOptional(data.Description),
	}

	if common.IsKnown(data.BaselineValue) {
		request.BaselineValues = &[]client.MetricBaselineValue{
			{
				Value: data.BaselineValue.ValueFloat64(),
			},
		}
	}

	if data.Timestamp != nil {
		request.Timestamp = &client.MetricTimestampSpoofing{
			ColumnName: common.StringValuePointerOptional(data.Timestamp.ColumnName),
			TimeFormat: common.StringValuePointerOptional(data.Timestamp.TimeFormat),
		}
	}

	if data.SampleCount != nil {
		request.SampleCount = &client.ColumnNameValue{
			ColumnName: data.SampleCount.ColumnName.ValueString(),
		}
	}

	if data.Value != nil {
		request.Value = &client.ColumnNameValue{
			ColumnName: data.Value.ColumnName.ValueString(),
		}
	}
	if data.Batch != nil {
		request.Value = &client.ColumnNameValue{
			ColumnName: data.Batch.ColumnName.ValueString(),
		}
	}

	_, err := r.service.UpdateCustomMetric(ctx, data.DeploymentID.ValueString(), data.ID.ValueString(), request)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Custom Metric", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomMetricFromJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.CustomMetricFromJobResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("DeleteCustomMetric")
	err := r.service.DeleteCustomMetric(ctx, data.DeploymentID.ValueString(), data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Custom Metric", err.Error())
			return
		}
	}
}

func (r *CustomMetricFromJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *CustomMetricFromJobResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Resource is being destroyed
		return
	}

	var plan models.CustomMetricFromJobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)

	if req.State.Raw.IsNull() {
		// resource is being created
		return
	}

	if !common.IsKnown(plan.ParameterOverrides) {
		// use empty list if runtime parameter values are unknown
		plan.ParameterOverrides, _ = common.ListValueFromRuntimeParameters(ctx, []models.RuntimeParameterValue{})
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}
