variable "workload_name" {
  type        = string
  description = "Name prefix for the artifacts and workloads"
  default     = "workload-replacement-example"
}

variable "container_image" {
  type        = string
  description = "Container image for both artifacts. Change this and re-apply to roll both workloads out in place, at the same time."
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

# Two independent artifact and workload pairs. Nothing links one pair to the
# other, so Terraform updates both artifacts, then rolls both workloads out, in
# parallel: the apply takes about as long as a single rollout.
locals {
  workloads = toset(["alpha", "beta"])
}

resource "datarobot_artifact" "app" {
  for_each = local.workloads

  name = "${var.workload_name}-${each.key}-artifact"
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
  for_each = local.workloads

  name        = "${var.workload_name}-${each.key}"
  description = "Workload replacement workflow example (${each.key})"
  artifact_id = datarobot_artifact.app[each.key].artifact_id

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

output "workload_ids" {
  value       = { for key, workload in datarobot_workload.api : key => workload.id }
  description = "Workload IDs, stable across artifact and runtime updates (in-place replacement)"
}

output "workload_endpoints" {
  value       = { for key, workload in datarobot_workload.api : key => workload.endpoint }
  description = "Inference endpoints, stable across in-place replacements"
}

output "artifact_version_ids" {
  value       = { for key, artifact in datarobot_artifact.app : key => artifact.artifact_id }
  description = "Current artifact version ID deployed on each workload"
}
