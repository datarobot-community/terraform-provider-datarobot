package provider

import (
	"context"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// artifactImageURIUseStateForUnknown carries the prior image_uri forward across plans only
// when `source` is configured. Without `source`, image_uri behaves like a plain Optional
// attribute: dropping it from config plans it null (clearing it), instead of Computed
// silently keeping the last known value in state forever.
type artifactImageURIUseStateForUnknown struct{}

func (m artifactImageURIUseStateForUnknown) Description(ctx context.Context) string {
	return "Carries the prior image_uri forward across plans only when `source` is configured."
}

func (m artifactImageURIUseStateForUnknown) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m artifactImageURIUseStateForUnknown) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var source types.Object
	diags := req.Config.GetAttribute(ctx, path.Root("source"), &source)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if source.IsNull() {
		resp.PlanValue = req.ConfigValue
		return
	}

	stringplanmodifier.UseStateForUnknown().PlanModifyString(ctx, req, resp)
}

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
		"success_threshold": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Minimum consecutive successes for the probe to be considered successful after having failed.",
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
		"success_threshold": datasourceschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Minimum consecutive successes for the probe to be considered successful after having failed.",
		},
	}
}

func artifactResourceEnvironmentVarAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"source": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(client.EnvironmentVariableSourceString),
			MarkdownDescription: `Source type: "string" for plain text values, "dr-credential" for DataRobot credentials, or "api-key" for a platform-managed per-workload DataRobot API token. Defaults to "string".`,
		},
		"name": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: `Name of the environment variable. Required when source is "string" or "dr-credential". Optional for "api-key" (defaults to DATAROBOT_API_TOKEN).`,
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

func artifactResourceRouteAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"path": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Route path relative to the workload root, excluding the URL prefix the workload is mounted on. Must start with `/` and be at most 1024 characters. Paths must be unique within a container.",
			Validators:          RoutePathValidators(),
		},
		"auth": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: `Authentication applied to this route: "required" rejects unauthenticated requests, "optional" authenticates when an Authorization header is present, "disabled" never attempts authentication.`,
			Validators:          RouteAuthValidators(),
		},
	}
}

func artifactDataSourceRouteAttributes() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"path": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Route path relative to the workload root.",
		},
		"auth": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Authentication applied to this route.",
		},
	}
}

func artifactDataSourceEnvironmentVarAttributes() map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"source": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: `Source type: "string" for plain text values, "dr-credential" for DataRobot credentials, or "api-key" for a platform-managed per-workload DataRobot API token.`,
		},
		"name": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: `Name of the environment variable. May be absent for "api-key" entries, in which case the token is injected as DATAROBOT_API_TOKEN.`,
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

func artifactResourceBuildAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"artifact_image_build_id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Artifact image build ID.",
		},
		"status": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Image build status. With `source.wait_for_build` enabled (the default) this is the terminal status of the build the provider waited on; otherwise it is the status at submit time.",
		},
		"created_at": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Build creation timestamp (UTC).",
		},
	}
}

func artifactResourceContainerAttributes(probeAttributes, imageBuildConfigAttributes map[string]schema.Attribute) map[string]schema.Attribute {
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
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Docker image URI. Populated by the provider after a completed image build when `source` and `image_build_config` are set. May be set explicitly when not using source-driven builds.",
			PlanModifiers: []planmodifier.String{
				artifactImageURIUseStateForUnknown{},
			},
		},
		"image_build_config": schema.SingleNestedAttribute{
			Optional:            true,
			MarkdownDescription: "Configuration for server-side image builds from source code.",
			Attributes:          imageBuildConfigAttributes,
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
		"routes": schema.ListNestedAttribute{
			Optional: true,
			MarkdownDescription: "Routes to expose publicly from this container. Primary containers only, at most 50. " +
				"The workload root (`/`) is authenticated by default unless declared here with another policy. " +
				"Route configuration is a cluster-level capability that is disabled by default: setting this on a cluster " +
				"where it is not enabled fails with `Route configuration is disabled on this cluster`.",
			Validators: RoutesListValidators(),
			NestedObject: schema.NestedAttributeObject{
				Attributes: artifactResourceRouteAttributes(),
			},
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
		"build": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Server-set image build metadata.",
			Attributes:          artifactResourceBuildAttributes(),
			PlanModifiers: []planmodifier.Object{
				objectplanmodifier.UseStateForUnknown(),
			},
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
		"routes": datasourceschema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Routes exposed publicly from this container.",
			NestedObject: datasourceschema.NestedAttributeObject{
				Attributes: artifactDataSourceRouteAttributes(),
			},
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
					MarkdownDescription: "Image build status. With `source.wait_for_build` enabled (the default) this is the terminal status of the build the provider waited on; otherwise it is the status at submit time.",
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

func artifactResourceSpecAttribute(probeAttributes, imageBuildConfigAttributes map[string]schema.Attribute) schema.SingleNestedAttribute {
	containerAttributes := artifactResourceContainerAttributes(probeAttributes, imageBuildConfigAttributes)
	return schema.SingleNestedAttribute{
		Required:            true,
		MarkdownDescription: "The artifact specification containing container group definitions.",
		Attributes: map[string]schema.Attribute{
			"a2a_enabled": schema.BoolAttribute{
				Optional: true,
				MarkdownDescription: "Turns on agent-to-agent (A2A) card management and the A2A surface for this agent. " +
					"Valid only when `type` is `agent`. Defaults to off in the Workload API.",
			},
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

func artifactDataSourceComputedAttributes(probeAttributes map[string]datasourceschema.Attribute) map[string]datasourceschema.Attribute {
	return map[string]datasourceschema.Attribute{
		"artifact_id": datasourceschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The artifact version ID.",
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
			MarkdownDescription: "The artifact type: `service`, `nim`, `agent`, or `mcp`.",
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
	}
}

func artifactDataSourceSpecAttribute(probeAttributes map[string]datasourceschema.Attribute) datasourceschema.SingleNestedAttribute {
	containerAttributes := artifactDataSourceContainerAttributes(probeAttributes)
	return datasourceschema.SingleNestedAttribute{
		Computed:            true,
		MarkdownDescription: "The artifact specification containing container group definitions.",
		Attributes: map[string]datasourceschema.Attribute{
			"a2a_enabled": datasourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether A2A card management and the A2A surface are enabled. Set on `agent` artifacts; omitted otherwise.",
			},
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
