
resource "datarobot_use_case" "example" {
  name = "Example use case"
}

resource "datarobot_notebook" "example" {
  file_path   = "/path/to/your/notebook.ipynb"
  use_case_id = datarobot_use_case.example.id
}
