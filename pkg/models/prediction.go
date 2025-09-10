package models

import "github.com/hashicorp/terraform-plugin-framework/types"

// BatchPredictionJobModel describes the batch prediction job resource.
type BatchPredictionJobDefinitionResourceModel struct {
	ID                          types.String        `tfsdk:"id"`
	DeploymentID                types.String        `tfsdk:"deployment_id"`
	Name                        types.String        `tfsdk:"name"`
	Enabled                     types.Bool          `tfsdk:"enabled"`
	Schedule                    *Schedule           `tfsdk:"schedule"`
	AbortOnError                types.Bool          `tfsdk:"abort_on_error"`
	ChunkSize                   types.Dynamic       `tfsdk:"chunk_size"`
	ColumnNamesRemapping        types.Map           `tfsdk:"column_names_remapping"`
	CSVSettings                 *CSVSettings        `tfsdk:"csv_settings"`
	ExplanationAlgorithm        types.String        `tfsdk:"explanation_algorithm"`
	IncludePredictionStatus     types.Bool          `tfsdk:"include_prediction_status"`
	IncludeProbabilities        types.Bool          `tfsdk:"include_probabilities"`
	IncludeProbabilitiesClasses []types.String      `tfsdk:"include_probabilities_classes"`
	IntakeSettings              IntakeSettings      `tfsdk:"intake_settings"`
	MaxExplanations             types.Int64         `tfsdk:"max_explanations"`
	NumConcurrent               types.Int64         `tfsdk:"num_concurrent"`
	OutputSettings              *OutputSettings     `tfsdk:"output_settings"`
	PassthroughColumns          []types.String      `tfsdk:"passthrough_columns"`
	PassthroughColumnsSet       types.String        `tfsdk:"passthrough_columns_set"`
	PredictionInstance          *PredictionInstance `tfsdk:"prediction_instance"`
	PredictionThreshold         types.Float64       `tfsdk:"prediction_threshold"`
	PredictionWarningEnabled    types.Bool          `tfsdk:"prediction_warning_enabled"`
	SkipDriftTracking           types.Bool          `tfsdk:"skip_drift_tracking"`
	ThresholdHigh               types.Float64       `tfsdk:"threshold_high"`
	ThresholdLow                types.Float64       `tfsdk:"threshold_low"`
	TimeseriesSettings          *TimeseriesSettings `tfsdk:"timeseries_settings"`
}


type IntakeSettings struct {
	Type         types.String `tfsdk:"type"`
	DatasetID    types.String `tfsdk:"dataset_id"`
	File         types.String `tfsdk:"file"`
	URL          types.String `tfsdk:"url"`
	CredentialID types.String `tfsdk:"credential_id"`
	EndpointURL  types.String `tfsdk:"endpoint_url"`
	DataStoreID  types.String `tfsdk:"data_store_id"`
	Query        types.String `tfsdk:"query"`
	Table        types.String `tfsdk:"table"`
	Schema       types.String `tfsdk:"schema"`
	Catalog      types.String `tfsdk:"catalog"`
	FetchSize    types.Int64  `tfsdk:"fetch_size"`
}



type Schedule struct {
	Minute     []types.String `tfsdk:"minute"`
	Hour       []types.String `tfsdk:"hour"`
	Month      []types.String `tfsdk:"month"`
	DayOfMonth []types.String `tfsdk:"day_of_month"`
	DayOfWeek  []types.String `tfsdk:"day_of_week"`
}



type TimeseriesSettings struct {
	ForecastPoint                    types.String `tfsdk:"forecast_point"`
	RelaxKnownInAdvanceFeaturesCheck types.Bool   `tfsdk:"relax_known_in_advance_features_check"`
	Type                             types.String `tfsdk:"type"`
	PredictionsStartDate             types.String `tfsdk:"predictions_start_date"`
	PredictionsEndDate               types.String `tfsdk:"predictions_end_date"`
}


type CSVSettings struct {
	Delimiter types.String `tfsdk:"delimiter"`
	Encoding  types.String `tfsdk:"encoding"`
	QuoteChar types.String `tfsdk:"quotechar"`
}


type PredictionInstance struct {
	ApiKey       types.String `tfsdk:"api_key"`
	DatarobotKey types.String `tfsdk:"datarobot_key"`
	HostName     types.String `tfsdk:"host_name"`
	SSLEnabled   types.Bool   `tfsdk:"ssl_enabled"`
}


type OutputSettings struct {
	CredentialID           types.String   `tfsdk:"credential_id"`
	Type                   types.String   `tfsdk:"type"`
	URL                    types.String   `tfsdk:"url"`
	Path                   types.String   `tfsdk:"path"`
	EndpointURL            types.String   `tfsdk:"endpoint_url"`
	DataStoreID            types.String   `tfsdk:"data_store_id"`
	Table                  types.String   `tfsdk:"table"`
	Schema                 types.String   `tfsdk:"schema"`
	Catalog                types.String   `tfsdk:"catalog"`
	StatementType          types.String   `tfsdk:"statement_type"`
	UpdateColumns          []types.String `tfsdk:"update_columns"`
	WhereColumns           []types.String `tfsdk:"where_columns"`
	CreateTableIfNotExists types.Bool     `tfsdk:"create_table_if_not_exists"`
}
