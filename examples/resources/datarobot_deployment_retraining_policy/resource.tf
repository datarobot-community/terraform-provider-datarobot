resource "datarobot_use_case" "example" {
  name = "Example use case"
}

resource "datarobot_custom_model" "example" {
  name                = "Example Custom Model"
  target_type         = "Binary"
  target_name         = "my_label"
  base_environment_id = "65f9b27eab986d30d4c64268"
  files               = ["example.py"]
}

resource "datarobot_registered_model" "example" {
  custom_model_version_id = datarobot_custom_model.example.version_id
  name                    = "Example Registered Model"
}

resource "datarobot_prediction_environment" "example" {
  name     = "Example Prediction Environment"
  platform = "datarobotServerless"
}

resource "datarobot_deployment" "example" {
  label                       = "An example deployment"
  prediction_environment_id   = datarobot_prediction_environment.example.id
  registered_model_version_id = datarobot_registered_model.example.version_id
}

resource "datarobot_custom_job" "example" {
  name           = "Example Custom Job"
  job_type       = "retraining"
  environment_id = "65f9b27eab986d30d4c64268"
  files = [
    "file1.py",
    "file2.py",
  ]
}

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
  use_case_id         = datarobot_use_case.example.id
}


output "datarobot_deployment_retraining_policy_id" {
  value       = datarobot_deployment_retraining_policy.example.id
  description = "The id for the example deployment retraining policy"
}
