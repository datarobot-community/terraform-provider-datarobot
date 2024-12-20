---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "datarobot_datasource Resource - datarobot"
subcategory: ""
description: |-
  Data source
---

# datarobot_datasource (Resource)

Data source

## Example Usage

```terraform
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `canonical_name` (String) The user-friendly name of the data source.
- `data_source_type` (String) The type of data source.
- `params` (Attributes) The data source parameters. (see [below for nested schema](#nestedatt--params))

### Read-Only

- `id` (String) The ID of the data source.

<a id="nestedatt--params"></a>
### Nested Schema for `params`

Required:

- `data_store_id` (String) The id of the DataStore.

Optional:

- `catalog` (String) The Catalog name in the database if supported.
- `fetch_size` (Number) A user specified fetch size in the range [1, 20000]. By default a fetchSize will be assigned to balance throughput and memory usage.
- `partition_column` (String) The name of the partition column.
- `path` (String) The user-specified path for BLOB storage.
- `query` (String) The user specified SQL query.
- `schema` (String) The name of the schema associated with the table.
- `table` (String) The name of specified database table.
