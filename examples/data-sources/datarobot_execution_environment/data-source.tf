data "datarobot_execution_environment" "lookup_by_name" {
  name = "[DataRobot] Python 3.12"
}

data "datarobot_execution_environment" "lookup_by_id_and_version_id" {
  id = "67a554bbfbef3a4ce2ab6700"  # Sample execution environment ID
  version_id = "68e53eb0b995c5121a0b583b"  # Sample execution environment version ID 
}
