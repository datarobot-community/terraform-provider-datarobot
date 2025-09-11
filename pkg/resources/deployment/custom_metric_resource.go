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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CustomMetricResource{}
var _ resource.ResourceWithImportState = &CustomMetricResource{}

func NewCustomMetricResource() resource.Resource {
	return &CustomMetricResource{}
}

// VectorDatabaseResource defines the resource implementation.
type CustomMetricResource struct {
	service client.Service
}

func (r *CustomMetricResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_metric"
}

func (r *CustomMetricResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Custom Metric",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the Custom Metric.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployment_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the Deployment for the Custom Metric.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the Custom Metric.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Description of the Custom Metric.",
			},
			"units": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The units, or the y-axis label, of the given Custom Metric.",
			},
			"is_model_specific": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Determines whether the metric is related to the model or deployment.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"is_geospatial": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Determines whether the metric is geospatial.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Aggregation type of the Custom Metric.",
				Validators:          common.CustomMetricAggregationTypeValidators(),
			},
			"directionality": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Directionality of the Custom Metric",
				Validators:          common.DirectionalityValidators(),
			},
			"baseline_value": schema.Float64Attribute{
				Optional:            true,
				MarkdownDescription: "The baseline value used to add “reference dots” to the values over time chart.",
			},
			"timestamp": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "A Custom Metric timestamp column source when reading values from columnar dataset.",
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
				Description: "A Custom Metric value source when reading values from columnar dataset.",
				Attributes: map[string]schema.Attribute{
					"column_name": schema.StringAttribute{
						Optional:    true,
						Description: "Column name.",
					},
				},
			},
			"sample_count": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "A Custom Metric sample source when reading values from columnar dataset.",
				Attributes: map[string]schema.Attribute{
					"column_name": schema.StringAttribute{
						Optional:    true,
						Description: "Column name.",
					},
				},
			},
			"batch": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "A Custom Metric batch ID source when reading values from columnar dataset.",
				Attributes: map[string]schema.Attribute{
					"column_name": schema.StringAttribute{
						Optional:    true,
						Description: "Column name.",
					},
				},
			},
		},
	}
}

func (r *CustomMetricResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil { return }
	accessor, ok := req.ProviderData.(common.ServiceAccessor)
	if !ok {
			resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
			return
	}
	r.service = accessor.GetService()
}

func (r *CustomMetricResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.CustomMetricResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := &client.CreateCustomMetricRequest{
		Name:            data.Name.ValueString(),
		Description:     data.Description.ValueString(),
		Units:           data.Units.ValueString(),
		IsModelSpecific: data.IsModelSpecific.ValueBool(),
		IsGeospatial:    data.IsGeospatial.ValueBool(),
		Type:            data.Type.ValueString(),
		Directionality:  data.Directionality.ValueString(),
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

	customMetric, err := r.service.CreateCustomMetric(ctx, data.DeploymentID.ValueString(), request)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Custom Metric", err.Error())
		return
	}
	data.ID = types.StringValue(customMetric.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomMetricResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.CustomMetricResourceModel

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
	data.Description = types.StringValue(customMetric.Description)
	data.Units = types.StringValue(customMetric.Units)
	data.Directionality = types.StringValue(customMetric.Directionality)
	data.Type = types.StringValue(customMetric.Type)
	data.IsModelSpecific = types.BoolValue(customMetric.IsModelSpecific)
	data.IsGeospatial = types.BoolValue(customMetric.IsGeospatial)
	if customMetric.BaselineValues != nil && len(*customMetric.BaselineValues) > 0 {
		baselineValues := *customMetric.BaselineValues
		data.BaselineValue = types.Float64Value(baselineValues[0].Value)
	}
	if customMetric.Batch != nil {
		data.Batch = &models.ColumnNameValue{
			ColumnName: types.StringValue(customMetric.Batch.ColumnName),
		}
	}
	if customMetric.Value != nil {
		data.Value = &models.ColumnNameValue{
			ColumnName: types.StringValue(customMetric.Value.ColumnName),
		}
	}
	if customMetric.SampleCount != nil {
		data.SampleCount = &models.ColumnNameValue{
			ColumnName: types.StringValue(customMetric.SampleCount.ColumnName),
		}
	}
	if customMetric.Timestamp != nil {
		data.Timestamp = &models.MetricTimestampSpoofing{
			ColumnName: types.StringPointerValue(customMetric.Timestamp.ColumnName),
			TimeFormat: types.StringPointerValue(customMetric.Timestamp.TimeFormat),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomMetricResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data models.CustomMetricResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := &client.UpdateCustomMetricRequest{
		Name:           common.StringValuePointerOptional(data.Name),
		Description:    common.StringValuePointerOptional(data.Description),
		Units:          common.StringValuePointerOptional(data.Units),
		Directionality: common.StringValuePointerOptional(data.Directionality),
		Type:           common.StringValuePointerOptional(data.Type),
	}

	request.BaselineValues = &[]client.MetricBaselineValue{}
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
		request.Batch = &client.ColumnNameValue{
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

func (r *CustomMetricResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.CustomMetricResourceModel

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

func (r *CustomMetricResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
