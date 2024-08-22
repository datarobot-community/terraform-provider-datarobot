resource "datarobot_llm_blueprint" "example" {
  name               = "An example LLM blueprint"
  description        = "Description for the example LLM blueprint"
  playground_id      = datarobot_playground.example.id
  vector_database_id = datarobot_vector_database.example.id
  llm_id             = "azure-openai-gpt-3.5-turbo"
}

output "example_id" {
  value       = datarobot_llm_blueprint.example.id
  description = "The id for the example LLM blueprint"
}
