---
name: new-resource
description: Create a new Terraform resource for the DataRobot provider
---

# Create New Terraform Resource

This skill helps you create a new Terraform resource for the DataRobot provider following established patterns.

## Steps

1. **Create the resource file** at `pkg/provider/{resource_name}_resource.go`:
   - Define the resource struct with `provider *Provider`
   - Implement `resource.Resource` interface (Metadata, Schema, Configure, Create, Read, Update, Delete)
   - Implement `resource.ResourceWithImportState` for import support
   - Add proper MarkdownDescription for all schema attributes

2. **Create the model file** at `pkg/provider/{resource_name}_model.go`:
   - Define the ResourceModel struct with `types.*` fields
   - Match field names to API response/request structures

3. **Add client service methods** in `internal/client/`:
   - Add interface methods to `service.go`
   - Implement methods in appropriate `*_service.go` file

4. **Create tests** at `pkg/provider/{resource_name}_resource_test.go`:
   - Add TestAcc{ResourceName}Resource for acceptance tests
   - Add Test{ResourceName}ResourceSchema for schema validation

5. **Add example** in `examples/resources/{resource_name}/`:
   - Create `resource.tf` with usage example

6. **Update documentation**:
   - Run `make generate` to create docs
   - Update CHANGELOG.md

## Template

```go
package provider

import (
    "context"
    "fmt"

    "github.com/datarobot-community/terraform-provider-datarobot/internal/client"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &{ResourceName}Resource{}
var _ resource.ResourceWithImportState = &{ResourceName}Resource{}

func New{ResourceName}Resource() resource.Resource {
    return &{ResourceName}Resource{}
}

type {ResourceName}Resource struct {
    provider *Provider
}

func (r *{ResourceName}Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_{resource_name}"
}
```
