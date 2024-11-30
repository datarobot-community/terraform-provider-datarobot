package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure Provider satisfies various provider interfaces.
var _ provider.Provider = &Provider{}
var _ provider.ProviderWithFunctions = &Provider{}

// NewService overrides the client method for testing.
var NewService = client.NewService

// Provider defines the provider implementation.
type Provider struct {
	// client can contain the upstream provider SDK or HTTP client used to
	// communicate with the upstream service. Resource and DataSource
	// implementations can then make calls using this client.
	service client.Service

	// configured is set to true at the end of the Configure method.
	// This can be used in Resource and DataSource implementations to verify
	// that the provider was previously configured.
	configured bool
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ProviderModel describes the provider data model.
type ProviderModel struct {
	Endpoint     types.String `tfsdk:"endpoint"`
	ApiKey       types.String `tfsdk:"apikey"`
	TraceContext types.String `tfsdk:"tracecontext"`
}

func (p *Provider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "datarobot"
	resp.Version = p.version
}

func (p *Provider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Endpoint for the DataRobot API",
				Optional:            true,
				Sensitive:           true,
			},
			"apikey": schema.StringAttribute{
				MarkdownDescription: "Key to access DataRobot API",
				Optional:            true,
				Sensitive:           true,
			},
			"tracecontext": schema.StringAttribute{
				MarkdownDescription: "DataRobot trace context",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *Provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check if the endpoint and token are set in the environment or in the configuration
	var endpoint string
	if !IsKnown(data.Endpoint) {
		endpoint = os.Getenv(DataRobotEndpointEnvVar)
	} else {
		endpoint = data.Endpoint.ValueString()
	}

	var apiKey string
	if !IsKnown(data.ApiKey) {
		apiKey = os.Getenv(DataRobotApiKeyEnvVar)
	} else {
		apiKey = data.ApiKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Unable to find Api Key",
			"Api Key cannot be an empty string")
		return
	}

	var traceContext string
	if !IsKnown(data.TraceContext) {
		traceContext = os.Getenv(DataRobotTraceContextEnvVar)
	} else {
		traceContext = data.TraceContext.ValueString()
	}

	// Create a new client configuration
	cfg := client.NewConfiguration(apiKey)
	if endpoint != "" {
		cfg.Endpoint = endpoint
	}

	cfg.UserAgent = fmt.Sprintf("%s/%s Terraform-%s", UserAgent, p.version, req.TerraformVersion)
	cfg.TraceContext = traceContext

	// set debug mode if TF_LOG is set to DEBUG or TRACE
	logLevel := os.Getenv("TF_LOG")
	if logLevel == "DEBUG" || logLevel == "TRACE" {
		cfg.Debug = true
	} else {
		logLevel = os.Getenv("TF_LOG_PROVIDER")
		if logLevel == "DEBUG" || logLevel == "TRACE" {
			cfg.Debug = true
		}
	}

	// retryablehttp gives us automatic retries with exponential backoff.
	httpClient := retryablehttp.NewClient()
	// The TF framework will pick up the default global logger.
	// HTTP requests are logged at DEBUG level.
	httpClient.Logger = &leveledTFLogger{baseCtx: ctx}
	httpClient.ErrorHandler = retryablehttp.PassthroughErrorHandler
	httpClient.CheckRetry = retryGetRequestsOnly
	cfg.HTTPClient = httpClient.StandardClient()

	// Example client configuration for data sources and resources
	cl := client.NewClient(cfg)
	p.service = NewService(cl)
	resp.DataSourceData = p
	resp.ResourceData = p

	p.configured = true
}

func (p *Provider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUseCaseResource,
		NewRemoteRepositoryResource,
		NewDatasetFromFileResource,
		NewDatasetFromURLResource,
		NewDatasetFromDatasourceResource,
		NewDatastoreResource,
		NewDatasourceResource,
		NewVectorDatabaseResource,
		NewPlaygroundResource,
		NewLLMBlueprintResource,
		NewCustomModelResource,
		NewCustomJobResource,
		NewCustomMetricJobResource,
		NewCustomMetricFromJobResource,
		NewRegisteredModelResource,
		NewRegisteredModelFromLeaderboardResource,
		NewPredictionEnvironmentResource,
		NewDeploymentResource,
		NewDeploymentRetrainingPolicyResource,
		NewQAApplicationResource,
		NewCustomApplicationResource,
		NewApplicationSourceResource,
		NewApiTokenCredentialResource,
		NewBasicCredentialResource,
		NewGoogleCloudCredentialResource,
		NewExecutionEnvironmentResource,
		NewBatchPredictionJobDefinitionResource,
	}
}

func (p *Provider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewGlobalModelDataSource,
		NewExecutionEnvironmentDataSource,
	}
}

func (p *Provider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &Provider{
			version: version,
		}
	}
}
