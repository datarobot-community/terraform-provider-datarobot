package environment

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/common"
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ExecutionEnvironmentDataSource struct {
	service client.Service
}

func NewExecutionEnvironmentDataSource() datasource.DataSource {
	return &ExecutionEnvironmentDataSource{}
}

func (d *ExecutionEnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_execution_environment"
}

func (r *ExecutionEnvironmentDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Execution Environment",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Execution Environment.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Execution Environment.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the Execution Environment.",
			},
			"programming_language": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The programming language of the Execution Environment.",
			},
			"version_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Execution Environment Version.",
			},
		},
	}
}

func (r *ExecutionEnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil { return }
	accessor, ok := req.ProviderData.(common.ServiceAccessor)
	if !ok {
			resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
			return
	}
	r.service = accessor.GetService()
}

func (r *ExecutionEnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.ExecutionEnvironmentDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	executionEnvironments, err := r.service.ListExecutionEnvironments(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Execution Environments", err.Error())
		return
	}

	found := false
	var executionEnvironment client.ExecutionEnvironment
	for _, executionEnvironment = range executionEnvironments {
		if executionEnvironment.Name == config.Name.ValueString() {
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError("Execution Environment not found", fmt.Sprintf("Execution Environment with name %q not found", config.Name.ValueString()))
		return
	}

	config.ID = types.StringValue(executionEnvironment.ID)
	config.Name = types.StringValue(executionEnvironment.Name)
	config.Description = types.StringValue(executionEnvironment.Description)
	config.ProgrammingLanguage = types.StringValue(executionEnvironment.ProgrammingLanguage)
	config.VersionID = types.StringValue(executionEnvironment.LatestVersion.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
