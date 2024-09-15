resource "datarobot_use_case" "example" {
  name        = "An example use case"
  description = "Description for the example use case"
}

resource "datarobot_dataset_from_file" "example" {
  file_path    = "[Path to file to upload]"
  use_case_ids = [datarobot_use_case.example.id]
}

resource "datarobot_vector_database" "example" {
  name        = "An example vector database"
  use_case_id = datarobot_use_case.example.id
  dataset_id  = datarobot_dataset_from_file.example.id

  # Optional
  # chunking_parameters = {
  #   chunk_overlap_percentage = 0
  #   chunk_size               = 512
  #   chunking_method          = "recursive"
  #   embedding_model          = "jinaai/jina-embedding-t-en-v1"
  #   separators               = ["\n", " "]
  # }
}

output "example_id" {
  value       = datarobot_vector_database.example.id
  description = "The id for the example vector database"
}
