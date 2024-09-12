resource "datarobot_dataset_from_file" "example" {
  file_path    = "[Path to file to upload]"
  use_case_ids = [datarobot_use_case.example.id]

  # Optional
  name = "Example Dataset"
}

output "example_id" {
  value       = datarobot_dataset_from_file.example.id
  description = "The id for the example dataset"
}
