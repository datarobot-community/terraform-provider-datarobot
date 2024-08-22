resource "datarobot_prediction_environment" "toxicity_prediction_environment" {
  name        = "Example Prediction Environment"
  description = "Description for the example prediction environment"
  platform    = "aws"
}
