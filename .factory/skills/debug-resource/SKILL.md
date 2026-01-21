---
name: debug-resource
description: Debug and troubleshoot Terraform resource issues
---

# Debug Terraform Resource

This skill helps debug issues with Terraform resources in the DataRobot provider.

## Common Issues

### 1. Resource Not Found After Create
- Check if the API returns the correct ID
- Verify the ID is stored in state: `data.ID = types.StringValue(response.ID)`
- Look for `client.NotFoundError` handling in Read

### 2. State Drift
- Ensure Read populates all computed fields
- Check for fields that should use `UseStateForUnknown()` plan modifier
- Verify Optional fields handle empty vs null correctly

### 3. Update Not Working
- Check if field has `RequiresReplace()` modifier (forces new resource)
- Verify Update sends only changed fields to API
- Look for proper error handling in Update

### 4. Import Fails
- Ensure `ImportState` uses `resource.ImportStatePassthroughID`
- Verify Read can reconstruct full state from just the ID

## Debug Commands

```bash
# Enable Terraform debug logging
export TF_LOG=DEBUG
export TF_LOG_PATH=./terraform.log

# Run with trace API calls
export TF_LOG_PROVIDER=TRACE

# Test specific resource
go test -v ./pkg/provider -run TestAcc{ResourceName} -timeout 30m
```

## Useful Patterns

```go
// Add debug tracing
traceAPICall("CreateResourceName")

// Check for not found errors
if _, ok := err.(*client.NotFoundError); ok {
    resp.State.RemoveResource(ctx)
    return
}

// Add diagnostic details
resp.Diagnostics.AddError(
    "Error creating resource",
    fmt.Sprintf("Could not create: %s", err.Error()),
)
```
