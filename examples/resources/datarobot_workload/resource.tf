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
