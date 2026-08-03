package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPlannedDockerfilePath_generated(t *testing.T) {
	t.Parallel()

	got := plannedDockerfilePath("generated", types.StringValue("./Dockerfile"), types.StringNull())
	if !got.IsNull() {
		t.Fatalf("expected null path for generated dockerfile, got %v", got)
	}
}

func TestPlannedDockerfilePath_providedDefault(t *testing.T) {
	t.Parallel()

	got := plannedDockerfilePath("provided", types.StringNull(), types.StringNull())
	if got.ValueString() != artifactDockerfileDefaultPath {
		t.Fatalf("expected default path %q, got %q", artifactDockerfileDefaultPath, got.ValueString())
	}
}

func TestPlannedDockerfilePath_providedExplicit(t *testing.T) {
	t.Parallel()

	got := plannedDockerfilePath("provided", types.StringValue("./Dockerfile.generated"), types.StringNull())
	if got.ValueString() != "./Dockerfile.generated" {
		t.Fatalf("expected explicit path preserved, got %q", got.ValueString())
	}
}

func TestPlannedDockerfilePath_providedUsesStateWhenPlanUnknown(t *testing.T) {
	t.Parallel()

	got := plannedDockerfilePath("provided", types.StringUnknown(), types.StringValue("./custom/Dockerfile"))
	if got.ValueString() != "./custom/Dockerfile" {
		t.Fatalf("expected state path preserved, got %q", got.ValueString())
	}
}
