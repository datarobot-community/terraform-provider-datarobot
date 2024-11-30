resource "datarobot_deployment_retraining_policy" "example" {
  deployment_id = datarobot_deployment.example.id
  name          = "Example Deployment Retraining Policy"
  description   = "Example Description"

  # Optional
  action                   = "create_model_package"
  model_selection_strategy = "custom_job"
  feature_list_strategy    = "informative_features"
  project_options_strategy = "custom"
  trigger = {
    custom_job_id = datarobot_custom_job.example.id
  }
  autopilot_options   = {}
  project_options     = {}
  time_series_options = {}
}


output "datarobot_deployment_retraining_policy_id" {
  value       = datarobot_deployment_retraining_policy.example.id
  description = "The id for the example deployment retraining policy"
}
