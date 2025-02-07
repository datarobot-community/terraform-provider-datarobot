resource "datarobot_custom_model_llm_validation" "example" {
  deployment_id      = datarobot_deployment.example.id
  prompt_column_name = "promptText"
  target_column_name = "resultText"

  # Optional
  prediction_timeout = 100
  use_case_id        = datarobot.use_case.example.id
}

output "example_id" {
  value       = datarobot_custom_model_llm_validation.example.id
  description = "The id for the example custom model LLM validation"
}
