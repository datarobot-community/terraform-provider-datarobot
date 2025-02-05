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
  time_step = "hours"
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
