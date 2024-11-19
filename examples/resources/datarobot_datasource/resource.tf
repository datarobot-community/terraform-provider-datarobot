resource "datarobot_datasource" "example_database_table" {
  canonical_name   = "Example Database Table Data Source"
  data_source_type = "dr-database-v1"
  params = {
    data_store_id = datarobot_datastore.example_database.id
    catalog       = "my-catalog"
    schema        = "my-schema"
    table         = "my-table"
  }
}

resource "datarobot_datasource" "example_database_query" {
  canonical_name   = "Example Database Query Data Source"
  data_source_type = "dr-database-v1"
  params = {
    data_store_id = datarobot_datastore.example_database.id
    query         = "SELECT * FROM my-table"
  }
}

resource "datarobot_datasource" "example_connector" {
  canonical_name   = "Example Connector Data Source"
  data_source_type = "dr-connector-v1"
  params = {
    data_store_id = datarobot_datastore.example_connector.id
    path          = "/my-folder/my-file.csv"
  }
}


output "example_id" {
  value       = datarobot_datasource.example_database_table.id
  description = "The id for the example database table datasource"
}
