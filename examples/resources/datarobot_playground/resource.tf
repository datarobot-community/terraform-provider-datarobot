resource "datarobot_use_case" "example" {
  name = "Example use case"
}

resource "datarobot_playground" "example" {
  name        = "An example playground"
  use_case_id = datarobot_use_case.example.id
}

output "example_id" {
  value       = datarobot_playground.example.id
  description = "The id for the example playground"
}
