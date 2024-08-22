resource "datarobot_deployment" "example" {
  label                       = "An example deployment"
  prediction_environment_id   = datarobot_prediction_environment.example.id
  registered_model_version_id = datarobot_registered_model.example.version_id

  # Optional settings
  settings = {
    prediction_row_storage = true
    association_id = {
      auto_generate_id = true
      feature_name     = "example_feature"
    }
  }
}

output "datarobot_deployment_id" {
  value       = datarobot_deployment.example.id
  description = "The id for the example deployment"
}