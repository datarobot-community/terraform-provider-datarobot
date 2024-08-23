resource "datarobot_use_case" "example" {
  name = "Example use case"
}

output "example_id" {
  value       = datarobot_use_case.example.id
  description = "The id for the example use case"
}
