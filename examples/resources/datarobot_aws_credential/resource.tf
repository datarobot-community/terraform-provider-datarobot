resource "datarobot_aws_credential" "example" {
  name                  = "An example AWS credential"
  description           = "Description for the example AWS credential"
  aws_access_key_id     = "example_access_key_id"
  aws_secret_access_key = "example_secret_access_key"
  aws_session_token     = "example_session_token"
}
