resource "datarobot_quota" "example" {
  # The id of the resource the quota applies to (a deployment id by default).
  resource_id   = "5e7e6a3f8e7d8c0001a1b2c3"
  resource_type = "deployment"

  default_rules = [
    {
      rule   = "requests"
      limit  = 750
      window = "day"
    },
    {
      rule   = "token"
      limit  = 100000000
      window = "day"
    },
  ]
}

output "example_id" {
  value       = datarobot_quota.example.id
  description = "The id for the example quota"
}
