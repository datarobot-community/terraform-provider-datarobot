package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BatchPredictionJobDefinitionResource{}
var _ resource.ResourceWithImportState = &BatchPredictionJobDefinitionResource{}

func NewBatchPredictionJobDefinitionResource() resource.Resource {
	return &BatchPredictionJobDefinitionResource{}
}

// BatchPredictionJobDefinitionResource defines the resource implementation.
type BatchPredictionJobDefinitionResource struct {
	provider *Provider
}

func (r *BatchPredictionJobDefinitionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_batch_prediction_job_definition"
}

func (r *BatchPredictionJobDefinitionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Batch Prediction Job Definition",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the batch prediction job definition.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployment_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the deployment to use for the batch prediction job.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name you want your job to be identified with. Must be unique across the organization’s existing jobs.",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether or not the job definition should be active on a scheduled basis. If True, schedule is required.",
			},
			"schedule": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Defines at what intervals the job should run.",
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
			"intake_settings": schema.SingleNestedAttribute{
				Required:    true,
				Description: "A dict configuring how data is coming from.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:    true,
						Description: "Type of data source.",
						Validators: []validator.String{
							stringvalidator.OneOf("localFile", "s3", "azure", "gcp", "dataset", "jdbc", "snowflake", "synapse", "bigquery", "datasphere"),
						},
					},
					"dataset_id": schema.StringAttribute{
						Optional:    true,
						Description: "The ID of the dataset to score for dataset type.",
					},
					"file": schema.StringAttribute{
						Optional:    true,
						Description: "String path to file of scoring data for localFile type.",
					},
					"url": schema.StringAttribute{
						Optional:    true,
						Description: "The URL to score (e.g.: s3://bucket/key) for S3 type.",
					},
					"credential_id": schema.StringAttribute{
						Optional:    true,
						Description: "The ID of the credentials for S3 or JDBC data source.",
					},
					"endpoint_url": schema.StringAttribute{
						Optional:    true,
						Description: "Any non-default endpoint URL for S3 access.",
					},
					"data_store_id": schema.StringAttribute{
						Optional:    true,
						Description: "The ID of the external data store connected to the JDBC data source.",
					},
					"query": schema.StringAttribute{
						Optional:    true,
						Description: "A self-supplied SELECT statement of the data set you wish to predict for JDBC type.",
					},
					"table": schema.StringAttribute{
						Optional:    true,
						Description: "The name of specified database table for JDBC type.",
					},
					"schema": schema.StringAttribute{
						Optional:    true,
						Description: "The name of specified database schema for JDBC type.",
					},
					"catalog": schema.StringAttribute{
						Optional:    true,
						Description: "The name of specified database catalog for JDBC type.",
					},
					"fetch_size": schema.Int64Attribute{
						Optional:    true,
						Description: "Changing the fetchSize can be used to balance throughput and memory usage for JDBC type.",
					},
				},
			},
			"output_settings": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A dict configuring how scored data is to be saved.",
				Default: objectdefault.StaticValue(types.ObjectValueMust(
					map[string]attr.Type{
						"type":                       types.StringType,
						"path":                       types.StringType,
						"url":                        types.StringType,
						"credential_id":              types.StringType,
						"endpoint_url":               types.StringType,
						"data_store_id":              types.StringType,
						"table":                      types.StringType,
						"schema":                     types.StringType,
						"catalog":                    types.StringType,
						"statement_type":             types.StringType,
						"update_columns":             types.ListType{ElemType: types.StringType},
						"where_columns":              types.ListType{ElemType: types.StringType},
						"create_table_if_not_exists": types.BoolType,
					},
					map[string]attr.Value{
						"type":                       types.StringValue("localFile"),
						"path":                       types.StringNull(),
						"url":                        types.StringNull(),
						"credential_id":              types.StringNull(),
						"endpoint_url":               types.StringNull(),
						"data_store_id":              types.StringNull(),
						"table":                      types.StringNull(),
						"schema":                     types.StringNull(),
						"catalog":                    types.StringNull(),
						"statement_type":             types.StringNull(),
						"update_columns":             types.ListNull(types.StringType),
						"where_columns":              types.ListNull(types.StringType),
						"create_table_if_not_exists": types.BoolNull(),
					},
				)),
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("localFile"),
						Description: "Type of output.",
						Validators: []validator.String{
							stringvalidator.OneOf("localFile", "s3", "azure", "gcp", "jdbc", "snowflake", "synapse", "bigquery", "datasphere"),
						},
					},
					"path": schema.StringAttribute{
						Optional:    true,
						Description: "Path to save the scored data as CSV for localFile type.",
					},
					"url": schema.StringAttribute{
						Optional:    true,
						Description: "The URL for storing the results (e.g.: s3://bucket/key) for S3 type.",
					},
					"credential_id": schema.StringAttribute{
						Optional:    true,
						Description: "The ID of the credentials for S3 or JDBC data source.",
					},
					"endpoint_url": schema.StringAttribute{
						Optional:    true,
						Description: "Any non-default endpoint URL for S3 access.",
					},
					"data_store_id": schema.StringAttribute{
						Optional:    true,
						Description: "The ID of the external data store connected to the JDBC data source.",
					},
					"table": schema.StringAttribute{
						Optional:    true,
						Description: "The name of specified database table for JDBC type.",
					},
					"schema": schema.StringAttribute{
						Optional:    true,
						Description: "The name of specified database schema for JDBC type.",
					},
					"catalog": schema.StringAttribute{
						Optional:    true,
						Description: "The name of specified database catalog for JDBC type.",
					},
					"statement_type": schema.StringAttribute{
						Optional:    true,
						Description: "The type of insertion statement to create for JDBC type.",
					},
					"update_columns": schema.ListAttribute{
						Optional:    true,
						Description: "A list of strings containing those column names to be updated for JDBC type.",
						ElementType: types.StringType,
					},
					"where_columns": schema.ListAttribute{
						Optional:    true,
						Description: "A list of strings containing those column names to be selected for JDBC type.",
						ElementType: types.StringType,
					},
					"create_table_if_not_exists": schema.BoolAttribute{
						Optional:    true,
						Description: "If no existing table is detected, attempt to create it before writing data for JDBC type.",
					},
				},
			},
			"csv_settings": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Default: objectdefault.StaticValue(types.ObjectValueMust(
					map[string]attr.Type{
						"delimiter": types.StringType,
						"quotechar": types.StringType,
						"encoding":  types.StringType,
					},
					map[string]attr.Value{
						"delimiter": types.StringValue(","),
						"quotechar": types.StringValue("\""),
						"encoding":  types.StringValue("utf-8"),
					},
				)),
				Description: "CSV intake and output settings.",
				Attributes: map[string]schema.Attribute{
					"delimiter": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString(","),
						Description: "Fields are delimited by this character. Use the string tab to denote TSV (TAB separated values). Must be either a one-character string or the string tab.",
					},
					"quotechar": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("\""),
						Description: "Fields containing the delimiter must be quoted using this character.",
					},
					"encoding": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("utf-8"),
						Description: "Encoding for the CSV files.",
					},
				},
			},
			"timeseries_settings": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Configuration for time-series scoring.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Optional:    true,
						Description: "Type of time-series prediction. Must be 'forecast' or 'historical'. Default is 'forecast'.",
						Validators: []validator.String{
							stringvalidator.OneOf("forecast", "historical"),
						},
					},
					"forecast_point": schema.StringAttribute{
						Optional:    true,
						Description: "Forecast point for the dataset, used for the forecast predictions. May be passed if timeseries_settings.type=forecast.",
					},
					"predictions_start_date": schema.StringAttribute{
						Optional:    true,
						Description: "Start date for historical predictions. May be passed if timeseries_settings.type=historical.",
					},
					"predictions_end_date": schema.StringAttribute{
						Optional:    true,
						Description: "End date for historical predictions. May be passed if timeseries_settings.type=historical.",
					},
					"relax_known_in_advance_features_check": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "If True, missing values in the known in advance features are allowed in the forecast window at the prediction time. Default is False.",
					},
				},
			},
			"num_concurrent": schema.Int64Attribute{
				Optional:    true,
				Description: "Number of concurrent chunks to score simultaneously. Defaults to the available number of cores of the deployment. Lower it to leave resources for real-time scoring.",
			},
			"chunk_size": schema.DynamicAttribute{
				Optional:    true,
				Description: "Which strategy should be used to determine the chunk size. Can be either a named strategy or a fixed size in bytes.",
			},
			"passthrough_columns": schema.ListAttribute{
				Optional:    true,
				Description: "Keep these columns from the scoring dataset in the scored dataset. This is useful for correlating predictions with source data.",
				ElementType: types.StringType,
			},
			"passthrough_columns_set": schema.StringAttribute{
				Optional:    true,
				Description: "To pass through every column from the scoring dataset, set this to all.",
				Validators: []validator.String{
					stringvalidator.OneOf("all"),
					stringvalidator.ConflictsWith(path.MatchRoot("passthrough_columns")),
				},
			},
			"max_explanations": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "Compute prediction explanations for this amount of features.",
			},
			"explanation_algorithm": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Which algorithm will be used to calculate prediction explanations.",
				Default:     stringdefault.StaticString("xemp"),
				Validators: []validator.String{
					stringvalidator.OneOf("shap", "xemp"),
				},
			},
			"threshold_high": schema.Float64Attribute{
				Optional:    true,
				Description: "Only compute prediction explanations for predictions above this threshold. Can be combined with threshold_low.",
			},
			"threshold_low": schema.Float64Attribute{
				Optional:    true,
				Description: "Only compute prediction explanations for predictions below this threshold. Can be combined with threshold_high.",
			},
			"prediction_warning_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Add prediction warnings to the scored data. Currently only supported for regression models. Defaults to False.",
			},
			"include_prediction_status": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Include the prediction_status column in the output. Defaults to False.",
			},
			"skip_drift_tracking": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Skips drift tracking on any predictions made from this job. This is useful when running non-production workloads to not affect drift tracking and cause unnecessary alerts. Defaults to false.",
			},
			"prediction_instance": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Defaults to instance specified by deployment or system configuration.",
				Attributes: map[string]schema.Attribute{
					"host_name": schema.StringAttribute{
						Required:    true,
						Description: "Hostname of the prediction instance.",
					},
					"ssl_enabled": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
						Description: "Set to false to run prediction requests from the batch prediction job without SSL. Defaults to true.",
					},
					"datarobot_key": schema.StringAttribute{
						Optional:    true,
						Description: "If running a job against a prediction instance in the Managed AI Cloud, you must provide the organization level DataRobot-Key.",
					},
					"api_key": schema.StringAttribute{
						Optional:    true,
						Description: "By default, prediction requests will use the API key of the user that created the job. This allows you to make requests on behalf of other users.",
					},
				},
			},
			"abort_on_error": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Default behavior is to abort the job if too many rows fail scoring. This will free up resources for other jobs that may score successfully. Set to false to unconditionally score every row no matter how many errors are encountered. Defaults to True.",
			},
			"column_names_remapping": schema.MapAttribute{
				Optional:    true,
				Description: "Mapping with column renaming for output table.",
				ElementType: types.StringType,
			},
			"include_probabilities": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Flag that enables returning of all probability columns. Defaults to True.",
			},
			"include_probabilities_classes": schema.ListAttribute{
				Optional:    true,
				Description: "List the subset of classes if a user doesn’t want all the classes. Defaults to [].",
				ElementType: types.StringType,
			},
			"prediction_threshold": schema.Float64Attribute{
				Optional:    true,
				Description: "Threshold is the point that sets the class boundary for a predicted value. This value can be set between 0.0 and 1.0.",
				Validators: []validator.Float64{
					float64validator.Between(0.0, 1.0),
				},
			},
		},
	}
}

func (r *BatchPredictionJobDefinitionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BatchPredictionJobDefinitionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BatchPredictionJobDefinitionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRequest, err := buildRequest(data)
	if err != nil {
		resp.Diagnostics.AddError("Error building Batch Prediction Job Definition request", err.Error())
		return
	}

	traceAPICall("CreateBatchPredictionJobDefinition")
	batchPredictionJobDefinition, err := r.provider.service.CreateBatchPredictionJobDefinition(ctx, createRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Batch Prediction Job Definition", err.Error())
		return
	}
	data.ID = types.StringValue(batchPredictionJobDefinition.ID)
	data.Name = types.StringValue(batchPredictionJobDefinition.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *BatchPredictionJobDefinitionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BatchPredictionJobDefinitionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetBatchPredictionJobDefinition")
	batchPredictionJobDefinition, err := r.provider.service.GetBatchPredictionJobDefinition(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Batch Prediction Job Definition not found",
				fmt.Sprintf("Batch Prediction Job Definition with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Batch Prediction Job Definition with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.Name = types.StringValue(batchPredictionJobDefinition.Name)
	data.Enabled = types.BoolValue(batchPredictionJobDefinition.Enabled)
	data.DeploymentID = types.StringValue(batchPredictionJobDefinition.BatchPredictionJob.DeploymentID)
	if batchPredictionJobDefinition.BatchPredictionJob.NumConcurrent != 0 {
		data.NumConcurrent = types.Int64Value(batchPredictionJobDefinition.BatchPredictionJob.NumConcurrent)
	}
	if len(batchPredictionJobDefinition.BatchPredictionJob.PassthroughColumns) > 0 {
		passThroughColumns := make([]types.String, 0, len(batchPredictionJobDefinition.BatchPredictionJob.PassthroughColumns))
		for _, column := range batchPredictionJobDefinition.BatchPredictionJob.PassthroughColumns {
			passThroughColumns = append(passThroughColumns, types.StringValue(column))
		}
		data.PassthroughColumns = passThroughColumns
	}
	data.PassthroughColumnsSet = types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.PassthroughColumnsSet)
	data.MaxExplanations = types.Int64Value(batchPredictionJobDefinition.BatchPredictionJob.MaxExplanations)
	data.ExplanationAlgorithm = types.StringValue(batchPredictionJobDefinition.BatchPredictionJob.ExplanationAlgorithm)
	data.AbortOnError = types.BoolValue(batchPredictionJobDefinition.BatchPredictionJob.AbortOnError)
	data.IncludePredictionStatus = types.BoolValue(batchPredictionJobDefinition.BatchPredictionJob.IncludePredictionStatus)
	data.IncludeProbabilities = types.BoolValue(batchPredictionJobDefinition.BatchPredictionJob.IncludeProbabilities)
	data.SkipDriftTracking = types.BoolValue(batchPredictionJobDefinition.BatchPredictionJob.SkipDriftTracking)
	data.PredictionWarningEnabled = types.BoolValue(batchPredictionJobDefinition.BatchPredictionJob.PredictionWarningEnabled)
	data.PredictionThreshold = types.Float64PointerValue(batchPredictionJobDefinition.BatchPredictionJob.PredictionThreshold)
	data.ThresholdHigh = types.Float64PointerValue(batchPredictionJobDefinition.BatchPredictionJob.ThresholdHigh)
	data.ThresholdLow = types.Float64PointerValue(batchPredictionJobDefinition.BatchPredictionJob.ThresholdLow)
	data.IntakeSettings = IntakeSettings{
		Type:         types.StringValue(batchPredictionJobDefinition.BatchPredictionJob.IntakeSettings.Type),
		DatasetID:    types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.IntakeSettings.DatasetID),
		File:         types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.IntakeSettings.File),
		URL:          types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.IntakeSettings.URL),
		CredentialID: types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.IntakeSettings.CredentialID),
		EndpointURL:  types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.IntakeSettings.EndpointURL),
		DataStoreID:  types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.IntakeSettings.DataStoreID),
		Query:        types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.IntakeSettings.Query),
		Table:        types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.IntakeSettings.Table),
		Schema:       types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.IntakeSettings.Schema),
		Catalog:      types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.IntakeSettings.Catalog),
		FetchSize:    types.Int64PointerValue(batchPredictionJobDefinition.BatchPredictionJob.IntakeSettings.FetchSize),
	}
	if batchPredictionJobDefinition.BatchPredictionJob.OutputSettings != nil {
		data.OutputSettings = &OutputSettings{
			Type:                   types.StringValue(batchPredictionJobDefinition.BatchPredictionJob.OutputSettings.Type),
			CredentialID:           types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.OutputSettings.CredentialID),
			URL:                    types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.OutputSettings.URL),
			EndpointURL:            types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.OutputSettings.EndpointURL),
			DataStoreID:            types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.OutputSettings.DataStoreID),
			Table:                  types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.OutputSettings.Table),
			Schema:                 types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.OutputSettings.Schema),
			Catalog:                types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.OutputSettings.Catalog),
			StatementType:          types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.OutputSettings.StatementType),
			CreateTableIfNotExists: types.BoolPointerValue(batchPredictionJobDefinition.BatchPredictionJob.OutputSettings.CreateTableIfNotExists),
		}
		if len(batchPredictionJobDefinition.BatchPredictionJob.OutputSettings.UpdateColumns) > 0 {
			data.OutputSettings.UpdateColumns = convertToTfStringList(batchPredictionJobDefinition.BatchPredictionJob.OutputSettings.UpdateColumns)
		}
		if len(batchPredictionJobDefinition.BatchPredictionJob.OutputSettings.WhereColumns) > 0 {
			data.OutputSettings.WhereColumns = convertToTfStringList(batchPredictionJobDefinition.BatchPredictionJob.OutputSettings.WhereColumns)
		}
	}
	if batchPredictionJobDefinition.BatchPredictionJob.CSVSettings != nil {
		data.CSVSettings = &CSVSettings{
			Delimiter: types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.CSVSettings.Delimiter),
			QuoteChar: types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.CSVSettings.QuoteChar),
			Encoding:  types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.CSVSettings.Encoding),
		}
	}
	if batchPredictionJobDefinition.BatchPredictionJob.PredictionInstance != nil {
		data.PredictionInstance = &PredictionInstance{
			HostName:     types.StringValue(batchPredictionJobDefinition.BatchPredictionJob.PredictionInstance.HostName),
			SSLEnabled:   types.BoolValue(batchPredictionJobDefinition.BatchPredictionJob.PredictionInstance.SSLEnabled),
			DatarobotKey: data.PredictionInstance.DatarobotKey,
			ApiKey:       data.PredictionInstance.ApiKey,
		}
	}
	if batchPredictionJobDefinition.BatchPredictionJob.TimeseriesSettings != nil {
		data.TimeseriesSettings = &TimeseriesSettings{
			Type:                             types.StringValue(batchPredictionJobDefinition.BatchPredictionJob.TimeseriesSettings.Type),
			ForecastPoint:                    types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.TimeseriesSettings.ForecastPoint),
			PredictionsStartDate:             types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.TimeseriesSettings.PredictionsStartDate),
			PredictionsEndDate:               types.StringPointerValue(batchPredictionJobDefinition.BatchPredictionJob.TimeseriesSettings.PredictionsEndDate),
			RelaxKnownInAdvanceFeaturesCheck: types.BoolPointerValue(batchPredictionJobDefinition.BatchPredictionJob.TimeseriesSettings.RelaxKnownInAdvanceFeaturesCheck),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BatchPredictionJobDefinitionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BatchPredictionJobDefinitionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest, err := buildRequest(data)
	if err != nil {
		resp.Diagnostics.AddError("Error building Batch Prediction Job Definition request", err.Error())
		return
	}

	traceAPICall("UpdateBatchPredictionJobDefinition")
	batchPredictionJobDefinition, err := r.provider.service.UpdateBatchPredictionJobDefinition(ctx, data.ID.ValueString(), updateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Batch Prediction Job Definition", err.Error())
		return
	}
	data.Name = types.StringValue(batchPredictionJobDefinition.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *BatchPredictionJobDefinitionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BatchPredictionJobDefinitionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteBatchPredictionJobDefinition")
	err := r.provider.service.DeleteBatchPredictionJobDefinition(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Batch Prediction Job Definition", err.Error())
			return
		}
	}
}

func (r *BatchPredictionJobDefinitionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildRequest(data BatchPredictionJobDefinitionResourceModel) (request *client.BatchPredictionJobDefinitionRequest, err error) {
	request = &client.BatchPredictionJobDefinitionRequest{
		Enabled:                     data.Enabled.ValueBool(),
		Name:                        StringValuePointerOptional(data.Name),
		DeploymentId:                data.DeploymentID.ValueString(),
		AbortOnError:                BoolValuePointerOptional(data.AbortOnError),
		ChunkSize:                   convertDynamicType(data.ChunkSize),
		ColumnNamesRemapping:        convertTfStringMap(data.ColumnNamesRemapping),
		ExplanationAlgorithm:        StringValuePointerOptional(data.ExplanationAlgorithm),
		IncludePredictionStatus:     BoolValuePointerOptional(data.IncludePredictionStatus),
		IncludeProbabilities:        BoolValuePointerOptional(data.IncludeProbabilities),
		IncludeProbabilitiesClasses: convertTfStringList(data.IncludeProbabilitiesClasses),
		IntakeSettings: &client.IntakeSettings{
			Type:         data.IntakeSettings.Type.ValueString(),
			DatasetID:    StringValuePointerOptional(data.IntakeSettings.DatasetID),
			File:         StringValuePointerOptional(data.IntakeSettings.File),
			URL:          StringValuePointerOptional(data.IntakeSettings.URL),
			CredentialID: StringValuePointerOptional(data.IntakeSettings.CredentialID),
			EndpointURL:  StringValuePointerOptional(data.IntakeSettings.EndpointURL),
			DataStoreID:  StringValuePointerOptional(data.IntakeSettings.DataStoreID),
			Query:        StringValuePointerOptional(data.IntakeSettings.Query),
			Table:        StringValuePointerOptional(data.IntakeSettings.Table),
			Schema:       StringValuePointerOptional(data.IntakeSettings.Schema),
			Catalog:      StringValuePointerOptional(data.IntakeSettings.Catalog),
			FetchSize:    Int64ValuePointerOptional(data.IntakeSettings.FetchSize),
		},
		MaxExplanations:          Int64ValuePointerOptional(data.MaxExplanations),
		NumConcurrent:            Int64ValuePointerOptional(data.NumConcurrent),
		PassthroughColumns:       convertTfStringList(data.PassthroughColumns),
		PassthroughColumnsSet:    StringValuePointerOptional(data.PassthroughColumnsSet),
		PredictionThreshold:      Float64ValuePointerOptional(data.PredictionThreshold),
		PredictionWarningEnabled: BoolValuePointerOptional(data.PredictionWarningEnabled),
		SkipDriftTracking:        BoolValuePointerOptional(data.SkipDriftTracking),
		ThresholdHigh:            Float64ValuePointerOptional(data.ThresholdHigh),
		ThresholdLow:             Float64ValuePointerOptional(data.ThresholdLow),
	}

	if data.Schedule != nil {
		var schedule client.Schedule
		if schedule, err = convertSchedule(*data.Schedule); err != nil {
			return
		}
		request.Schedule = &schedule
	}

	if data.OutputSettings != nil {
		request.OutputSettings = &client.OutputSettings{
			Type:                   data.OutputSettings.Type.ValueString(),
			CredentialID:           StringValuePointerOptional(data.OutputSettings.CredentialID),
			URL:                    StringValuePointerOptional(data.OutputSettings.URL),
			Path:                   StringValuePointerOptional(data.OutputSettings.Path),
			EndpointURL:            StringValuePointerOptional(data.OutputSettings.EndpointURL),
			DataStoreID:            StringValuePointerOptional(data.OutputSettings.DataStoreID),
			Table:                  StringValuePointerOptional(data.OutputSettings.Table),
			Schema:                 StringValuePointerOptional(data.OutputSettings.Schema),
			Catalog:                StringValuePointerOptional(data.OutputSettings.Catalog),
			StatementType:          StringValuePointerOptional(data.OutputSettings.StatementType),
			UpdateColumns:          convertTfStringList(data.OutputSettings.UpdateColumns),
			WhereColumns:           convertTfStringList(data.OutputSettings.WhereColumns),
			CreateTableIfNotExists: BoolValuePointerOptional(data.OutputSettings.CreateTableIfNotExists),
		}
	}

	if data.CSVSettings != nil {
		request.CSVSettings = &client.CSVSettings{
			Delimiter: StringValuePointerOptional(data.CSVSettings.Delimiter),
			QuoteChar: StringValuePointerOptional(data.CSVSettings.QuoteChar),
			Encoding:  StringValuePointerOptional(data.CSVSettings.Encoding),
		}
	}

	if data.PredictionInstance != nil {
		request.PredictionInstance = &client.PredictionInstance{
			HostName:     data.PredictionInstance.HostName.ValueString(),
			SSLEnabled:   data.PredictionInstance.SSLEnabled.ValueBool(),
			ApiKey:       StringValuePointerOptional(data.PredictionInstance.ApiKey),
			DatarobotKey: StringValuePointerOptional(data.PredictionInstance.DatarobotKey),
		}
	}

	if data.TimeseriesSettings != nil {
		request.TimeseriesSettings = &client.TimeseriesSettings{
			Type:                             data.TimeseriesSettings.Type.ValueString(),
			ForecastPoint:                    StringValuePointerOptional(data.TimeseriesSettings.ForecastPoint),
			RelaxKnownInAdvanceFeaturesCheck: BoolValuePointerOptional(data.TimeseriesSettings.RelaxKnownInAdvanceFeaturesCheck),
			PredictionsStartDate:             StringValuePointerOptional(data.TimeseriesSettings.PredictionsStartDate),
			PredictionsEndDate:               StringValuePointerOptional(data.TimeseriesSettings.PredictionsEndDate),
		}
	}

	return
}
