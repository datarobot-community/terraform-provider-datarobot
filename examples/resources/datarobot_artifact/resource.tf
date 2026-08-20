# Three common artifact shapes:
# - Prebuilt image: set image_uri (works with status = "locked" or "draft")
# - Build from source with local upload: set image_build_config and source { dir }
# - Build from source with existing catalog refs: set image_build_config.code_ref manually (no source block)

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
      }]
    }]
  }
}

# Create as draft so this example is copy-pasteable. After the image build
# populates image_uri, set status = "locked". Applying locked without image_uri
# is rejected by workload-api (422).
resource "datarobot_artifact" "from_source" {
  name        = "example-c2w-draft"
  description = "Draft artifact with local source upload (code-to-workload)"
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
  description = "Locked artifact with local source upload (clone → upload → lock)"
  # Note: workload-api requires a completed image build (image_uri populated) before lock.
  # Create as draft, run image build, then set status = "locked" — or use this block only
  # after image_uri is available from a prior build on the same artifact version.
  status = "locked"

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
