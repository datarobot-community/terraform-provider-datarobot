# Common artifact shapes:
# - Prebuilt image: set image_uri (works with status = "locked" or "draft")
# - Build from source with local upload: set image_build_config and source { dir }
# - Build from source with existing catalog refs: set image_build_config.code_ref manually (no source block)
# - Agent: type = "agent" with optional spec.a2a_enabled for A2A card management
# - MCP: type = "mcp" (same spec shape as service)

resource "datarobot_artifact" "prebuilt" {
  name        = "example-prebuilt-service"
  description = "Artifact with a prebuilt container image"
  status      = "locked" # default; use "draft" for in-place updates

  spec = {
    container_groups = [{
      containers = [{
        image_uri = "nginx:latest"
        primary   = true
        port      = 8080
        # Extra paths to expose from the workload's public endpoint, each with
        # its own auth policy: "required", "optional", or "disabled". Reserve
        # "disabled" for documents a client must fetch before it holds a token,
        # such as an MCP server's OAuth discovery document.
        # Route configuration is disabled by default at the cluster level; on a
        # cluster without it, this block fails with
        # "Route configuration is disabled on this cluster".
        routes = [{
          path = "/index.html"
          auth = "required"
        }]
      }]
    }]
  }
}

# Create as draft so this example is copy-pasteable. After the image build
# populates image_uri, set status = "locked". Applying locked without image_uri
# is rejected by workload-api (422).
#
# wait_for_build = true (the default; set explicitly here) makes apply block
# until the image build finishes and streams build log lines to the
# provider's stderr while it waits. Terraform only shows provider stderr
# when TF_LOG is set (TF_LOG=DEBUG or more verbose - it's also emitted via
# tflog.Debug); on a plain `terraform apply` with TF_LOG unset, apply still
# blocks until the build finishes, it just prints nothing in between.
resource "datarobot_artifact" "from_source" {
  name        = "example-c2w-draft"
  description = "Draft artifact with local source upload (code-to-workload)"
  status      = "draft"

  # apply synchronizes this directory with the catalog in both directions:
  # local changes are uploaded, catalog-only files are downloaded and files
  # deleted from the catalog are removed locally (each reported as a warning).
  # A file edited on both sides since the last sync fails the apply; resolve
  # it with `dr artifact code sync` and apply again. Sync bookkeeping lives in
  # app/.datarobot/workload/ (it ships its own .gitignore). .datarobot.yaml is
  # never uploaded.
  source = {
    dir            = "${path.module}/app"
    wait_for_build = true
    # generate_ignore = true  # default: write .drignore if missing; never overwrite
  }

  spec = {
    container_groups = [{
      containers = [{
        name    = "primary"
        primary = true
        port    = 8080

        image_build_config = {
          # code_ref is populated automatically from source.dir after upload

          dockerfile = {
            source = "provided"
            path   = "./Dockerfile"
          }
        }
      }]
    }]
  }
}

output "prebuilt_artifact_id" {
  value       = datarobot_artifact.prebuilt.artifact_id
  description = "Artifact ID for the prebuilt-image example"
}

output "from_source_artifact_id" {
  value       = datarobot_artifact.from_source.artifact_id
  description = "Artifact ID for the draft image-build example (stable until lock)"
}

resource "datarobot_artifact" "from_source_locked" {
  name        = "example-c2w-locked"
  description = "Locked artifact with local source upload (create as draft → upload → build → lock)"
  # The provider creates a draft, uploads source, triggers a build (waits by default), then locks.
  status = "locked"

  source = {
    dir = "${path.module}/app"
    # generate_ignore = true  # default: write .drignore if missing; never overwrite
  }

  spec = {
    container_groups = [{
      containers = [{
        name    = "primary"
        primary = true
        port    = 8080

        image_build_config = {
          dockerfile = {
            source = "provided"
            path   = "./Dockerfile"
          }
        }
      }]
    }]
  }
}

output "from_source_locked_artifact_id" {
  value       = datarobot_artifact.from_source_locked.artifact_id
  description = "Artifact ID for the locked image-build example (new version on source change)"
}

# Build from source with a DataRobot-generated Dockerfile. The base image comes
# from an execution environment, so the two ids are read from a variable or a
# data source rather than pasted in: every attribute below accepts a variable,
# data source, or resource reference, and only a null value counts as unset.
#
# Set both variables to build this example; left empty it plans and applies as
# nothing, so the rest of this file works on any instance. Execution environment
# names differ between instances, which is why this does not look one up by a
# hardcoded name. To resolve them by name instead, add the data source and point
# the two attributes at it:
#
#   data "datarobot_execution_environment" "python" {
#     name = "<a name that exists on your instance>"
#   }
#
#   execution_environment_id         = data.datarobot_execution_environment.python.id
#   execution_environment_version_id = data.datarobot_execution_environment.python.version_id
variable "execution_environment_id" {
  type        = string
  description = "Execution environment supplying the base image for the generated Dockerfile. Empty disables that example."
  default     = ""
}

variable "execution_environment_version_id" {
  type        = string
  description = "Execution environment version pinning the base image. Empty disables the generated-Dockerfile example."
  default     = ""
}

resource "datarobot_artifact" "generated_dockerfile" {
  count = var.execution_environment_id != "" && var.execution_environment_version_id != "" ? 1 : 0

  name        = "example-generated-dockerfile"
  description = "Draft artifact whose Dockerfile is generated from an execution environment"
  status      = "draft"

  source = {
    dir = "${path.module}/app"
  }

  spec = {
    container_groups = [{
      containers = [{
        name    = "primary"
        primary = true
        port    = 8080

        image_build_config = {
          dockerfile = {
            source                           = "generated"
            execution_environment_id         = var.execution_environment_id
            execution_environment_version_id = var.execution_environment_version_id
            entrypoint                       = ["python", "app.py"]
          }
        }
      }]
    }]
  }
}

output "generated_dockerfile_artifact_id" {
  value       = one(datarobot_artifact.generated_dockerfile[*].artifact_id)
  description = "Artifact ID for the generated-Dockerfile example; null unless the execution environment variables are set"
}

resource "datarobot_artifact" "agent" {
  name        = "example-agent"
  description = "Agent artifact with A2A card management enabled"
  type        = "agent"

  spec = {
    a2a_enabled = true
    container_groups = [{
      containers = [{
        image_uri = "nginx:latest"
        primary   = true
        port      = 8080
      }]
    }]
  }
}

output "agent_artifact_id" {
  value       = datarobot_artifact.agent.artifact_id
  description = "Artifact ID for the agent example"
}

resource "datarobot_artifact" "mcp" {
  name        = "example-mcp"
  description = "MCP server artifact"
  type        = "mcp"

  spec = {
    container_groups = [{
      containers = [{
        image_uri = "nginx:latest"
        primary   = true
        port      = 8080
      }]
    }]
  }
}

output "mcp_artifact_id" {
  value       = datarobot_artifact.mcp.artifact_id
  description = "Artifact ID for the MCP example"
}
