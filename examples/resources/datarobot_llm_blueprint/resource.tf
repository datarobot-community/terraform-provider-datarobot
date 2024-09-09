resource "datarobot_use_case" "example" {
  name = "Example use case"
}

resource "datarobot_playground" "example" {
  name        = "An example playground"
  description = "Description for the example playground"
  use_case_id = datarobot_use_case.example.id
}

resource "datarobot_llm_blueprint" "example" {
  name          = "An example LLM blueprint"
  description   = "Description for the example LLM blueprint"
  playground_id = datarobot_playground.example.id
  llm_id        = "azure-openai-gpt-3.5-turbo"
  prompt_type   = "ONE_TIME_PROMPT"

  # Optional
  # llm_settings {
  #   max_completion_length = 1000
  #   temperature           = 0.5
  #   top_p                 = 0.9
  #   system_prompt         = "My Prompt:"
  # }
  # vector_database_settings = {
  #   max_documents_retrieved_per_prompt = 5
  #   max_tokens = 1000
  # }
}

output "example_id" {
  value       = datarobot_llm_blueprint.example.id
  description = "The id for the example LLM blueprint"
}
