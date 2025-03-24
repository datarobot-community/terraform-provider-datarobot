resource "datarobot_azure_credential" "example" {
  name                    = "An example Azure credential"
  description             = "Description for the example Azure credential"
  azure_connection_string = "example_connection_string"
}
