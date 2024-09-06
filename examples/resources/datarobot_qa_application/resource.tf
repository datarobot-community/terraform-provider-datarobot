resource "datarobot_custom_model" "example" {
  name                  = "Example Custom Model"
  description           = "Description for the example custom model"
  target_type           = "Binary"
  target_name           = "my_label"
  base_environment_name = "[GenAI] Python 3.11 with Moderations"
  files                 = ["example.py"]
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
}

resource "datarobot_qa_application" "example" {
  name                    = "An example Q&A application"
  deployment_id           = datarobot_deployment.example.id
  external_access_enabled = true
  external_access_recipients = [
    "recipient@example.com",
  ]
}

output "datarobot_qa_application_id" {
  value       = datarobot_qa_application.example.id
  description = "The ID of the example Q&A application"
}

output "datarobot_qa_application_source_id" {
  value       = datarobot_qa_application.example.source_id
  description = "The ID of the application source for the example Q&A application"
}

output "datarobot_qa_application_source_version_id" {
  value       = datarobot_qa_application.example.source_version_id
  description = "The version ID of the application source for the example Q&A application"
}

output "datarobot_qa_application_url" {
  value       = datarobot_qa_application.example.application_url
  description = "The URL for the example Q&A application"
}