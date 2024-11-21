resource "datarobot_dataset_from_datasource" "example" {
  datasource_id = datarobot_datasource.example.id
  credential_id = datarobot_credential.example.id

  # Optional
  do_snapshot                  = false
  persist_data_after_ingestion = false
  use_kerberos                 = true
  categories                   = ["TRAINING"]
  use_case_ids                 = [datarobot_use_case.example.id]
}

output "example_id" {
  value       = datarobot_dataset_from_datasource.example.id
  description = "The id for the example dataset"
}
