variable "use_case_name" {
  type = string
}

variable "google_cloud_credential_source_file" {
  type = string
}

terraform {
  required_providers {
    datarobot = {
      source = "datarobot-community/datarobot"
    }
  }
}

provider "datarobot" {
  # There are two options to set your API Key using either an environment variable or set in the code here.
  # Option one is to use this line in your terminal `export DATAROBOT_API_TOKEN="the API Key value here"`
  # Option two is to uncomment the line below and enter your API Key
  # apikey = "<REPLACE_WITH_YOUR_API_KEY>"

  # This is the default endpoint but you may find value in altering it.
  # For example either our EU or JP endpoints like: https://app.eu.datarobot.com/api/v2
  endpoint = "https://app.datarobot.com/api/v2"
}

resource "datarobot_use_case" "example" {
  name        = var.use_case_name
  description = "Low Code RAG Example"
}

resource "datarobot_dataset_from_file" "example" {
  use_case_ids = [datarobot_use_case.example.id]
  file_path    = "datarobot_english_documentation_docsassist.zip"
}

resource "datarobot_playground" "example" {
  use_case_id = datarobot_use_case.example.id
  name        = datarobot_use_case.example.name
  description = datarobot_use_case.example.description
}

resource "datarobot_vector_database" "example" {
  name        = datarobot_use_case.example.name
  use_case_id = datarobot_use_case.example.id
  dataset_id  = datarobot_dataset_from_file.example.id
}

resource "datarobot_llm_blueprint" "example" {
  name               = datarobot_use_case.example.name
  description        = datarobot_use_case.example.description
  playground_id      = datarobot_playground.example.id
  vector_database_id = datarobot_vector_database.example.id
  llm_id             = "google-bison"
}

resource "datarobot_google_cloud_credential" "example" {
  name         = "${var.use_case_name} Google Cloud Service Account"
  gcp_key_file = var.google_cloud_credential_source_file
}

resource "datarobot_custom_model" "example" {
  name                    = datarobot_use_case.example.name
  description             = datarobot_use_case.example.description
  source_llm_blueprint_id = datarobot_llm_blueprint.example.id
  runtime_parameter_values = [
    {
      key   = "GOOGLE_SERVICE_ACCOUNT",
      type  = "credential",
      value = datarobot_google_cloud_credential.example.id
    }
  ]
}

resource "datarobot_registered_model" "example" {
  custom_model_version_id = datarobot_custom_model.example.version_id
  name                    = datarobot_use_case.example.name
  description             = datarobot_use_case.example.description
}

resource "datarobot_prediction_environment" "example" {
  name        = datarobot_use_case.example.name
  description = datarobot_use_case.example.description
  platform    = "datarobotServerless"
}

resource "datarobot_deployment" "example" {
  label                       = datarobot_use_case.example.name
  prediction_environment_id   = datarobot_prediction_environment.example.id
  registered_model_version_id = datarobot_registered_model.example.version_id
}

resource "datarobot_qa_application" "example" {
  name          = datarobot_use_case.example.name
  deployment_id = datarobot_deployment.example.id
}

output "datarobot_qa_application_url" {
  value       = datarobot_qa_application.example.application_url
  description = "The URL for the example Q&A application"
}
