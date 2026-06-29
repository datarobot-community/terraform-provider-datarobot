package provider

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var _ datasource.DataSource = &ArtifactDataSource{}

func NewArtifactDataSource() datasource.DataSource {
	return &ArtifactDataSource{}
}

type ArtifactDataSource struct {
	provider *Provider
}

func (d *ArtifactDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_artifact"
}

func (d *ArtifactDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	probeAttributes := artifactDataSourceProbeAttributes()
	attributes := artifactDataSourceComputedAttributes(probeAttributes)
	attributes["artifact_id"] = datasourceschema.StringAttribute{
		Required:            true,
		MarkdownDescription: "The artifact version ID to look up.",
	}

	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Look up an existing Workload API artifact by ID. Returns the full artifact definition including spec, status, creator, tags, and permissions.",

		Attributes: attributes,
	}
}

func (d *ArtifactDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ArtifactDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ArtifactDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	artifactID := config.ArtifactID.ValueString()
	if artifactID == "" {
		resp.Diagnostics.AddError("Missing required attribute", "artifact_id must be specified to look up an artifact.")
		return
	}

	traceAPICall("GetArtifact")
	artifact, err := d.provider.service.GetArtifact(ctx, artifactID)
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddError(
				"Artifact not found",
				fmt.Sprintf("Artifact with ID %q was not found or is not accessible.", artifactID),
			)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Artifact with ID %s", artifactID),
				err.Error(),
			)
		}
		return
	}

	loadArtifactIntoDataSourceModel(artifact, &config)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
