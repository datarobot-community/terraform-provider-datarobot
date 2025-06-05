resource "datarobot_app_oauth" "test" {
  name          = "An example OAuth app"
  description   = "An example OAuth app for testing purposes"
  client_id     = "example-client-id"
  client_secret = "example-client-secret"

}
