package provider

import (
	"regexp"
	"slices"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// With `source` and `image_build_config` the primary container's image_uri is
// whatever the build produces, so a configured one is refused before an artifact
// exists. A locked artifact cannot skip waiting for that build either: the
// platform locks an artifact only once its build has completed.

func testPrimaryWithImageURI(imageURI types.String) ArtifactContainerModel {
	container := testPrimaryWithBuildConfig()
	container.ImageURI = imageURI
	return container
}

func TestValidateContainerImageURINotSourceManaged(t *testing.T) {
	t.Parallel()

	sidecar := ArtifactContainerModel{
		Name:     types.StringValue("sidecar"),
		Primary:  types.BoolValue(false),
		ImageURI: types.StringValue("busybox:latest"),
	}

	tests := []struct {
		name             string
		container        ArtifactContainerModel
		sourceConfigured bool
		wantRefused      bool
	}{
		{
			name:             "configured image_uri on a source-built container",
			container:        testPrimaryWithImageURI(types.StringValue("containous/whoami:latest")),
			sourceConfigured: true,
			wantRefused:      true,
		},
		{
			name:             "empty image_uri",
			container:        testPrimaryWithImageURI(types.StringValue("")),
			sourceConfigured: true,
			wantRefused:      true,
		},
		{
			name:             "unresolved image_uri waits for a later walk",
			container:        testPrimaryWithImageURI(types.StringUnknown()),
			sourceConfigured: true,
		},
		{
			name:             "absent image_uri",
			container:        testPrimaryWithImageURI(types.StringNull()),
			sourceConfigured: true,
		},
		{
			name:             "sidecar without image_build_config keeps its image_uri",
			container:        sidecar,
			sourceConfigured: true,
		},
		{
			name:      "no source leaves image_uri alone",
			container: testPrimaryWithImageURI(types.StringValue("containous/whoami:latest")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := &tfresource.ValidateConfigResponse{}
			validateContainerImageURINotSourceManaged(resp, testArtifactContainerPath(), tt.container, tt.sourceConfigured)

			want := ""
			if tt.wantRefused {
				want = "Conflicting image source|"
			}
			if got := artifactDiagSummaries(resp); got != want {
				t.Fatalf("diagnostics = %q, want %q", got, want)
			}
		})
	}
}

// Status, wait_for_build and image_uri on a source-built primary, through the
// whole validator.
func TestArtifactSourceImageURIAndWaitForBuildRules(t *testing.T) {
	t.Parallel()

	const (
		refused = "Conflicting image source"
		noWait  = "Invalid wait_for_build on locked artifact"
	)
	draft := types.StringValue(string(client.ArtifactStatusDraft))
	locked := types.StringValue(string(client.ArtifactStatusLocked))
	image := types.StringValue("containous/whoami:latest")
	dir := t.TempDir()

	tests := []struct {
		name     string
		status   types.String
		wait     types.Bool
		imageURI types.String
		want     []string
	}{
		{name: "draft", status: draft, wait: types.BoolNull(), imageURI: types.StringNull()},
		{name: "draft not waiting", status: draft, wait: types.BoolValue(false), imageURI: types.StringNull()},
		{name: "draft not waiting with image_uri", status: draft, wait: types.BoolValue(false), imageURI: image, want: []string{refused}},
		{name: "draft with empty image_uri", status: draft, wait: types.BoolNull(), imageURI: types.StringValue(""), want: []string{refused}},
		{name: "locked waiting with image_uri", status: locked, wait: types.BoolValue(true), imageURI: image, want: []string{refused}},
		{name: "locked not waiting", status: locked, wait: types.BoolValue(false), imageURI: types.StringNull(), want: []string{noWait}},
		{name: "locked not waiting with image_uri", status: locked, wait: types.BoolValue(false), imageURI: image, want: []string{refused, noWait}},
		{name: "status unset means locked", status: types.StringNull(), wait: types.BoolValue(false), imageURI: types.StringNull(), want: []string{noWait}},
		{name: "unresolved wait_for_build", status: locked, wait: types.BoolUnknown(), imageURI: types.StringNull()},
		{name: "unresolved status", status: types.StringUnknown(), wait: types.BoolValue(false), imageURI: types.StringNull()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := &tfresource.ValidateConfigResponse{}
			validateArtifactModel(resp, ArtifactResourceModel{
				Name:   types.StringValue("rules"),
				Status: tt.status,
				Type:   types.StringValue(string(client.ArtifactTypeService)),
				Source: &ArtifactSourceModel{Dir: types.StringValue(dir), WaitForBuild: tt.wait},
				Spec:   testDraftSourceSpec(testPrimaryWithImageURI(tt.imageURI)),
			})

			var got []string
			for _, d := range resp.Diagnostics.Errors() {
				got = append(got, d.Summary())
			}
			slices.Sort(got)
			want := slices.Clone(tt.want)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Fatalf("errors = %v, want %v", got, want)
			}
		})
	}
}

// The reported configuration: a draft that does not wait for its build, with an
// explicit image_uri. It used to plan and then fail apply with "Provider produced
// inconsistent result after apply ... image_uri", after the artifact existed.
func TestIntegrationArtifactPlanRejectsExplicitImageURIWithSource(t *testing.T) {
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

// A variable is unknown on the validate walk and refused on the plan walk.
func TestIntegrationArtifactPlanRejectsVariableImageURIWithSource(t *testing.T) {
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

// Locking without waiting for the build used to be accepted with an image_uri.
// The platform refuses that lock until the build completes, so apply failed after
// creating the artifact.
func TestIntegrationArtifactPlanRejectsLockedWithoutWaitingForBuild(t *testing.T) {
	dir := t.TempDir()

	testArtifactPlanOnlyStep(t, `
resource "datarobot_artifact" "test" {
  name   = "locked-without-waiting"
  status = "locked"
  source = {
    dir            = "`+dir+`"
    wait_for_build = false
  }
  spec = {
    container_groups = [{
      containers = [{
        primary   = true
        port      = 8080
        image_uri = "nginx:latest"
        image_build_config = {
          dockerfile = { source = "provided" }
        }
      }]
    }]
  }
}`, regexp.MustCompile("Invalid wait_for_build on locked artifact"))
}

// A status behind a variable is unknown on the validate walk, where it must not
// be read as locked: this draft that skips the wait is valid.
func TestIntegrationArtifactPlanAcceptsVariableDraftStatusWithoutWaiting(t *testing.T) {
	dir := t.TempDir()

	testArtifactPlanOnlyStep(t, `
variable "status" {
  type    = string
  default = "draft"
}

resource "datarobot_artifact" "test" {
  name   = "variable-draft-without-waiting"
  status = var.status
  source = {
    dir            = "`+dir+`"
    wait_for_build = false
  }
  spec = {
    container_groups = [{
      containers = [{
        primary = true
        port    = 8080
        image_build_config = {
          dockerfile = { source = "provided" }
        }
      }]
    }]
  }
}`, nil)
}
