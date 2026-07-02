package provider

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Look up an existing Workload API artifact by ID. Returns the full artifact definition including spec, status, creator, tags, and permissions.",

		Attributes: map[string]datasourceschema.Attribute{
			"artifact_id": datasourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The artifact version ID to look up.",
			},
			"name": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The name of the Artifact.",
			},
			"description": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the Artifact.",
			},
			"type": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The artifact type: `service` or `nim`.",
			},
			"status": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Artifact status: `draft` or `locked`.",
			},
			"version": datasourceschema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Version number of the artifact. Set only for locked artifacts.",
			},
			"artifact_repository_id": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the artifact repository for versioning.",
			},
			"created_at": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp of when the artifact was created.",
			},
			"updated_at": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp of when the artifact was last updated.",
			},
			"creator": datasourceschema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "User who created the artifact.",
				Attributes: map[string]datasourceschema.Attribute{
					"id": datasourceschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "User ID.",
					},
					"full_name": datasourceschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "User's full name.",
					},
					"email": datasourceschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "User email address.",
					},
					"username": datasourceschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Username.",
					},
					"userhash": datasourceschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "User's gravatar hash.",
					},
				},
			},
			"tags": datasourceschema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Tags associated with this artifact.",
				NestedObject: datasourceschema.NestedAttributeObject{
					Attributes: map[string]datasourceschema.Attribute{
						"id": datasourceschema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Tag ID.",
						},
						"name": datasourceschema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Tag name.",
						},
						"value": datasourceschema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Tag value.",
						},
					},
				},
			},
			"permissions": datasourceschema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Effective repository-level permissions for the authenticated user.",
			},
			"spec": artifactDataSourceSpecAttribute(probeAttributes),
		},
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
