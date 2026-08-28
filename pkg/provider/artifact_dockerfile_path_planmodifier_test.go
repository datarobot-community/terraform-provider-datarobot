package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPlannedDockerfilePath_generated(t *testing.T) {
	t.Parallel()

	got := plannedDockerfilePath("generated", types.StringValue("./Dockerfile"))
	if !got.IsNull() {
		t.Fatalf("expected null path for generated dockerfile, got %v", got)
	}
}

func TestPlannedDockerfilePath_providedDefault(t *testing.T) {
	t.Parallel()

	got := plannedDockerfilePath("provided", types.StringNull())
	if got.ValueString() != artifactDockerfileDefaultPath {
		t.Fatalf("expected default path %q, got %q", artifactDockerfileDefaultPath, got.ValueString())
	}
}

func TestPlannedDockerfilePath_providedExplicit(t *testing.T) {
	t.Parallel()

	got := plannedDockerfilePath("provided", types.StringValue("./Dockerfile.generated"))
	if got.ValueString() != "./Dockerfile.generated" {
		t.Fatalf("expected explicit path preserved, got %q", got.ValueString())
	}
}

// A previously-set custom path must not stick around once the user removes
// `path` from config: an omitted path always resolves to the documented
// default, regardless of what was in state before.
func TestPlannedDockerfilePath_providedOmittedResetsCustomStateToDefault(t *testing.T) {
	t.Parallel()

	got := plannedDockerfilePath("provided", types.StringNull())
	if got.ValueString() != artifactDockerfileDefaultPath {
		t.Fatalf("expected omitted path to reset to default %q, got %q", artifactDockerfileDefaultPath, got.ValueString())
	}
}
