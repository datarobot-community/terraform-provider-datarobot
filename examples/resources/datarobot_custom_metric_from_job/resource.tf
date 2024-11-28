resource "datarobot_custom_metric_from_job" "example" {
  name          = "Example Custom Metric From Job"
  deployment_id = datarobot_deployment.example.id
  custom_job_id = datarobot_custom_metric_job.example.id
}

output "example_id" {
  value       = datarobot_custom_metric_from_job.example.id
  description = "The id for the example custom metric"
}
