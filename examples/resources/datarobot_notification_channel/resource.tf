resource "datarobot_notification_channel" "example" {
  name                = "example notification channel"
  channel_type        = "DataRobotUser"
  related_entity_id   = datarobot_deployment.example.id
  related_entity_type = "deployment"

  # Optional
  content_type = "application/json"
  custom_headers = [
    {
      name  = "header1"
      value = "value1"
    }
  ]
  dr_entities = [
    {
      id   = "11111111111111"
      name = "example user"
    }
  ]
  language_code     = "en"
  email_address     = "example@datarobot.com"
  payload_url       = "https://example.com"
  secret_token      = "example_secret_token"
  validate_ssl      = true
  verification_code = "11111"
}

output "datarobot_notification_policy_id" {
  value       = datarobot_notification_policy.example.id
  description = "The id for the example notification policy"
}
