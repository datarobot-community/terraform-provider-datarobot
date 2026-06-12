resource "datarobot_pipeline_environment" "example" {
  name        = "my-pipeline-env"
  description = "Python environment for my pipeline"
  packages    = ["numpy==1.26.0", "pandas>=2.0,<3.0"]
}

output "example_id" {
  value       = datarobot_pipeline_environment.example.id
  description = "The ID of the pipeline environment."
}
