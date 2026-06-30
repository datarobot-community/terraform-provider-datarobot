resource "datarobot_artifact" "example" {
  name        = "example-service"
  description = "Artifact for a containerized workload"
  status = "draft"  # optional: "draft" (mutable) or "locked" (default)

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

output "example_artifact_id" {
  value       = datarobot_artifact.example.artifact_id
  description = "Current artifact ID (stable across draft updates and lock; new ID on locked version changes)"
}
