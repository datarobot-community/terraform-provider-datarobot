---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "datarobot_datastore Resource - datarobot"
subcategory: ""
description: |-
  Data store
---

# datarobot_datastore (Resource)

Data store

## Example Usage

```terraform
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `canonical_name` (String) The user-friendly name of the data store.
- `data_store_type` (String) The type of data store.

### Optional

- `connector_id` (String) The identifier of the Connector if data_store_type is DR_CONNECTOR_V1
- `driver_id` (String) The identifier of the DataDriver if data_store_type is JDBC or DR_DATABASE_V1
- `fields` (List of Map of String) If the type is dr-database-v1, then the fields specify the configuration.
- `jdbc_url` (String) The full JDBC URL (for example: jdbc:postgresql://my.dbaddress.org:5432/my_db).

### Read-Only

- `id` (String) The ID of the data store.
