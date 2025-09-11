package govern

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

type GlobalModelDataSource struct {
	service client.Service
}

func NewGlobalModelDataSource() datasource.DataSource {
	return &GlobalModelDataSource{}
}

func (d *GlobalModelDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_model"
}

func (r *GlobalModelDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Global Model",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Registered Model.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Global Model.",
			},
			"version_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Global Model Version.",
			},
		},
	}
}

func (r *GlobalModelDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil { return }
	accessor, ok := req.ProviderData.(common.ServiceAccessor)
	if !ok {
			resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
			return
	}
	r.service = accessor.GetService()
}

func (r *GlobalModelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.GlobalModelDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	registeredModels, err := r.service.ListRegisteredModels(ctx, &client.ListRegisteredModelsRequest{
		IsGlobal: true,
		Search:   config.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Global Models", err.Error())
		return
	}

	if len(registeredModels) == 0 {
		resp.Diagnostics.AddError("Global Model not found", fmt.Sprintf("Global Model with name %q not found", config.Name.ValueString()))
		return
	}

	var globalModel *client.RegisteredModel
	for i := range registeredModels {
		model := registeredModels[i]
		if model.Name == config.Name.ValueString() {
			globalModel = &model
		}
	}
	if globalModel == nil {
		resp.Diagnostics.AddError("Global Model not found", fmt.Sprintf("Global Model with name %q not found", config.Name.ValueString()))
		return
	}

	globalModelVersion, err := r.service.GetLatestRegisteredModelVersion(ctx, globalModel.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Global Model version", err.Error())
		return
	}

	config.ID = types.StringValue(globalModel.ID)
	config.Name = types.StringValue(globalModel.Name)
	config.VersionID = types.StringValue(globalModelVersion.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
