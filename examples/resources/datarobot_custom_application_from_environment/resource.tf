resource "datarobot_custom_application_from_environment" "example" {
  name           = "example custom application from environment"
  environment_id = "6542cd582a9d3d51bf4ac71e"

  # optional settings
  external_access_enabled = true
  external_access_recipients = [
    "recipient@example.com",
  ]
}

output "datarobot_custom_application_id" {
  value       = datarobot_custom_application_from_environment.example.id
  description = "The ID of the example custom application"
}

output "datarobot_custom_application_url" {
  value       = datarobot_custom_application_from_environment.example.application_url
  description = "The URL for the example custom application"
}
