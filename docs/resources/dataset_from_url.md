---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "datarobot_dataset_from_url Resource - datarobot"
subcategory: ""
description: |-
  Data set from file
---

# datarobot_dataset_from_url (Resource)

Data set from file

## Example Usage

```terraform
resource "datarobot_dataset_from_url" "example" {
  url          = "[URL to upload from]"
  use_case_ids = [datarobot_use_case.example.id]

  # Optional
  name = "Example Dataset"
}

output "example_id" {
  value       = datarobot_dataset_from_url.example.id
  description = "The id for the example dataset"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `url` (String) The URL to upload the Dataset from.

### Optional

- `name` (String) The name of the Dataset.
- `use_case_ids` (List of String) The list of Use Case IDs to add the Dataset to.

### Read-Only

- `id` (String) The ID of the Dataset.