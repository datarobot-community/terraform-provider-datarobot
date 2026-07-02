data "datarobot_artifacts" "all" {
  # Optional filters:
  # status = "locked"
  # limit  = 50
}

output "artifact_count" {
  value = length(data.datarobot_artifacts.all.artifacts)
}

output "artifact_names" {
  value = [for a in data.datarobot_artifacts.all.artifacts : a.name]
}
