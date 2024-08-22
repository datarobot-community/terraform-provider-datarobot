resource "datarobot_basic_credential" "example" {
  name        = "An example basic credential"
  description = "Description for the example basic credential"
  user        = "example_user"
  password    = "example_password"
}