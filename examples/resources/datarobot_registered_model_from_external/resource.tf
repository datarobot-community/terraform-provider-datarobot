resource "datarobot_registered_model_from_external" "example" {
  name = "example registered model from external"
  target = {
    type                 = "Binary"
    name                 = "readmitted"
    class_names          = ["True", "False"]
    prediction_threshold = 0.5
  }

  # Optional
  description  = "example description"
  version_name = "example version name"
  model_description = {
    model_name             = "example external model"
    description            = "example version description"
    location               = "s3://example-bucket/models/readmission"
    build_environment_type = "Python"
  }
  datasets = {
    training_data_catalog_id = datarobot_dataset_from_file.training.id
    holdout_data_catalog_id  = datarobot_dataset_from_file.holdout.id
  }
  tags = [
    {
      name  = "team"
      value = "example"
    },
  ]
  use_case_ids = [datarobot_use_case.example.id]
}

output "datarobot_registered_model_from_external_id" {
  value       = datarobot_registered_model_from_external.example.id
  description = "The id for the example registered model from external"
}

output "datarobot_registered_model_from_external_version_id" {
  value       = datarobot_registered_model_from_external.example.version_id
  description = "The version id for the example registered model from external"
}
