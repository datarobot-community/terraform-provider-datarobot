terraform {
  required_providers {
    datarobot = {
      source = "datarobot-community/datarobot"
    }
  }
}

provider "datarobot" {
  # export DATAROBOT_API_TOKEN="the API Key value here"
}

resource "datarobot_registered_model_from_leaderboard" "example" {
  name                 = "example registered model from leaderboard"
  model_id             = "673b75ec97f1021bbfb61d3b"
  prediction_threshold = 0.7
  # version_name = "example version name is something else"
}
