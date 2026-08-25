# Common artifact shapes:
# - Prebuilt image: set image_uri (works with status = "locked" or "draft")
# - Build from source with local upload: set image_build_config and source { dir }
# - Build from source with existing catalog refs: set image_build_config.code_ref manually (no source block)
# - Agent: type = "agent" with optional spec.a2a_enabled for A2A card management

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
        # Extra paths to route from the workload's public endpoint, e.g. the
        # OAuth discovery document for an MCP server. "disabled" auth is
        # required here since clients fetch this before they have a token.
        routes = [{
          path = "/.well-known/oauth-authorization-server"
          auth = "disabled"
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
