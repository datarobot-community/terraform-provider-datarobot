resource "datarobot_dataset_from_url" "example" {
  url          = "[URL to upload from]"
  use_case_ids = [datarobot_use_case.example.id]

  # Optional
  name = "Example Dataset"
}

output "example_id" {
  value       = datarobot_dataset_from_url.example.id
  description = "The id for the example dataset"
}
