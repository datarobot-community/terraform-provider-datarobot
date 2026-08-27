data "datarobot_workload" "example" {
  id = var.workload_id
}

output "workload_endpoint" {
  value = data.datarobot_workload.example.endpoint
}

output "workload_status" {
  value = data.datarobot_workload.example.status
}