resource "datarobot_application_source" "example" {
  name        = "example application source"
  local_files = ["start-app.sh", "streamlit-app.py"]
}

output "datarobot_application_source_id" {
  value       = datarobot_application_source.example.id
  description = "The ID for the example application source"
}

output "datarobot_application_source_version_id" {
  value       = datarobot_application_source.example.version_id
  description = "The version ID for the example application source"
}