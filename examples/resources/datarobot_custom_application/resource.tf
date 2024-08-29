resource "datarobot_application_source" "example" {
	local_files = ["start-app.sh", "streamlit-app.py"]
}

resource "datarobot_custom_application" "example" {
	name = "example custom application"
	source_version_id = datarobot_application_source.example.version_id

    # optional settings
    external_access_enabled = true
    external_access_recipients = [
        "recipient@example.com",
    ]
}

output "datarobot_custom_application_url" {
  value       = datarobot_custom_application.example.application_url
  description = "The URL for the example custom application"
}