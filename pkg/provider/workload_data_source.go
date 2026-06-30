package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WorkloadDataSource struct {
	provider *Provider
}

func NewWorkloadDataSource() datasource.DataSource {
	return &WorkloadDataSource{}
}

func (d *WorkloadDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload"
}

func (d *WorkloadDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a Workload from the Workload API by ID. Exposes the full `WorkloadFormatted` response returned by `GET /workloads/{workload_id}`.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the Workload.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Name of the Workload.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Workload description.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp of when the Workload was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp of when the Workload was last updated.",
			},
			"creator": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Owner user details including ID, username and email.",
				Attributes:          workloadUserDataSchemaAttributes(),
			},
			"proton_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the currently active proton for this Workload.",
			},
			"artifact_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the currently active artifact for this Workload.",
			},
			"artifact": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Basic information about the currently active artifact for this Workload.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Unique identifier of the artifact.",
					},
					"name": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Name of the artifact.",
					},
					"type": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Artifact type.",
					},
					"status": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Artifact status.",
					},
					"version": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Version number of the artifact (set only for locked artifacts).",
					},
					"artifact_repository_id": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "ID of the artifact repository this artifact belongs to (for versioning).",
					},
					"template_id": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "ID of the template used to create this artifact.",
					},
				},
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Workload artifact type.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current Workload status.",
			},
			"replacement": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Information about an active replacement process for this Workload, if any.",
				Attributes: map[string]schema.Attribute{
					"status": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Replacement status.",
					},
					"candidate_proton_ids": schema.ListAttribute{
						Computed:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "IDs of protons pending promotion during artifact replacement.",
					},
					"strategy": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Replacement strategy.",
					},
				},
			},
			"runtime": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Runtime configuration sourced from the current proton.",
				Attributes: map[string]schema.Attribute{
					"container_groups": schema.ListNestedAttribute{
						Computed:            true,
						MarkdownDescription: "Per-group runtime configuration.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: workloadDataSourceGroupRuntimeSchemaAttributes(),
						},
					},
				},
			},
			"permissions": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "User permissions for this Workload.",
			},
			"importance": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Workload importance level.",
			},
			"request_stats": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Request statistics for this Workload.",
				Attributes: map[string]schema.Attribute{
					"total_requests": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Total number of requests.",
					},
					"concurrent_requests": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Number of concurrent requests.",
					},
					"last_request_at": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Timestamp of the last request.",
					},
					"response_time": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Average response time in milliseconds.",
					},
					"error_rate": schema.Float64Attribute{
						Computed:            true,
						MarkdownDescription: "Error rate percentage.",
					},
					"request_rates": schema.ListAttribute{
						Computed:            true,
						ElementType:         types.Int64Type,
						MarkdownDescription: "Request rates over the last 7 time periods.",
					},
					"error_rates": schema.ListAttribute{
						Computed:            true,
						ElementType:         types.Int64Type,
						MarkdownDescription: "Error rates over the last 7 time periods.",
					},
				},
			},
			"tags": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Tags associated with this Workload.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Unique identifier of the tag.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Name of the tag.",
						},
						"value": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Value of the tag.",
						},
					},
				},
			},
			"endpoint": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Workload endpoint URL.",
			},
			"last_response": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp of the last response received from this Workload.",
			},
			"owners": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of Workload owners.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: workloadUserDataSchemaAttributes(),
				},
			},
		},
	}
}

func workloadUserDataSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "User ID associated with this resource.",
		},
		"full_name": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "User's full name.",
		},
		"email": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "User email address.",
		},
		"username": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Username.",
		},
		"userhash": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "User's gravatar hash.",
		},
	}
}

func workloadDataSourceGroupRuntimeSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Group name. Must match a container group name declared in the artifact.",
		},
		"replica_count": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Number of replicas.",
		},
		"autoscaling": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Autoscaling configuration for this group.",
			Attributes: map[string]schema.Attribute{
				"enabled": schema.BoolAttribute{
					Computed:            true,
					MarkdownDescription: "Whether autoscaling is enabled.",
				},
				"policies": schema.ListNestedAttribute{
					Computed:            true,
					MarkdownDescription: "Scaling policies that define when and how to scale.",
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"scaling_metric": schema.StringAttribute{
								Computed:            true,
								MarkdownDescription: "Metric used for scaling decisions.",
							},
							"target": schema.Float64Attribute{
								Computed:            true,
								MarkdownDescription: "Target value for the scaling metric.",
							},
							"min_count": schema.Int64Attribute{
								Computed:            true,
								MarkdownDescription: "Minimum number of replicas.",
							},
							"max_count": schema.Int64Attribute{
								Computed:            true,
								MarkdownDescription: "Maximum number of replicas.",
							},
							"priority": schema.Int64Attribute{
								Computed:            true,
								MarkdownDescription: "Policy priority when multiple policies are defined.",
							},
						},
					},
				},
			},
		},
		"resource_bundles": schema.ListAttribute{
			Computed:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Ordered list of bundle IDs. One is selected at scheduling time.",
		},
		"bundle_selection_policy": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "How to select among resource_bundles.",
		},
		"containers": schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Per-container overrides for this group.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Container name.",
					},
					"resource_allocation": schema.SingleNestedAttribute{
						Computed:            true,
						MarkdownDescription: "Resource allocation for this container.",
						Attributes: map[string]schema.Attribute{
							"cpu": schema.Float64Attribute{
								Computed:            true,
								MarkdownDescription: "CPU cores allocated to this container.",
							},
							"gpu": schema.Float64Attribute{
								Computed:            true,
								MarkdownDescription: "GPUs allocated to this container.",
							},
							"gpu_memory": schema.StringAttribute{
								Computed:            true,
								MarkdownDescription: "GPU VRAM allocated.",
							},
							"memory": schema.StringAttribute{
								Computed:            true,
								MarkdownDescription: "RAM allocated to this container.",
							},
						},
					},
				},
			},
		},
		"resolved_bundle": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Full details of the bundle selected at scheduling time.",
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "Bundle identifier that was selected.",
				},
				"cpu_count": schema.Float64Attribute{
					Computed:            true,
					MarkdownDescription: "Number of CPU cores.",
				},
				"memory_bytes": schema.Int64Attribute{
					Computed:            true,
					MarkdownDescription: "Memory size in bytes.",
				},
				"gpu_count": schema.Int64Attribute{
					Computed:            true,
					MarkdownDescription: "Number of GPU units.",
				},
				"gpu_maker": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "GPU manufacturer.",
				},
				"gpu_type_label": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "GPU type label.",
				},
			},
		},
	}
}

func (d *WorkloadDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	var ok bool
	if d.provider, ok = req.ProviderData.(*Provider); !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected %T, got: %T. Please report this issue to the provider developers.", &Provider{}, req.ProviderData),
		)
	}
}

func (d *WorkloadDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WorkloadDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workloadID := config.ID.ValueString()
	if workloadID == "" {
		resp.Diagnostics.AddError("Missing required attribute", "The `id` attribute must be specified to look up a Workload.")
		return
	}

	workload, err := d.provider.service.GetWorkload(ctx, workloadID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Workload", err.Error())
		return
	}

	state := WorkloadDataSourceModel{ID: config.ID}
	loadWorkloadIntoDataSourceModel(workload, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
