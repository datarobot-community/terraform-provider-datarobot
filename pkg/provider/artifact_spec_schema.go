package provider

import (
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func artifactResourceProbeAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"path": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "URL path to query for health check.",
		},
		"port": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Port number to access on the container.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"scheme": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Scheme to use for connecting to the host (HTTP or HTTPS).",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"host": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Host name to connect to, defaults to the pod IP.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"initial_delay_seconds": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Number of seconds to wait before the first probe is executed.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"period_seconds": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "How often (in seconds) to perform the probe.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"timeout_seconds": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Number of seconds after which the probe times out.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"failure_threshold": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Minimum consecutive failures for the probe to be considered failed.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
	}
}

func artifactDataSourceProbeAttributes() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"path": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "URL path to query for health check.",
		},
		"port": datasourceschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Port number to access on the container.",
		},
		"scheme": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Scheme to use for connecting to the host (HTTP or HTTPS).",
		},
		"host": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Host name to connect to, defaults to the pod IP.",
		},
		"initial_delay_seconds": datasourceschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Number of seconds to wait before the first probe is executed.",
		},
		"period_seconds": datasourceschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "How often (in seconds) to perform the probe.",
		},
		"timeout_seconds": datasourceschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Number of seconds after which the probe times out.",
		},
		"failure_threshold": datasourceschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Minimum consecutive failures for the probe to be considered failed.",
		},
	}
}

func artifactResourceEnvironmentVarAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"source": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(client.EnvironmentVariableSourceString),
			MarkdownDescription: `Source type: "string" for plain text values, "dr-credential" for DataRobot credentials. Defaults to "string".`,
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Name of the environment variable.",
		},
		"value": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: `Value of the environment variable. Required when source is "string".`,
		},
		"dr_credential_id": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: `DataRobot credential ID. Required when source is "dr-credential".`,
		},
		"key": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: `Key within the credential. Required when source is "dr-credential".`,
		},
	}
}

func artifactDataSourceEnvironmentVarAttributes() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"source": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: `Source type: "string" for plain text values, "dr-credential" for DataRobot credentials.`,
		},
		"name": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Name of the environment variable.",
		},
		"value": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: `Value of the environment variable when source is "string".`,
		},
		"dr_credential_id": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: `DataRobot credential ID when source is "dr-credential".`,
		},
		"key": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: `Key within the credential when source is "dr-credential".`,
		},
	}
}

func artifactResourceContainerAttributes(probeAttributes map[string]schema.Attribute) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Name of the container.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"image_uri": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Docker image URI.",
		},
		"primary": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Whether this is the primary container.",
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"description": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Description of the container.",
		},
		"port": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Container access port (1024-65535). Required for primary containers; omit for non-primary.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"entrypoint": schema.ListAttribute{
			Optional:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Container entrypoint.",
		},
		"environment_vars": schema.ListNestedAttribute{
			Optional:            true,
			MarkdownDescription: "Environment variables for the container.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: artifactResourceEnvironmentVarAttributes(),
			},
		},
		"startup_probe": schema.SingleNestedAttribute{
			Optional:            true,
			MarkdownDescription: "Container startup check configuration.",
			Attributes:          probeAttributes,
		},
		"readiness_probe": schema.SingleNestedAttribute{
			Optional:            true,
			MarkdownDescription: "Container readiness check configuration.",
			Attributes:          probeAttributes,
		},
		"liveness_probe": schema.SingleNestedAttribute{
			Optional:            true,
			MarkdownDescription: "Container liveness check configuration.",
			Attributes:          probeAttributes,
		},
	}
}

func artifactDataSourceContainerAttributes(probeAttributes map[string]datasourceschema.Attribute) map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"name": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Name of the container.",
		},
		"image_uri": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Docker image URI.",
		},
		"primary": datasourceschema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether this is the primary container.",
		},
		"description": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Description of the container.",
		},
		"port": datasourceschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Container access port (1024-65535).",
		},
		"entrypoint": datasourceschema.ListAttribute{
			Computed:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Container entrypoint.",
		},
		"environment_vars": datasourceschema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Environment variables for the container.",
			NestedObject: datasourceschema.NestedAttributeObject{
				Attributes: artifactDataSourceEnvironmentVarAttributes(),
			},
		},
		"startup_probe": datasourceschema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Container startup check configuration.",
			Attributes:          probeAttributes,
		},
		"readiness_probe": datasourceschema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Container readiness check configuration.",
			Attributes:          probeAttributes,
		},
		"liveness_probe": datasourceschema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Container liveness check configuration.",
			Attributes:          probeAttributes,
		},
		"image_build_config": datasourceschema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Configuration for server-side image builds from source code.",
			Attributes: map[string]datasourceschema.Attribute{
				"code_ref": datasourceschema.SingleNestedAttribute{
					Computed:            true,
					MarkdownDescription: "Reference to source code in the DataRobot catalog.",
					Attributes: map[string]datasourceschema.Attribute{
						"provider": datasourceschema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Code provider.",
						},
						"type": datasourceschema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Code reference type.",
						},
						"datarobot": datasourceschema.SingleNestedAttribute{
							Computed:            true,
							MarkdownDescription: "DataRobot catalog reference.",
							Attributes: map[string]datasourceschema.Attribute{
								"catalog_id": datasourceschema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Catalog ID.",
								},
								"catalog_version_id": datasourceschema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Catalog version ID.",
								},
							},
						},
					},
				},
				"dockerfile": datasourceschema.SingleNestedAttribute{
					Computed:            true,
					MarkdownDescription: "Dockerfile configuration for image builds.",
					Attributes: map[string]datasourceschema.Attribute{
						"source": datasourceschema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Dockerfile source: `provided` or `generated`.",
						},
						"path": datasourceschema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Relative path to the Dockerfile when source is `provided`.",
						},
						"entrypoint": datasourceschema.ListAttribute{
							Computed:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "Entrypoint when source is `generated`.",
						},
						"execution_environment_id": datasourceschema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Execution environment ID when source is `generated`.",
						},
						"execution_environment_version_id": datasourceschema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Execution environment version ID when source is `generated`.",
						},
					},
				},
			},
		},
		"build": datasourceschema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Server-set image build metadata.",
			Attributes: map[string]datasourceschema.Attribute{
				"artifact_image_build_id": datasourceschema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "Artifact image build ID.",
				},
				"status": datasourceschema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "Image build status at submit time.",
				},
				"created_at": datasourceschema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "Build creation timestamp (UTC).",
				},
			},
		},
		"security_context": datasourceschema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Container security context.",
			Attributes: map[string]datasourceschema.Attribute{
				"allow_privilege_escalation": datasourceschema.BoolAttribute{
					Computed:            true,
					MarkdownDescription: "Whether a process can gain more privileges than its parent.",
				},
				"read_only_root_filesystem": datasourceschema.BoolAttribute{
					Computed:            true,
					MarkdownDescription: "Whether the root filesystem is read-only.",
				},
				"capabilities": datasourceschema.SingleNestedAttribute{
					Computed:            true,
					MarkdownDescription: "Linux capabilities to add or drop.",
					Attributes: map[string]datasourceschema.Attribute{
						"add": datasourceschema.ListAttribute{
							Computed:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "Capabilities to add.",
						},
						"drop": datasourceschema.ListAttribute{
							Computed:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "Capabilities to drop.",
						},
					},
				},
				"seccomp_profile": datasourceschema.SingleNestedAttribute{
					Computed:            true,
					MarkdownDescription: "Seccomp profile for the container.",
					Attributes: map[string]datasourceschema.Attribute{
						"type": datasourceschema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Seccomp profile type.",
						},
						"localhost_profile": datasourceschema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Path to a seccomp profile on the node when type is Localhost.",
						},
					},
				},
			},
		},
	}
}

func artifactResourceSpecAttribute(probeAttributes map[string]schema.Attribute) schema.SingleNestedAttribute {
	containerAttributes := artifactResourceContainerAttributes(probeAttributes)
	return schema.SingleNestedAttribute{
		Required:            true,
		MarkdownDescription: "The artifact specification containing container group definitions.",
		Attributes: map[string]schema.Attribute{
			"container_groups": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "List of container groups.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"containers": schema.ListNestedAttribute{
							Required:            true,
							MarkdownDescription: "List of containers in this group.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: containerAttributes,
							},
						},
					},
				},
			},
		},
	}
}

func artifactDataSourceSpecAttribute(probeAttributes map[string]datasourceschema.Attribute) datasourceschema.SingleNestedAttribute {
	containerAttributes := artifactDataSourceContainerAttributes(probeAttributes)
	return datasourceschema.SingleNestedAttribute{
		Computed:            true,
		MarkdownDescription: "The artifact specification containing container group definitions.",
		Attributes: map[string]datasourceschema.Attribute{
			"container_groups": datasourceschema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of container groups.",
				NestedObject: datasourceschema.NestedAttributeObject{
					Attributes: map[string]datasourceschema.Attribute{
						"name": datasourceschema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Name of the container group.",
						},
						"containers": datasourceschema.ListNestedAttribute{
							Computed:            true,
							MarkdownDescription: "List of containers in this group.",
							NestedObject: datasourceschema.NestedAttributeObject{
								Attributes: containerAttributes,
							},
						},
					},
				},
			},
			"storage": datasourceschema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "NIM model weight storage configuration.",
				Attributes: map[string]datasourceschema.Attribute{
					"mode": datasourceschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Storage mode: `dedicatedPvc` or `nimCache`.",
					},
					"pvc_size": datasourceschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "PVC size for dedicated storage (e.g. `150Gi`).",
					},
				},
			},
			"template_id": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the template used to create this NIM artifact.",
			},
		},
	}
}
