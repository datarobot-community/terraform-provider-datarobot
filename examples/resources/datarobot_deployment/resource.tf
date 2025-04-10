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
  challenger_models_settings            = {}
  challenger_replay_settings            = {}
  segment_analysis_settings             = {}
  bias_and_fairness_settings            = {}
  predictions_by_forecast_date_settings = {}
  drift_tracking_settings               = {}
  association_id_settings               = {}
  predictions_data_collection_settings  = {}
  prediction_warning_settings           = {}
  prediction_intervals_settings         = {}
  predictions_settings                  = {}
  feature_cache_settings                = {}
  health_settings                       = {}
  runtime_parameter_values = [
    {
      key   = "EXAMPLE_PARAM",
      type  = "string",
      value = "val",
    },
  ]
  retraining_settings = {}
}

output "datarobot_deployment_id" {
  value       = datarobot_deployment.example.id
  description = "The id for the example deployment"
}
