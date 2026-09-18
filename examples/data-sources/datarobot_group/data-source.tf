data "datarobot_group" "finance" {
  name = "FinanceTeam"
}

output "finance_group_id" {
  value = data.datarobot_group.finance.id
}
