variable "use_case_name" {
  type = string
}

variable "google_cloud_credential_source_file" {
  type = string
}

variable "toxicity_classifier_registered_version_id" {
  type = string
}

terraform {
  required_providers {
    datarobot = {
      source  = "datarobot/datarobot"
      version = "0.0.10"
    }
  }
}

provider "datarobot" {
  # export DATAROBOT_API_KEY="the API Key value here"
}

data "datarobot_global_model" "toxicity_classifier" {
  name = "[Hugging Face] Toxicity Classifier"
}

resource "datarobot_use_case" "example" {
  name        = var.use_case_name
  description = "Docsassist UC"
}

resource "datarobot_dataset_from_file" "example" {
  use_case_id = datarobot_use_case.example.id
  source_file = "datarobot_english_documentation_docsassist.zip"
}

resource "datarobot_playground" "example" {
  use_case_id = datarobot_use_case.example.id
  name        = datarobot_use_case.example.name
  description = datarobot_use_case.example.description
}

resource "datarobot_vector_database" "example" {
  name                = datarobot_use_case.example.name
  use_case_id         = datarobot_use_case.example.id
  dataset_id          = datarobot_dataset_from_file.example.id
  chunking_parameters = {}
}

resource "datarobot_llm_blueprint" "example" {
  name               = datarobot_use_case.example.name
  description        = datarobot_use_case.example.description
  playground_id      = datarobot_playground.example.id
  vector_database_id = datarobot_vector_database.example.id
  llm_id             = "google-bison"
}

resource "datarobot_google_cloud_credential" "example" {
  name        = "Google Cloud Service Account docsassist"
  source_file = var.google_cloud_credential_source_file
}


resource "datarobot_prediction_environment" "example" {
  name        = datarobot_use_case.example.name
  description = datarobot_use_case.example.description
  platform    = "datarobotServerless"
}

resource "datarobot_deployment" "toxicity_global" {
  label                       = "Toxicity Classifier"
  registered_model_version_id = var.toxicity_classifier_registered_version_id
  prediction_environment_id   = datarobot_prediction_environment.example.id
  settings = {
    predictions_settings = {
      real_time    = "true"
      max_computes = 1
      min_computes = 0
    }
  }
}

resource "datarobot_custom_model" "example" {
  name                    = datarobot_use_case.example.name
  description             = datarobot_use_case.example.description
  source_llm_blueprint_id = datarobot_llm_blueprint.example.id
  runtime_parameters = [
    {
      key   = "GOOGLE_SERVICE_ACCOUNT",
      type  = "credential",
      value = datarobot_google_cloud_credential.example.id
    }
  ]
  guard_configurations = [
    {
      template_name = "Toxicity"
      name          = "Global Toxicity Classifier"
      stages        = ["prompt"]
      intervention = {
        action  = "block"
        message = "I have detected this question may violate my acceptable use guidelines and am unable to answer. [Toxicity]"
        condition = {
          comparand  = 0.5
          comparator = "greaterThan"
        }
      }
      deployment_id = datarobot_deployment.toxicity_global.id
    }
  ]
}

resource "datarobot_registered_model" "example" {
  custom_model_version_id = datarobot_custom_model.example.version_id
  name                    = datarobot_use_case.example.name
  description             = datarobot_use_case.example.description
}

resource "datarobot_deployment" "example" {
  label                       = datarobot_use_case.example.name
  prediction_environment_id   = datarobot_prediction_environment.example.id
  registered_model_version_id = datarobot_registered_model.example.version_id
  settings = {
    prediction_row_storage = true
  }
}

resource "datarobot_chat_application" "example" {
  name          = datarobot_use_case.example.name
  deployment_id = datarobot_deployment.example.id
}

output "datarobot_chat_application_url" {
  value       = datarobot_chat_application.example.application_url
  description = "The URL for the example chat application"
}
