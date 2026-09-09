# Share a deployment with a group, addressing the group by the name it has in
# the identity provider rather than by its DataRobot ID. The name can come from
# a variable or the environment, so no ID is hardcoded in the stack.
resource "datarobot_deployment_shared_role" "finance_consumer" {
  deployment_id = datarobot_deployment.example.id
  group_name    = "FinanceTeam"
  role          = "CONSUMER"
}
