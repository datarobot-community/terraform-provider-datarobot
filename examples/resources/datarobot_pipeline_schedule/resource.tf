resource "datarobot_pipeline" "example" {
  source_file = "pipeline.py"
  mode        = "locked"
}

resource "datarobot_pipeline_input" "example" {
  pipeline_id = datarobot_pipeline.example.id
  version     = datarobot_pipeline.example.current_version
  payload     = jsonencode({ param1 = "value1" })
}

resource "datarobot_pipeline_schedule" "example" {
  pipeline_id      = datarobot_pipeline.example.id
  version          = datarobot_pipeline.example.current_version
  pipeline_input_id = datarobot_pipeline_input.example.id
  cron_expression  = "0 9 * * 1-5"
  timezone         = "America/New_York"
}

output "schedule_id" {
  value       = datarobot_pipeline_schedule.example.id
  description = "The ID of the pipeline schedule."
}
