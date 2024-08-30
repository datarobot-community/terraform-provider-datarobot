resource "datarobot_remote_repository" "github_example" {
  name        = "An example GitHub remote repository"
  description = "Description for the example remote repository"
  location    = "https://github.com/datarobot/datarobot-user-models"
  source_type = "github"

  # optional personal access token
  # personal_access_token = "your_personal_access_token"
}

resource "datarobot_remote_repository" "gitlab_example" {
  name                  = "An example GitLab remote repository"
  location              = "https://gitlab.yourcompany.com/username/repository"
  source_type           = "gitlab-cloud"
  personal_access_token = "your_personal_access_token"
}

resource "datarobot_remote_repository" "bitbucket_example" {
  name        = "An example BitBucket remote repository"
  location    = "https://bitbucket.yourcompany.com/projects/PROJECTKEY/repos/REPONAME/browse"
  source_type = "bitbucket-server"

  # optional personal access token
  # personal_access_token = "your_personal_access_token"
}

resource "datarobot_remote_repository" "s3_example" {
  name        = "An example S3 remote repository"
  location    = "my-s3-bucket"
  source_type = "s3"

  # optional aws credentials for private buckets
  # aws_access_key_id = "your_aws_access_key_id"
  # aws_secret_access_key = "your_aws_secret_access_key"
  # aws_session_token = "your_aws_session_token"
}
