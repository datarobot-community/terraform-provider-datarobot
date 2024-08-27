resource "datarobot_custom_model" "example" {
  name                  = "Example Custom Model"
  description           = "Description for the example custom model"
  target_type           = "Binary"
  target                = "my_label"
  base_environment_name = "[GenAI] Python 3.11 with Moderations"
  local_files           = ["example.py"]
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
output "datarobot_registered_model_id" {
  value       = datarobot_registered_model.example.id
  description = "The id for the example registered model"
}

output "datarobot_registered_model_version_id" {
  value       = datarobot_registered_model.example.version_id
  description = "The version id for the example registered model"
}