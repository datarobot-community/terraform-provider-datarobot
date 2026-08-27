# Basic: deploy a prebuilt container image as a workload, with an in-place
# replacement policy. Artifact and runtime changes roll out in place; the
# workload id and endpoint stay stable.

resource "datarobot_artifact" "example" {
  name = "example-workload-artifact"
  type = "service"

  spec = {
    container_groups = [
      {
        containers = [
          {
            name       = "main"
            image_uri  = "containous/whoami:latest"
            port       = 8080
            primary    = true
            entrypoint = ["/whoami", "--port", "8080"]
          }
        ]
      }
    ]
  }
}

resource "datarobot_workload" "example" {
  name        = "example-workload"
  description = "Example workload with in-place replacement"
  artifact_id = datarobot_artifact.example.artifact_id

  runtime = {
    container_groups = [
      {
        replica_count    = 2
        resource_bundles = ["cpu.small"]
      }
    ]
    replacement_policy = {
      warmup_minutes           = 5
      keep_old_version_minutes = 10
    }
  }
}

output "datarobot_workload_id" {
  value       = datarobot_workload.example.id
  description = "Stable workload ID — unchanged when the artifact spec is updated"
}

output "datarobot_workload_endpoint" {
  value       = datarobot_workload.example.endpoint
  description = "Stable inference endpoint — unchanged across in-place replacements"
}

# Code-to-workload: upload local FastAPI source, build a container image, lock the artifact, deploy a workload.
# The app runs uvicorn on port 8080 so the workload stays healthy (a one-shot script would exit and fail).

resource "datarobot_artifact" "app" {
  name        = "example-c2w-workload"
  description = "Artifact built from local source for workload deployment"
  status      = "locked"

  source = {
    dir = "${path.module}/app"
    # wait_for_build defaults to true: upload, trigger build, poll until image_uri is ready, then lock
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

resource "datarobot_workload" "api" {
  name        = "example-c2w-workload"
  description = "Workload serving the built artifact"
  importance  = "low"
  artifact_id = datarobot_artifact.app.artifact_id

  runtime = {
    container_groups = [{
      replica_count    = 1
      resource_bundles = ["cpu.small"]
    }]
  }

  depends_on = [datarobot_artifact.app]
}

output "artifact_id" {
  value       = datarobot_artifact.app.artifact_id
  description = "Locked artifact version ID deployed by the workload"
}

output "workload_endpoint" {
  value       = datarobot_workload.api.endpoint
  description = "Inference endpoint URL for the deployed workload"
}
