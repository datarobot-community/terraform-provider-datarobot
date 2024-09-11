resource "datarobot_prediction_environment" "example" {
  name        = "Example Prediction Environment"
  description = "Description for the example prediction environment"
  platform    = "datarobotServerless"

  # Optional
  batch_jobs_max_concurrent = 20
  batch_jobs_priority       = "high"
  supported_model_formats   = ["datarobot", "customModel"]
  managed_by                = "selfManaged"
  credential_id             = "<credential_id>"
  datastore_id              = "<datastore_id>"
}
