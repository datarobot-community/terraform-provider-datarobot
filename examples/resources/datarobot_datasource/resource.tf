resource "datarobot_datastore" "example_database" {
  canonical_name  = "Example Database Datastore"
  data_store_type = "dr-database-v1"
  driver_id       = "64a288a50636598d75df7f82"
  fields = [
    {
      "id" : "bq.project_id",
      "name" : "Project Id",
      "value" : "project-id"
    },
  ]
}

resource "datarobot_datastore" "example_connector" {
  canonical_name  = "Example Connector Datastore"
  data_store_type = "dr-connector-v1"
  connector_id    = "65538041dde6a1d664d0b2ec"
  fields = [
    {
      "id" : "fs.defaultFS",
      "name" : "Bucket Name",
      "value" : "my-bucket"
    }
  ]
}

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
