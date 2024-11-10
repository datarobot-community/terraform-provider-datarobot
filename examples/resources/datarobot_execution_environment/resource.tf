resource "datarobot_execution_environment" "example" {
  name                 = "Example Execution Environment"
  description          = "Example Execution Environment Description"
  programming_language = "python"
  use_cases            = ["customModel"]
  docker_context_path  = "docker_context.zip"
}
