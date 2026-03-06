resource "datarobot_user_mcp_resource_metadata" "example" {
  name = "resource name"
  type = "userResce"
  mcp_server_version_id = "69a761e40746897942318e2f"
  uri: "uri://example_uri"
}


output "example_id" {
  value       = datarobot_user_mcp_resource_metadata.example.id
  description = "The id for the user mcp resource metadata resource"
}

