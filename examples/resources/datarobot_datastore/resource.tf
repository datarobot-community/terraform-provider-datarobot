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

resource "datarobot_datastore" "example_jdbc" {
  canonical_name  = "Example JDBC Datastore"
  data_store_type = "jdbc"
  driver_id       = "5b4752844bf542000175dbea"
  fields = [
    {
      "name" : "address",
      "value" : "my-address"
    },
    {
      "name" : "database",
      "value" : "my-database"
    }
  ]
}

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

output "example_connector_id" {
  value       = datarobot_datastore.example_connector.id
  description = "The id for the example_connector datastore"
}
