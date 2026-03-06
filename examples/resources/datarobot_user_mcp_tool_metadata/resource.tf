resource "datarobot_user_mcp_tool_metadata" "example" {
  name = "tool name"
  type = "userTool"
  mcp_server_version_id = "69a761e40746897942318e2f"
}


output "example_id" {
  value       = datarobot_user_mcp_tool_metadata.example.id
  description = "The id for the user mcp tool metadata resource"
}
