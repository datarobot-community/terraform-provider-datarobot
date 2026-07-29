# Two common artifact shapes:
# - Prebuilt image: set image_uri (works with status = "locked" or "draft")
# - Build from source: set image_build_config on a draft artifact (omit image_uri until after build)

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

resource "datarobot_artifact" "from_source" {
  name        = "example-c2w-draft"
  description = "Draft artifact with image build configuration (code-to-workload)"
  status      = "draft"

  spec = {
    container_groups = [{
      containers = [{
        name    = "primary"
        primary = true
        port    = 8080

        image_build_config = {
          # code_ref is optional at create; required before build or lock
          # code_ref = {
          #   catalog_id         = "<24-char-hex>"
          #   catalog_version_id = "<24-char-hex>"
          # }

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
