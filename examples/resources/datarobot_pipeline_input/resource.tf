resource "datarobot_pipeline" "example" {
  source_file = "pipeline.py"
  mode        = "locked"
}

# Draft input (no version specified).
resource "datarobot_pipeline_input" "draft_example" {
  pipeline_id = datarobot_pipeline.example.id
  payload     = jsonencode({ param1 = "value1", param2 = 42 })
}

# Locked-version input.
resource "datarobot_pipeline_input" "locked_example" {
  pipeline_id = datarobot_pipeline.example.id
  version     = datarobot_pipeline.example.current_version
  payload     = jsonencode({ param1 = "value1", param2 = 42 })
}

output "input_id" {
  value       = datarobot_pipeline_input.locked_example.id
  description = "The ID of the pipeline input."
}
