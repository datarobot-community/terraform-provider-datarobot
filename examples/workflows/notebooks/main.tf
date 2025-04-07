variable "use_case_name" {
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
  name = var.use_case_name
}

resource "datarobot_notebook" "example" {
  file_path   = "./test_notebook.ipynb"
  use_case_id = datarobot_use_case.example.id
}

output "datarobot_notebook_url" {
  value       = datarobot_notebook.example.url
  description = "The URL for the Notebook"
}
