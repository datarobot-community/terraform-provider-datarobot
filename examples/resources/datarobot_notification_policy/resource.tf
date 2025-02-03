resource "datarobot_notification_policy" "example" {
  name                = "example notification policy"
  channel_id          = "11111111111111"
  channel_scope       = "template"
  event_group         = "model_deployments.all"
  related_entity_id   = datarobot_deployment.example.id
  related_entity_type = "deployment"

  # Optional
  event_type        = "model_deployments.accuracy_green"
  maximal_frequency = "PT1H"
}

output "datarobot_notification_policy_id" {
  value       = datarobot_notification_policy.example.id
  description = "The id for the example notification policy"
}
