resource "datarobot_application_source" "example" {
  name                = "example application source"
  base_environment_id = "6542cd582a9d3d51bf4ac71e"
  files = [
    ["start-app.sh"],
    ["streamlit-app.py"],
  ]
}

output "datarobot_application_source_id" {
  value       = datarobot_application_source.example.id
  description = "The ID for the example application source"
}

output "datarobot_application_source_version_id" {
  value       = datarobot_application_source.example.version_id
  description = "The version ID for the example application source"
}