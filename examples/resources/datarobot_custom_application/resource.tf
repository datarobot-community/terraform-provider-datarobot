resource "datarobot_application_source" "example" {
  files = [
    ["start-app.sh"],
    ["streamlit-app.py"]
  ]
}

resource "datarobot_custom_application" "example" {
  name              = "example-custom-application"
  source_version_id = datarobot_application_source.example.version_id

  # optional settings
  external_access_enabled = true
  external_access_recipients = [
    "recipient@example.com",
  ]
  allow_auto_stopping = false

  resources {
    replicas                          = 2
    resource_label                    = "cpu.medium"
    session_affinity                  = true
    service_web_requests_on_root_path = false
  }
}

output "datarobot_custom_application_id" {
  value       = datarobot_custom_application.example.id
  description = "The ID of the example custom application"
}

output "datarobot_custom_application_source_id" {
  value       = datarobot_custom_application.example.source_id
  description = "The ID of the application source for the example custom application"
}

output "datarobot_custom_application_source_version_id" {
  value       = datarobot_custom_application.example.source_version_id
  description = "The version ID of the application source for the example custom application"
}

output "datarobot_custom_application_url" {
  value       = datarobot_custom_application.example.application_url
  description = "The URL for the example custom application"
}