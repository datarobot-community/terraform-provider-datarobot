resource "datarobot_registered_model_from_leaderboard" "example" {
  name     = "example registered model from leaderboard"
  model_id = "111111111111"

  # Optional
  description                      = "example description"
  version_name                     = "example version name"
  prediction_threshold             = 0.5
  compute_all_ts_intervals         = true
  distribution_prediction_model_id = "222222222222"
  use_case_ids                     = [datarobot_use_case.example.id]
}

output "datarobot_registered_model_from_leaderboard_id" {
  value       = datarobot_registered_model_from_leaderboard.example.id
  description = "The id for the example registered model from leaderboard"
}

output "datarobot_registered_model_from_leaderboard_version_id" {
  value       = datarobot_registered_model_from_leaderboard.example.version_id
  description = "The version id for the example registered model from leaderboard"
}
