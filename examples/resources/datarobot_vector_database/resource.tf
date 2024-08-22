resource "datarobot_vector_database" "example" {
  name        = "An example vector database"
  use_case_id = datarobot_use_case.example.id
  dataset_id  = datarobot_dataset_from_file.example.id
  chunking_parameters = {
    chunk_overlap_percentage = 0
    chunk_size               = 256
    chunking_method          = "recursive"
    embedding_model          = "jinaai/jina-embedding-t-en-v1"
  }
}

output "example_id" {
  value       = datarobot_vector_database.example.id
  description = "The id for the example vector database"
}
