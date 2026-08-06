variable "workload_name" {
  type        = string
  description = "Name for the artifact and workload resources"
  default     = "workload-replacement-example"
}

variable "container_image" {
  type        = string
  description = "Container image for the artifact. Change this and re-apply to trigger an in-place workload replacement."
  default     = "containous/whoami:latest"
}

variable "replica_count" {
  type        = number
  description = "Number of workload replicas. Changing only this triggers a runtime settings update (same artifact)."
  default     = 2
}

terraform {
  required_providers {
    datarobot = {
      source = "datarobot-community/datarobot"
    }
  }
}

provider "datarobot" {
  # export DATAROBOT_API_TOKEN="your-api-key"
  endpoint = "https://app.datarobot.com/api/v2"
}

resource "datarobot_artifact" "app" {
  name = "${var.workload_name}-artifact"
  type = "service"

  spec = {
    container_groups = [
      {
        containers = [
          {
            name       = "main"
            image_uri  = var.container_image
            port       = 8080
            primary    = true
            entrypoint = ["/whoami", "--port", "8080"]
          }
        ]
      }
    ]
  }
}

resource "datarobot_workload" "api" {
  name        = var.workload_name
  description = "Workload replacement workflow example"
  artifact_id = datarobot_artifact.app.artifact_id

  runtime = {
    container_groups = [
      {
        replica_count    = var.replica_count
        resource_bundles = ["cpu.small"]
      }
    ]
    replacement_policy = {
      warmup_minutes           = 5
      keep_old_version_minutes = 10
    }
  }
}

output "workload_id" {
  value       = datarobot_workload.api.id
  description = "Workload ID — stable across artifact and runtime updates (in-place replacement)"
}

output "workload_endpoint" {
  value       = datarobot_workload.api.endpoint
  description = "Inference endpoint — stable across in-place replacements"
}

output "artifact_version_id" {
  value       = datarobot_artifact.app.artifact_id
  description = "Current artifact version ID deployed on the workload"
}
