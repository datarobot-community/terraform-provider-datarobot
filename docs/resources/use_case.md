---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "datarobot_use_case Resource - datarobot"
subcategory: ""
description: |-
  Use case
---

# datarobot_use_case (Resource)

Use case

## Example Usage

```terraform
resource "datarobot_use_case" "example" {
  name = "Example use case"
}

output "example_id" {
  value       = datarobot_use_case.example.id
  description = "The id for the example use case"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the Use Case.

### Optional

- `description` (String) The description of the Use Case.

### Read-Only

- `id` (String) The ID of the Use Case.
