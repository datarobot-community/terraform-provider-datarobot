resource "datarobot_prediction_environment" "example" {
  name        = "Example Prediction Environment"
  description = "Description for the example prediction environment"
  platform    = "datarobotServerless"
}
