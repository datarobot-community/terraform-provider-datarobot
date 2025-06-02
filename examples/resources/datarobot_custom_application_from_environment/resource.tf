resource "datarobot_custom_application_from_environment" "example" {
  name           = "example-custom-app-from-environment"
  environment_id = datarobot_execution_environment.example.id

  # optional settings
  external_access_enabled = true
  external_access_recipients = [
    "recipient@example.com",
  ]
  allow_auto_stopping = false

  resources {
    replicas                          = 1
    resource_label                    = "cpu.small"
    session_affinity                  = false
    service_web_requests_on_root_path = true
  }
}

output "datarobot_custom_application_id" {
  value       = datarobot_custom_application_from_environment.example.id
  description = "The ID of the example custom application"
}

output "datarobot_custom_application_url" {
  value       = datarobot_custom_application_from_environment.example.application_url
  description = "The URL for the example custom application"
}
