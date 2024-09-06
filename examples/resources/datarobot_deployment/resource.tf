resource "datarobot_custom_model" "example" {
  name                  = "Example Custom Model"
  description           = "Description for the example custom model"
  target_type           = "Binary"
  target_name           = "my_label"
  base_environment_name = "[GenAI] Python 3.11 with Moderations"
  files                 = ["example.py"]
}

resource "datarobot_registered_model" "example" {
  custom_model_version_id = datarobot_custom_model.example.version_id
  name                    = "Example Registered Model"
  description             = "Description for the example registered model"
}

resource "datarobot_prediction_environment" "example" {
  name        = "Example Prediction Environment"
  description = "Description for the example prediction environment"
  platform    = "datarobotServerless"
}

resource "datarobot_deployment" "example" {
  label                       = "An example deployment"
  prediction_environment_id   = datarobot_prediction_environment.example.id
  registered_model_version_id = datarobot_registered_model.example.version_id

  # Optional settings
  # settings = {
  #   prediction_row_storage = true
  # }
}

output "datarobot_deployment_id" {
  value       = datarobot_deployment.example.id
  description = "The id for the example deployment"
}