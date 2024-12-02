resource "datarobot_batch_prediction_job_definition" "example" {
  name          = "Example Batch Prediction Job Definition"
  deployment_id = datarobot_deployment.batch_prediction_job_definition.id
  intake_settings = {
    type          = "s3"
    url           = "s3://datarobot-public-datasets-redistributable/1k_diabetes_simplified_features.csv"
    credential_id = "${datarobot_basic_credential.batch_prediction_job_definition.id}"
  }

  # Optional parameters
  output_settings = {
    type          = "s3"
    url           = "s3://my-test-bucket/predictions.csv"
    credential_id = "${datarobot_basic_credential.batch_prediction_job_definition.id}"
  }
  csv_settings = {
    delimiter = "."
    quotechar = "'"
    encoding  = "utf-8"
  }
  num_concurrent            = 1
  chunk_size                = 10
  max_explanations          = 5
  threshold_high            = 0.8
  threshold_low             = 0.2
  prediction_threshold      = 0.5
  include_prediction_status = true
  skip_drift_tracking       = true
  passthrough_columns_set   = "all"
  abort_on_error            = false
  include_probabilities     = true
  column_names_remapping = {
    "col1" = "newCol1"
  }
  schedule = {
    minute       = ["15", "45"]
    hour         = ["*"]
    month        = ["*"]
    day_of_month = ["*"]
    day_of_week  = ["*"]
  }
}

output "example_id" {
  value       = datarobot_batch_prediction_job_definition.example.id
  description = "The id for the example batch prediction job definition"
}
