resource "datarobot_pipeline_image" "example" {
  name              = "my-pipeline-image"
  description       = "Python image for my pipeline"
  packages          = ["numpy==1.26.0", "pandas>=2.0,<3.0"]
  python_base_image = "covalent-runtime-image:latest"
}

output "example_id" {
  value       = datarobot_pipeline_image.example.id
  description = "The ID of the pipeline image."
}
