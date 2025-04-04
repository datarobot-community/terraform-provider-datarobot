resource "datarobot_remote_repository" "example" {
  name        = "Datarobot User Models"
  description = "GitHub repository with Datarobot user models"
  location    = "https://github.com/datarobot/datarobot-user-models"
  source_type = "github"

  # set the credential id for private repositories
  # credential_id = datarobot_api_token_credential.example.id
}

resource "datarobot_custom_model" "example" {
  name        = "Example from GitHub"
  description = "An example custom model from GitHub repository"
  files = [
    "file1.py",
    "file2.py",
  ]
  target_type         = "Binary"
  target_name         = "my_label"
  base_environment_id = "65f9b27eab986d30d4c64268"

  # Optional
  # source_remote_repositories = [
  #   {
  #     id  = datarobot_remote_repository.example.id
  #     ref = "master"
  #     source_paths = [
  #       "model_templates/python3_dummy_binary",
  #     ]
  #   }
  # ]
  # guard_configurations = [
  #   {
  #     template_name = "Rouge 1"
  #     name          = "Rouge 1 response"
  #     stages        = ["response"]
  #     intervention = {
  #       action  = "block"
  #       message = "response has been blocked by Rogue 1 guard"
  #       condition = jsonencode({
  #         "comparand": 0.5, 
  #         "comparator": "greaterThan"
  #       })
  #     }
  #   },
  # ]
  # overall_moderation_configuration = {
  #   timeout_sec    = 120
  #   timeout_action = "score"
  # }
  # memory_mb      = 512
  # replicas       = 2
  # network_access = "NONE"
}

output "example_id" {
  value       = datarobot_custom_model.example.id
  description = "The id for the example custom model"
}
