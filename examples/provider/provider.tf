provider "datarobot" {
  # Instructions for using the DataRobot API
  # https://docs.datarobot.com/en/docs/api/reference/index.html
  #
  # Instructions for getting an API Key
  # https://docs.datarobot.com/en/docs/platform/account-mgmt/acct-settings/api-key-mgmt.html#api-key-management
  #
  # The Terraform provider requires an environment variable DATAROBOT_API_TOKEN
  # export DATAROBOT_API_TOKEN="the API Key value here"
  # 
  # (Optional) The Endpoint of the DataRobot API can be configured using the environment variable 
  # export DATAROBOT_ENDPOINT="the API endpoint value here"
  # If not specified the default endpoint is "https://app.datarobot.com/api/v2"
}
