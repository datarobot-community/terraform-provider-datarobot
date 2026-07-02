package provider

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ datasource.DataSource = &ArtifactsDataSource{}

func NewArtifactsDataSource() datasource.DataSource {
	return &ArtifactsDataSource{}
}

type ArtifactsDataSource struct {
	provider *Provider
}

func (d *ArtifactsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_artifacts"
}

func (d *ArtifactsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	probeAttributes := artifactDataSourceProbeAttributes()

	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "List Workload API artifacts with optional status filter and result limit. Each entry mirrors the full artifact API response, including spec.",

		Attributes: map[string]datasourceschema.Attribute{
			"status": datasourceschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter artifacts by status: `draft` or `locked`.",
				Validators: []validator.String{
					stringvalidator.OneOf(string(client.ArtifactStatusDraft), string(client.ArtifactStatusLocked)),
				},
			},
			"limit": datasourceschema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum number of artifacts to return. When omitted, all matching artifacts are returned (paginated in pages of 100).",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"artifacts": datasourceschema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Artifacts matching the filter.",
				NestedObject: datasourceschema.NestedAttributeObject{
					Attributes: artifactDataSourceComputedAttributes(probeAttributes),
				},
			},
		},
	}
}

func (d *ArtifactsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	var ok bool
	if d.provider, ok = req.ProviderData.(*Provider); !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected %T, got: %T. Please report this issue to the provider developers.", Provider{}, req.ProviderData),
		)
	}
}

func (d *ArtifactsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ArtifactsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	listReq := &client.ListArtifactsRequest{}
	if !config.Status.IsNull() && !config.Status.IsUnknown() {
		listReq.Status = config.Status.ValueString()
	}
	if !config.Limit.IsNull() && !config.Limit.IsUnknown() {
		listReq.Limit = int(config.Limit.ValueInt64())
	}

	traceAPICall("ListArtifacts")
	artifacts, err := d.provider.service.ListArtifacts(ctx, listReq)
	if err != nil {
		resp.Diagnostics.AddError("Error listing artifacts", err.Error())
		return
	}

	config.Artifacts = make([]ArtifactDataSourceModel, len(artifacts))
	for i := range artifacts {
		loadArtifactIntoDataSourceModel(&artifacts[i], &config.Artifacts[i])
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
