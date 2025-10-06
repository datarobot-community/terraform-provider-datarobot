resource "datarobot_custom_model" "example" {
  name                = "Example Custom Model"
  description         = "Description for the example custom model"
  target_type         = "Binary"
  target_name         = "my_label"
  base_environment_id = "65f9b27eab986d30d4c64268"
  files               = ["example.py"]
}

resource "datarobot_registered_model" "example" {
  custom_model_version_id = datarobot_custom_model.example.version_id
  name                    = "Example Registered Model"
  description             = "Description for the example registered model"

  tags {
    name  = "ab-test"
    value = "a1"
  }

  tags {
    name  = "team"
    value = "marketing"
  }
}

output "datarobot_registered_model_id" {
  value       = datarobot_registered_model.example.id
  description = "The id for the example registered model"
}

output "datarobot_registered_model_version_id" {
  value       = datarobot_registered_model.example.version_id
  description = "The version id for the example registered model"
}