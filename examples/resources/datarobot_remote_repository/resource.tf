resource "datarobot_remote_repository" "example" {
  name        = "An example GitHub remote repository"
  description = "Description for the example remote repository"
  location    = "https://github.com/datarobot/datarobot-user-models"
  source_type = "github"

  # set the credential id for private repositories
  # credential_id = datarobot_api_token_credential.example.id
}
