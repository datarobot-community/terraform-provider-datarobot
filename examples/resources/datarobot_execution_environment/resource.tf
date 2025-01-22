resource "datarobot_execution_environment" "example" {
  name                 = "Example Execution Environment"
  programming_language = "python"

  # Optional
  description         = "Example Execution Environment Description"
  docker_context_path = "docker_context.zip"
  docker_image        = "docker_image.tar"
  use_cases           = ["customModel"]
}

output "datarobot_execution_environment_id" {
  value       = datarobot_execution_environment.example.id
  description = "The id for the example execution environment"
}

output "datarobot_execution_environment_version_id" {
  value       = datarobot_execution_environment.example.version_id
  description = "The version id for the example execution environment"
}