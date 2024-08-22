resource "datarobot_dataset_from_file" "example" {
  source_file = "[Path to file to upload]"
  use_case_id = datarobot_use_case.example.id
}

output "example_id" {
  value       = datarobot_dataset_from_file.example.id
  description = "The id for the example dataset"
}
