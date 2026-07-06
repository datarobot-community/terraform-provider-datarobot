data "datarobot_artifact" "existing" {
  artifact_id = var.artifact_id
}

output "artifact_name" {
  value = data.datarobot_artifact.existing.name
}

output "artifact_status" {
  value = data.datarobot_artifact.existing.status
}
