resource "datarobot_custom_job" "example" {
  name     = "Example Custom Job"
  job_type = "retraining"
  files = [
    "file1.py",
    "file2.py",
  ]
  environment_id = "65f9b27eab986d30d4c64268"

  # Optional
  description = "Example Custom Job Description"
  runtime_parameter_values = [
    {
      key   = "EXAMPLE_PARAM",
      type  = "string",
      value = "val",
    },
  ]
  egress_network_policy = "none"
  resource_bundle_id    = "cpu.micro"
  schedule = {
    minute       = ["15", "45"]
    hour         = ["*"]
    month        = ["*"]
    day_of_month = ["*"]
    day_of_week  = ["*"]
  }
}

output "example_id" {
  value       = datarobot_custom_job.example.id
  description = "The id for the example custom job"
}
