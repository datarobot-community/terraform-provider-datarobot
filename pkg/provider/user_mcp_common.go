package provider

import (
	"context"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// No-op trace messages for User MCP metadata resources (Read/Update/Delete are intentional no-ops).
const (
	userMCPReadNoOpMsg = "Read is a no-op. This is by design. " +
		"Once this resource is created, it can't be updated outside the Terraform (No public API/UI support) and causes " +
		"resource drift."

	userMCPUpdateNoOpMsg = "Update is a no-op. This is by design. " +
		"All non RequiresReplace attributes are attributes with computed=True. And they can't be updated outside the " +
		"Terraform. So no Update will be triggered."

	userMCPDeleteNoOpMsg = "Delete is a no-op. This is by design. " +
		"This resource is associated with custom model version resource (custom_model_resource.go). It will be " +
		"created and frozen (no deletion) once a custom model version is created. " +
		"All user provided attributes of this resource are RequiresReplace. When they are updated, a new custom model " +
		"version will be created (Because updating these attributes also updates the files attribute " +
		"CustomModelResource)." +
		"With that, the expected behavior after the update of these RequiresReplace attributes is " +
		"Create + Delete (no-op)."
)

// NoOpRead loads state from req, logs a trace message, and writes state back to resp.
// statePtr must be a pointer to the resource's state model (e.g. &data).
// Use for immutable resources that have no GET API.
func NoOpRead(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse, traceMsg string, statePtr interface{}) {
	resp.Diagnostics.Append(req.State.Get(ctx, statePtr)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if traceMsg != "" {
		traceAPICall(traceMsg)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, reflect.ValueOf(statePtr).Elem().Interface())...)
}

// NoOpUpdate loads state from req, logs a trace message, and writes state back to resp.
// statePtr must be a pointer to the resource's state model.
// Use for immutable resources where Update is a no-op (echo state back).
func NoOpUpdate(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse, traceMsg string, statePtr interface{}) {
	resp.Diagnostics.Append(req.State.Get(ctx, statePtr)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if traceMsg != "" {
		traceAPICall(traceMsg)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, reflect.ValueOf(statePtr).Elem().Interface())...)
}

// NoOpDelete loads state from req and logs a trace message. It does not set resp state (Terraform clears it after Delete).
// statePtr must be a pointer to the resource's state model.
// Use when Delete is intentionally a no-op (e.g. resource is frozen with custom model version).
func NoOpDelete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse, traceMsg string, statePtr interface{}) {
	resp.Diagnostics.Append(req.State.Get(ctx, statePtr)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if traceMsg != "" {
		traceAPICall(traceMsg)
	}
}
