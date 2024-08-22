resource "datarobot_registered_model" "example" {
  name                    = "Example Registered Model"
  description             = "Description for the example registered model"
  custom_model_version_id = datarobot_custom_model.example.version_id
}

output "datarobot_registered_model_id" {
  value       = datarobot_registered_model.example.id
  description = "The id for the example registered model"
}

output "datarobot_registered_model_version_id" {
  value       = datarobot_registered_model.example.version_id
  description = "The version id for the example registered model"
}