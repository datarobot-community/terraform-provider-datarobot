package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// `source` plus `image_build_config` makes the primary container's image_uri a
// provider-managed value: apply uploads the source, builds an image and writes
// whatever the platform reports. A configured image_uri cannot survive that, and
// Terraform will not let the provider plan around it either, because a plan may
// not turn a known configured value unknown. The combination is refused instead,
// at plan time, before an artifact exists to orphan.
//
// One shape is exempt, and has to stay exempt: locking without waiting for the
// build. There the lock request needs an image the build has not produced, so
// validateArtifactSource *requires* image_uri. The two rules are complements;
// TestArtifactImageURISourceRulesAreComplements pins that they cannot both fire.

func testSourceBuiltContainer(imageURI types.String) ArtifactContainerModel {
	return ArtifactContainerModel{
		Name:    types.StringValue("main"),
		Primary: types.BoolValue(true),
		Port:    types.Int64Value(8080),
		Build:   artifactBuildNull(),
		ImageBuildConfig: &ArtifactImageBuildConfigModel{
			CodeRef:    artifactCodeRefNull(),
			Dockerfile: &ArtifactDockerfileModel{Source: types.StringValue("provided")},
		},
		ImageURI: imageURI,
	}
}

func TestValidateContainerImageURINotSourceManaged(t *testing.T) {
	t.Parallel()

	sidecar := ArtifactContainerModel{
		Name:     types.StringValue("sidecar"),
		Primary:  types.BoolValue(false),
		ImageURI: types.StringValue("busybox:latest"),
	}

	tests := []struct {
		name            string
		container       ArtifactContainerModel
		sourceBuildsImg bool
		wantSummary     string
	}{
		{
			name:            "configured image_uri on a source-built container is refused",
			container:       testSourceBuiltContainer(types.StringValue("containous/whoami:latest")),
			sourceBuildsImg: true,
			wantSummary:     "Conflicting image source",
		},
		{
			// Unknown means set-but-unresolved. The plan walk re-validates with
			// variables and data sources resolved, and Create re-validates a
			// same-apply resource reference, so nothing is lost by waiting.
			name:            "unresolved image_uri defers",
			container:       testSourceBuiltContainer(types.StringUnknown()),
			sourceBuildsImg: true,
		},
		{
			name:            "absent image_uri is the supported shape",
			container:       testSourceBuiltContainer(types.StringNull()),
			sourceBuildsImg: true,
		},
		{
			name:            "empty image_uri is not a configured image",
			container:       testSourceBuiltContainer(types.StringValue("")),
			sourceBuildsImg: true,
		},
		{
			// Nothing builds a sidecar, so naming its image stays correct.
			name:            "container without image_build_config keeps its image_uri",
			container:       sidecar,
			sourceBuildsImg: true,
		},
		{
			name:            "no source-driven build leaves image_uri alone",
			container:       testSourceBuiltContainer(types.StringValue("containous/whoami:latest")),
			sourceBuildsImg: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := &tfresource.ValidateConfigResponse{}
			validateContainerImageURINotSourceManaged(resp, testArtifactContainerPath(), tt.container, tt.sourceBuildsImg)

			got := artifactDiagSummaries(resp)
			if tt.wantSummary == "" {
				if got != "" {
					t.Fatalf("unexpected diagnostics: %s", got)
				}
				return
			}
			if got != tt.wantSummary+"|" {
				t.Fatalf("diagnostics = %q, want %q", got, tt.wantSummary+"|")
			}
		})
	}
}

func TestArtifactSourceBuildsContainerImage(t *testing.T) {
	t.Parallel()

	withSource := func(wait types.Bool) ArtifactResourceModel {
		return ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:          types.StringValue("app"),
				WaitForBuild: wait,
			},
		}
	}

	tests := []struct {
		name   string
		data   ArtifactResourceModel
		status string
		want   bool
	}{
		{
			name:   "no source means no source-driven build",
			data:   ArtifactResourceModel{},
			status: string(client.ArtifactStatusDraft),
		},
		{
			name:   "draft waiting for the build",
			data:   withSource(types.BoolNull()),
			status: string(client.ArtifactStatusDraft),
			want:   true,
		},
		{
			// A draft has nothing to lock, so image_uri buys the user nothing
			// here and is still overwritten. This is the reported repro.
			name:   "draft not waiting for the build",
			data:   withSource(types.BoolValue(false)),
			status: string(client.ArtifactStatusDraft),
			want:   true,
		},
		{
			name:   "locked waiting for the build",
			data:   withSource(types.BoolValue(true)),
			status: string(client.ArtifactStatusLocked),
			want:   true,
		},
		{
			name:   "locked with wait_for_build unset defaults to waiting",
			data:   withSource(types.BoolNull()),
			status: string(client.ArtifactStatusLocked),
			want:   true,
		},
		{
			// The exemption: the lock needs an image the build has not produced.
			name:   "locked not waiting for the build",
			data:   withSource(types.BoolValue(false)),
			status: string(client.ArtifactStatusLocked),
		},
		{
			name:   "locked with an unresolved wait_for_build defaults to waiting",
			data:   withSource(types.BoolUnknown()),
			status: string(client.ArtifactStatusLocked),
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := artifactSourceBuildsContainerImage(tt.data, tt.status); got != tt.want {
				t.Fatalf("artifactSourceBuildsContainerImage = %v, want %v", got, tt.want)
			}
		})
	}
}

// One rule requires image_uri, the other refuses it. If they ever overlap the
// configuration becomes unconfigurable, so walk the matrix and assert that each
// cell produces at most one of them.
func TestArtifactImageURISourceRulesAreComplements(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	for _, status := range []string{string(client.ArtifactStatusDraft), string(client.ArtifactStatusLocked)} {
		for _, wait := range []types.Bool{types.BoolNull(), types.BoolValue(true), types.BoolValue(false)} {
			for _, imageURI := range []types.String{types.StringNull(), types.StringValue("containous/whoami:latest")} {
				data := ArtifactResourceModel{
					Name:   types.StringValue("complements"),
					Status: types.StringValue(status),
					Type:   types.StringValue(string(client.ArtifactTypeService)),
					Source: &ArtifactSourceModel{
						Dir:          types.StringValue(dir),
						WaitForBuild: wait,
					},
					Spec: &ArtifactSpecModel{
						ContainerGroups: []ArtifactContainerGroupModel{{
							Containers: []ArtifactContainerModel{testSourceBuiltContainer(imageURI)},
						}},
					},
				}

				resp := &tfresource.ValidateConfigResponse{}
				validateArtifactModel(resp, data)

				var refused, required bool
				for _, d := range resp.Diagnostics.Errors() {
					switch d.Summary() {
					case "Conflicting image source":
						refused = true
					case "Invalid wait_for_build on locked artifact":
						required = true
					}
				}
				if refused && required {
					t.Fatalf("status=%s wait=%v image_uri=%v: image_uri both required and refused",
						status, wait, imageURI)
				}

				// The one configuration that needs image_uri must accept it.
				lockedNoWait := status == string(client.ArtifactStatusLocked) &&
					!wait.IsNull() && !wait.ValueBool()
				if lockedNoWait && !imageURI.IsNull() && resp.Diagnostics.HasError() {
					t.Fatalf("status=%s wait=%v: locked artifact skipping the build wait must accept image_uri, got %s",
						status, wait, artifactDiagSummaries(resp))
				}
				if lockedNoWait && imageURI.IsNull() && !required {
					t.Fatalf("status=%s wait=%v: expected image_uri to be required", status, wait)
				}
			}
		}
	}
}

// The reported configuration, end to end through a real plan: a draft that does
// not wait for its build, with an explicit image_uri. Before this, plan
// succeeded and apply failed with "Provider produced inconsistent result after
// apply ... image_uri: was cty.StringVal(...), but now null", leaving the
// artifact and its repository behind.
func TestArtifactPlanRejectsExplicitImageURIWithSourceBuild(t *testing.T) {
	dir := t.TempDir()

	testArtifactPlanOnlyStep(t, `
resource "datarobot_artifact" "test" {
  name   = "explicit-image-uri-with-source"
  status = "draft"
  source = {
    dir            = "`+dir+`"
    wait_for_build = false
  }
  spec = {
    container_groups = [{
      containers = [{
        primary   = true
        port      = 8080
        image_uri = "containous/whoami:latest"
        image_build_config = {
          dockerfile = { source = "provided" }
        }
      }]
    }]
  }
}`, regexp.MustCompile("Conflicting image source"))
}

// Waiting for the build hits the same wall from the other side: the build
// finishes and apply writes the image it produced, not the configured one.
func TestArtifactPlanRejectsExplicitImageURIWhenWaitingForBuild(t *testing.T) {
	dir := t.TempDir()

	testArtifactPlanOnlyStep(t, `
resource "datarobot_artifact" "test" {
  name   = "explicit-image-uri-waiting"
  status = "draft"
  source = {
    dir = "`+dir+`"
  }
  spec = {
    container_groups = [{
      containers = [{
        primary   = true
        port      = 8080
        image_uri = "containous/whoami:latest"
        image_build_config = {
          dockerfile = { source = "provided" }
        }
      }]
    }]
  }
}`, regexp.MustCompile("Conflicting image source"))
}

// The rule reads unknown as set-but-unresolved, so it does not fire on the
// validate walk. Terraform validates again on the plan walk with the variable
// resolved, and the rule catches it there.
func TestArtifactPlanRejectsVariableImageURIWithSourceBuild(t *testing.T) {
	dir := t.TempDir()

	testArtifactPlanOnlyStep(t, `
variable "container_image" {
  type    = string
  default = "containous/whoami:latest"
}

resource "datarobot_artifact" "test" {
  name   = "variable-image-uri-with-source"
  status = "draft"
  source = {
    dir = "`+dir+`"
  }
  spec = {
    container_groups = [{
      containers = [{
        primary   = true
        port      = 8080
        image_uri = var.container_image
        image_build_config = {
          dockerfile = { source = "provided" }
        }
      }]
    }]
  }
}`, regexp.MustCompile("Conflicting image source"))
}

// Only the built container is constrained. A sidecar is never built from
// source, so its image stays the user's to name.
func TestArtifactPlanAcceptsSidecarImageURIWithSourceBuild(t *testing.T) {
	dir := t.TempDir()

	testArtifactPlanOnlyStep(t, `
resource "datarobot_artifact" "test" {
  name   = "sidecar-image-uri-with-source"
  status = "draft"
  source = {
    dir = "`+dir+`"
  }
  spec = {
    container_groups = [{
      containers = [
        {
          name    = "main"
          primary = true
          port    = 8080
          image_build_config = {
            dockerfile = { source = "provided" }
          }
        },
        {
          name      = "sidecar"
          primary   = false
          image_uri = "busybox:latest"
        },
      ]
    }]
  }
}`, nil)
}

// A reference to a resource created in the same apply is still unknown on the
// plan walk, so Create is the last place to catch it. It has to refuse before
// CreateArtifact runs, or the failed apply leaves an artifact behind — the
// orphan the report describes, one per attempt.
func TestArtifactCreateRejectsExplicitImageURIWithSourceBuild(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// No EXPECT: reaching the API at all fails the test.
	mockService := mock_client.NewMockService(ctrl)
	artifactResource := &ArtifactResource{provider: &Provider{service: mockService}}

	data := artifactResourceModelWithSource("apply-explicit-image-uri", t.TempDir())
	data.Spec.ContainerGroups[0].Containers[0].ImageURI = types.StringValue("containous/whoami:latest")

	_, diags := testArtifactApplyCreate(context.Background(), artifactResource, data)
	if !diags.HasError() {
		t.Fatal("expected Create to reject an explicit image_uri on a source-built container")
	}
	if got := diags.Errors()[0].Summary(); got != "Conflicting image source" {
		t.Fatalf("summary = %q, want %q", got, "Conflicting image source")
	}
}

// The same rule through Update, so a configuration that starts valid cannot be
// edited into the broken shape.
func TestArtifactUpdateRejectsExplicitImageURIWithSourceBuild(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	artifactResource := &ArtifactResource{provider: &Provider{service: mockService}}

	dir := t.TempDir()
	state := artifactResourceModelWithSource("apply-explicit-image-uri", dir)
	state.ArtifactID = types.StringValue("6ffab89d-de30-4fbf-a0f0-b6dfc00ae542")

	plan := artifactResourceModelWithSource("apply-explicit-image-uri", dir)
	plan.ArtifactID = state.ArtifactID
	plan.Spec.ContainerGroups[0].Containers[0].ImageURI = types.StringValue("containous/whoami:latest")

	_, diags := testArtifactApplyUpdate(context.Background(), artifactResource, plan, state)
	if !diags.HasError() {
		t.Fatal("expected Update to reject an explicit image_uri on a source-built container")
	}
	if got := diags.Errors()[0].Summary(); got != "Conflicting image source" {
		t.Fatalf("summary = %q, want %q", got, "Conflicting image source")
	}
}
