resource "datarobot_memory_space" "example" {
  description = "My workspace memories"
}

output "datarobot_memory_space_id" {
  value = datarobot_memory_space.example.id
}
