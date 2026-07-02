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

resource "datarobot_custom_metric" "example" {
  deployment_id     = datarobot_deployment.example.id
  name              = "example custom metric"
  description       = "example description"
  units             = "dollars"
  directionality    = "higherIsBetter"
  type              = "sum"
  baseline_value    = 0.5
  is_model_specific = true
  is_geospatial     = false

  # Optional
  time_step = "hour"
  timestamp = {
    column_name = "timestamp_column"
    time_format = "%Y-%m-%dT%H:%M:%SZ"
  }
  value = {
    column_name = "value_column"
  }
  batch = {
    column_name = "batch_column"
  }
  sample_count = {
    column_name = "sample_count_column"
  }
  association_id = {
    column_name = "association_id_column"
  }
}

output "example_id" {
  value       = datarobot_custom_metric.example.id
  description = "The id for the example custom metric"
}
