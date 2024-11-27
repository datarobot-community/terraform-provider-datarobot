resource "datarobot_custom_metric_job" "example" {
  name = "Example Custom Metric Job"
  files = [
    "file1.py",
    "file2.py",
  ]
  environment_id = "65f9b27eab986d30d4c64268"

  # Optional
  description = "Example Custom Metric Job Description"
  runtime_parameter_values = [
    {
      key   = "EXAMPLE_PARAM",
      type  = "string",
      value = "val",
    },
  ]
  egress_network_policy = "none"
  resource_bundle_id    = "cpu.micro"
  units                 = "count"
  directionality        = "lowerIsBetter"
  type                  = "sum"
  is_model_specific     = false
}

output "example_id" {
  value       = datarobot_custom_job.example.id
  description = "The id for the example custom metric job"
}
