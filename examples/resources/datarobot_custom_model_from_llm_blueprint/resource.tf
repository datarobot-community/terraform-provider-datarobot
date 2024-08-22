resource "datarobot_custom_model" "example_from_llm_blueprint" {
  name                    = "Example Custom Model from LLM Blueprint"
  description             = "example description"
  source_llm_blueprint_id = datarobot_llm_blueprint.example.id
  runtime_parameters = [
    {
      key   = "OPENAI_API_BASE",
      type  = "string",
      value = "https://datarobot-genai-enablement.openai.azure.com/"
    },
    {
      key   = "OPENAI_API_KEY",
      type  = "credential",
      value = datarobot_api_token_credential.example.id
    }
  ]
}

output "example_id" {
  value       = datarobot_custom_model.example.id
  description = "The id for the example custom model"
}
