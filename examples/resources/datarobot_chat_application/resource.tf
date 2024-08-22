resource "datarobot_chat_application" "example" {
  name          = "An example chat application"
  deployment_id = datarobot_deployment.example.id
}

output "datarobot_chat_application_url" {
  value       = datarobot_chat_application.example.application_url
  description = "The URL for the example chat application"
}