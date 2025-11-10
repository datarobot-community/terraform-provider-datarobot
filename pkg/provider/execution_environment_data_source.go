package provider

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ExecutionEnvironmentDataSource struct {
	provider *Provider
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
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The name of the Execution Environment. Either `id` or `name` must be provided.",
			},
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The ID of the Execution Environment. Either `id` or `name` must be provided.",
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
				Optional:            true,
				MarkdownDescription: "The ID of the Execution Environment Version.",
			},
		},
	}
}

func (r *ExecutionEnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *ExecutionEnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ExecutionEnvironmentDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var executionEnvironment *client.ExecutionEnvironment
	var err error
	found := false

	if config.ID.ValueString() != "" {
		executionEnvironment, err = r.provider.service.GetExecutionEnvironment(ctx, config.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to get Execution Environment by ID", err.Error())
			return
		}
		found = true
	} else if config.Name.ValueString() != "" {
		executionEnvironments, err := r.provider.service.ListExecutionEnvironments(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list Execution Environments", err.Error())
			return
		}
		for idx := range executionEnvironments {
			if executionEnvironments[idx].Name == config.Name.ValueString() {
				executionEnvironment = &executionEnvironments[idx]
				found = true
				break
			}
		}
	} else {
		resp.Diagnostics.AddError("Missing required attributes", "Either 'id' or 'name' must be specified to look up an Execution Environment.")
		return
	}

	if !found {
		resp.Diagnostics.AddError("Execution Environment not found", fmt.Sprintf("Execution Environment with ID %q or name %q not found", config.ID.ValueString(), config.Name.ValueString()))
		return
	}

	executionEnvironmentVersion := &executionEnvironment.LatestVersion
	if config.VersionID.ValueString() != "" {
		version, err := r.provider.service.GetExecutionEnvironmentVersion(ctx, executionEnvironment.ID, config.VersionID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to get Execution Environment Version", err.Error())
			return
		}
		executionEnvironmentVersion = version
	}

	config.ID = types.StringValue(executionEnvironment.ID)
	config.Name = types.StringValue(executionEnvironment.Name)
	config.Description = types.StringValue(executionEnvironment.Description)
	config.ProgrammingLanguage = types.StringValue(executionEnvironment.ProgrammingLanguage)
	config.VersionID = types.StringValue(executionEnvironmentVersion.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
