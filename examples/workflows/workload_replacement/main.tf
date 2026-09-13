variable "workload_name_prefix" {
  type        = string
  description = "Prefix for artifact and workload resource names"
  default     = "workload-replacement-example"
}

variable "replica_count" {
  type        = number
  description = "Number of workload replicas. Changing only this triggers a runtime settings update (same artifact)."
  default     = 2
}

locals {
  # Two independent artifact + workload pairs. Terraform updates them in parallel
  # when container_image changes (default -parallelism=10).
  workloads = {
    alpha = "alpha"
    beta  = "beta"
  }
}

resource "datarobot_artifact" "app" {
  for_each = local.workloads

  name = "${var.workload_name_prefix}-${each.key}-artifact"
  type = "service"

  spec = {
    container_groups = [
      {
        containers = [
          {
            name       = "main"
            image_uri  = "containous/whoami:latest" // -> change this to "containous/whoami:v1.5.0" to trigger replacement
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

  name        = "${var.workload_name_prefix}-${each.key}"
  description = "Parallel workload replacement example (${each.key})"
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
  value = {
    for key, workload in datarobot_workload.api : key => workload.id
  }
  description = "Workload IDs — stable across artifact and runtime updates (in-place replacement)"
}

output "workload_endpoints" {
  value = {
    for key, workload in datarobot_workload.api : key => workload.endpoint
  }
  description = "Inference endpoints — stable across in-place replacements"
}

output "artifact_version_ids" {
  value = {
    for key, artifact in datarobot_artifact.app : key => artifact.artifact_id
  }
  description = "Current artifact version IDs deployed on each workload"
}
