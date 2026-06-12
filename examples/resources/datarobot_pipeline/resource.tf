resource "datarobot_pipeline" "example" {
  source_file = "pipeline.py"
  description = "My data pipeline"
  mode        = "draft"
}

# Lock the pipeline once the source is stable.
resource "datarobot_pipeline" "locked_example" {
  source_file = "pipeline.py"
  mode        = "locked"
}

output "example_id" {
  value       = datarobot_pipeline.example.id
  description = "The ID of the pipeline."
}

output "example_task_names" {
  value       = datarobot_pipeline.example.task_names
  description = "Task names extracted from the @task-decorated functions."
}
