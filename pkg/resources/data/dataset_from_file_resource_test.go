package data

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

// Schema-only test retained in data package. Full acceptance test remains under provider package.
func TestDatasetFromFileResourceSchema(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	req := fwresource.SchemaRequest{}
	resp := &fwresource.SchemaResponse{}
	NewDatasetFromFileResource().Schema(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema diagnostics: %+v", resp.Diagnostics)
	}
	diags := resp.Schema.ValidateImplementation(ctx)
	if diags.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diags)
	}
}
