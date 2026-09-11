package provider

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-framework/path"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Terraform's validate walk runs before variables, data sources, and other
// resources resolve, so everything non-literal reaches ValidateConfig unknown.
// A rule that reads unknown as "absent" rejects configurations that are in fact
// complete. These tests pin the distinction: unknown means set-but-unresolved,
// null means absent.
//
// The plan-level cases below are deliberately written as literal HCL with
// `variable` blocks rather than through fmt.Sprintf. Interpolating a value into
// the configuration produces a literal, which is exactly the shape that hid
// this class of bug from the rest of the suite.

func testArtifactContainerPath() path.Path {
	return path.Root("spec").
		AtName("container_groups").AtListIndex(0).
		AtName("containers").AtListIndex(0)
}

func artifactDiagSummaries(resp *tfresource.ValidateConfigResponse) string {
	var b strings.Builder
	for _, d := range resp.Diagnostics.Errors() {
		b.WriteString(d.Summary())
		b.WriteString("|")
	}
	return b.String()
}

func TestArtifactContainerImageURIUnknownIsAnImageSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		imageURI    types.String
		wantSummary string
	}{
		{
			name:     "unknown image_uri defers the check",
			imageURI: types.StringUnknown(),
		},
		{
			name:     "literal image_uri satisfies the check",
			imageURI: types.StringValue("nginx:latest"),
		},
		{
			name:        "null image_uri is still missing",
			imageURI:    types.StringNull(),
			wantSummary: "Missing image source",
		},
		{
			name:        "empty image_uri is still missing",
			imageURI:    types.StringValue(""),
			wantSummary: "Missing image source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := &tfresource.ValidateConfigResponse{}
			validateArtifactContainer(resp, testArtifactContainerPath(), ArtifactContainerModel{
				Primary:  types.BoolValue(true),
				ImageURI: tt.imageURI,
			}, string(client.ArtifactStatusDraft), string(client.ArtifactTypeService), 1, false)

			if tt.wantSummary == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected no errors, got: %s", artifactDiagSummaries(resp))
				}
				return
			}
			if !strings.Contains(artifactDiagSummaries(resp), tt.wantSummary) {
				t.Fatalf("expected %q, got: %s", tt.wantSummary, artifactDiagSummaries(resp))
			}
		})
	}
}

// A locked artifact needs a built image. An unknown image_uri may well carry
// one, so the rule cannot fire until the reference resolves.
func TestArtifactLockedBuildConfigUnknownImageURIDefers(t *testing.T) {
	t.Parallel()

	buildConfig := func() *ArtifactImageBuildConfigModel {
		return &ArtifactImageBuildConfigModel{
			Dockerfile: &ArtifactDockerfileModel{Source: types.StringValue("provided")},
		}
	}

	resp := &tfresource.ValidateConfigResponse{}
	validateArtifactContainer(resp, testArtifactContainerPath(), ArtifactContainerModel{
		Primary:          types.BoolValue(true),
		ImageURI:         types.StringUnknown(),
		ImageBuildConfig: buildConfig(),
	}, string(client.ArtifactStatusLocked), string(client.ArtifactTypeService), 1, false)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unknown image_uri must defer the locked check, got: %s", artifactDiagSummaries(resp))
	}

	resp = &tfresource.ValidateConfigResponse{}
	validateArtifactContainer(resp, testArtifactContainerPath(), ArtifactContainerModel{
		Primary:          types.BoolValue(true),
		ImageURI:         types.StringNull(),
		ImageBuildConfig: buildConfig(),
	}, string(client.ArtifactStatusLocked), string(client.ArtifactTypeService), 1, false)
	if !strings.Contains(artifactDiagSummaries(resp), "Incomplete build configuration for locked artifact") {
		t.Fatalf("null image_uri must still be rejected, got: %s", artifactDiagSummaries(resp))
	}
}

func TestArtifactGeneratedDockerfileUnknownExecutionEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		eeID         types.String
		eeVersionID  types.String
		wantSummary  string
		wantNoErrors bool
	}{
		{
			name:         "unknown ids defer both checks",
			eeID:         types.StringUnknown(),
			eeVersionID:  types.StringUnknown(),
			wantNoErrors: true,
		},
		{
			name:        "null id is still missing",
			eeID:        types.StringNull(),
			eeVersionID: types.StringUnknown(),
			wantSummary: "Missing execution environment ID",
		},
		{
			name:        "null version id is still missing",
			eeID:        types.StringUnknown(),
			eeVersionID: types.StringNull(),
			wantSummary: "Missing execution environment version ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := &tfresource.ValidateConfigResponse{}
			validateImageBuildConfig(resp, testArtifactContainerPath(), &ArtifactImageBuildConfigModel{
				Dockerfile: &ArtifactDockerfileModel{
					Source:                        types.StringValue("generated"),
					ExecutionEnvironmentID:        tt.eeID,
					ExecutionEnvironmentVersionID: tt.eeVersionID,
					Entrypoint:                    []types.String{types.StringValue("/bin/sh")},
				},
			}, string(client.ArtifactTypeService))

			if tt.wantNoErrors {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected no errors, got: %s", artifactDiagSummaries(resp))
				}
				return
			}
			if !strings.Contains(artifactDiagSummaries(resp), tt.wantSummary) {
				t.Fatalf("expected %q, got: %s", tt.wantSummary, artifactDiagSummaries(resp))
			}
		})
	}
}

// wait_for_build = false on a locked artifact is only allowed when image_uri is
// given. It reads image_uri through artifactHasPrimaryImageURI, so an unknown
// value has to count there too.
func TestArtifactHasPrimaryImageURIUnknown(t *testing.T) {
	t.Parallel()

	specWith := func(uri types.String) *ArtifactSpecModel {
		return &ArtifactSpecModel{
			ContainerGroups: []ArtifactContainerGroupModel{{
				Containers: []ArtifactContainerModel{{
					Primary:  types.BoolValue(true),
					ImageURI: uri,
				}},
			}},
		}
	}

	tests := []struct {
		name string
		uri  types.String
		want bool
	}{
		{name: "unknown counts as configured", uri: types.StringUnknown(), want: true},
		{name: "literal counts as configured", uri: types.StringValue("nginx:latest"), want: true},
		{name: "null does not", uri: types.StringNull(), want: false},
		{name: "empty does not", uri: types.StringValue(""), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := artifactHasPrimaryImageURI(specWith(tt.uri)); got != tt.want {
				t.Fatalf("artifactHasPrimaryImageURI = %v, want %v", got, tt.want)
			}
		})
	}
}

// source.dir fed by a variable is unknown on the validate walk. The filesystem
// checks need the literal path and wait; a null dir is still reported.
func TestArtifactSourceDirUnknownDefersFilesystemChecks(t *testing.T) {
	t.Parallel()

	spec := &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{{
			Containers: []ArtifactContainerModel{{
				Primary: types.BoolValue(true),
				ImageBuildConfig: &ArtifactImageBuildConfigModel{
					Dockerfile: &ArtifactDockerfileModel{Source: types.StringValue("provided")},
				},
			}},
		}},
	}

	resp := &tfresource.ValidateConfigResponse{}
	validateArtifactSource(resp, ArtifactResourceModel{
		Status: types.StringValue("draft"),
		Source: &ArtifactSourceModel{Dir: types.StringUnknown()},
		Spec:   spec,
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unknown source.dir must defer, got: %s", artifactDiagSummaries(resp))
	}

	resp = &tfresource.ValidateConfigResponse{}
	validateArtifactSource(resp, ArtifactResourceModel{
		Status: types.StringValue("draft"),
		Source: &ArtifactSourceModel{Dir: types.StringNull()},
		Spec:   spec,
	})
	if !strings.Contains(artifactDiagSummaries(resp), "Missing source directory") {
		t.Fatalf("null source.dir must still be rejected, got: %s", artifactDiagSummaries(resp))
	}

	// An unknown dir must not swallow the rules that do not need the path.
	resp = &tfresource.ValidateConfigResponse{}
	validateArtifactSource(resp, ArtifactResourceModel{
		Status: types.StringValue("draft"),
		Type:   types.StringValue(string(client.ArtifactTypeNim)),
		Source: &ArtifactSourceModel{Dir: types.StringUnknown()},
		Spec:   spec,
	})
	if !strings.Contains(artifactDiagSummaries(resp), "Unsupported source on NIM artifacts") {
		t.Fatalf("expected the NIM rule to still run, got: %s", artifactDiagSummaries(resp))
	}
}

// primary = var.is_primary leaves "is this the primary container" undecidable
// until it resolves, so the primary-only rules defer rather than reject.
func TestArtifactPrimaryUnknownDefersPrimaryOnlyRules(t *testing.T) {
	t.Parallel()

	container := func(primary types.Bool) ArtifactContainerModel {
		return ArtifactContainerModel{
			Primary:  primary,
			ImageURI: types.StringValue("nginx:latest"),
			Routes: []ArtifactContainerRouteModel{{
				Path: types.StringValue("/"),
				Auth: types.StringValue("required"),
			}},
			ImageBuildConfig: &ArtifactImageBuildConfigModel{
				Dockerfile: &ArtifactDockerfileModel{Source: types.StringValue("provided")},
			},
		}
	}

	// Two containers: the sole-container shortcut does not apply.
	resp := &tfresource.ValidateConfigResponse{}
	validateImageBuildConfigPrimary(resp, testArtifactContainerPath(), container(types.BoolUnknown()), 2)
	validateArtifactContainerRoutes(resp, testArtifactContainerPath(), container(types.BoolUnknown()), 2)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unknown primary must defer, got: %s", artifactDiagSummaries(resp))
	}

	resp = &tfresource.ValidateConfigResponse{}
	validateImageBuildConfigPrimary(resp, testArtifactContainerPath(), container(types.BoolValue(false)), 2)
	validateArtifactContainerRoutes(resp, testArtifactContainerPath(), container(types.BoolValue(false)), 2)
	if got := artifactDiagSummaries(resp); !strings.Contains(got, "Unsupported on non-primary container") {
		t.Fatalf("primary = false must still be rejected, got: %s", got)
	}

	// Rules that do not depend on primary keep running under an unknown.
	dup := container(types.BoolUnknown())
	dup.Routes = append(dup.Routes, ArtifactContainerRouteModel{
		Path: types.StringValue("/"),
		Auth: types.StringValue("required"),
	})
	resp = &tfresource.ValidateConfigResponse{}
	validateArtifactContainerRoutes(resp, testArtifactContainerPath(), dup, 2)
	if !strings.Contains(artifactDiagSummaries(resp), "Duplicate route path") {
		t.Fatalf("expected the duplicate-path rule to still run, got: %s", artifactDiagSummaries(resp))
	}
}

func testArtifactPlanOnlyStep(t *testing.T, config string, expectError *regexp.Regexp) {
	t.Helper()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:             config,
			PlanOnly:           true,
			ExpectNonEmptyPlan: true,
			ExpectError:        expectError,
		}},
	})
}

// The configuration from examples/workflows/workload_replacement, which the
// workload docs link to as a runnable walkthrough.
func TestArtifactPlanAcceptsVariableImageURI(t *testing.T) {
	testArtifactPlanOnlyStep(t, `
variable "container_image" {
  type    = string
  default = "containous/whoami:latest"
}

resource "datarobot_artifact" "test" {
  name = "variable-image-uri"
  spec = {
    container_groups = [{
      containers = [{
        primary   = true
        port      = 8080
        image_uri = var.container_image
      }]
    }]
  }
}`, nil)
}

// Deferring the unknown must not lose the check: Terraform validates a second
// time during the plan walk, where the variable has resolved.
func TestArtifactPlanRejectsVariableImageURIResolvingToEmpty(t *testing.T) {
	testArtifactPlanOnlyStep(t, `
variable "container_image" {
  type    = string
  default = ""
}

resource "datarobot_artifact" "test" {
  name = "empty-image-uri"
  spec = {
    container_groups = [{
      containers = [{
        primary   = true
        port      = 8080
        image_uri = var.container_image
      }]
    }]
  }
}`, regexp.MustCompile("Missing image source"))
}

func TestArtifactPlanAcceptsVariableExecutionEnvironment(t *testing.T) {
	testArtifactPlanOnlyStep(t, `
variable "execution_environment_id" {
  type    = string
  default = "65f9b1bd6b6e0dfa5f6e1a4e"
}

variable "execution_environment_version_id" {
  type    = string
  default = "65f9b1bd6b6e0dfa5f6e1a4f"
}

resource "datarobot_artifact" "test" {
  name   = "variable-execution-environment"
  status = "draft"
  spec = {
    container_groups = [{
      containers = [{
        primary   = true
        port      = 8080
        image_uri = "nginx:latest"
        image_build_config = {
          dockerfile = {
            source                           = "generated"
            execution_environment_id         = var.execution_environment_id
            execution_environment_version_id = var.execution_environment_version_id
            entrypoint                       = ["/bin/sh", "-c", "start"]
          }
        }
      }]
    }]
  }
}`, nil)
}

// wait_for_build = false on a locked artifact, with image_uri behind a variable.
func TestArtifactPlanAcceptsVariableImageURIWithWaitForBuildFalse(t *testing.T) {
	dir := t.TempDir()

	testArtifactPlanOnlyStep(t, `
variable "container_image" {
  type    = string
  default = "nginx:latest"
}

resource "datarobot_artifact" "test" {
  name   = "variable-wait-for-build"
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
        image_uri = var.container_image
        image_build_config = {
          dockerfile = { source = "provided" }
        }
      }]
    }]
  }
}`, nil)
}

func TestArtifactPlanAcceptsVariableSourceDir(t *testing.T) {
	dir := t.TempDir()

	testArtifactPlanOnlyStep(t, `
variable "source_dir" {
  type    = string
  default = "`+dir+`"
}

resource "datarobot_artifact" "test" {
  name   = "variable-source-dir"
  status = "draft"
  source = {
    dir = var.source_dir
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

func TestArtifactPlanAcceptsVariablePrimaryOnMultiContainerGroup(t *testing.T) {
	testArtifactPlanOnlyStep(t, `
variable "is_primary" {
  type    = bool
  default = true
}

resource "datarobot_artifact" "test" {
  name = "variable-primary"
  spec = {
    container_groups = [{
      containers = [
        {
          name      = "main"
          primary   = var.is_primary
          port      = 8080
          image_uri = "nginx:latest"
          routes    = [{ path = "/", auth = "required" }]
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

// A reference to a resource created in the same apply is unknown even on the
// plan walk, so Create is the last place the deferred rules can run. It must
// reject the container before the API is called.
func TestArtifactCreateEnforcesDeferredImageSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// No EXPECT: reaching the API at all fails the test.
	mockService := mock_client.NewMockService(ctrl)
	artifactResource := &ArtifactResource{provider: &Provider{service: mockService}}

	data := artifactResourceModelWithSource("apply-missing-image", t.TempDir())
	data.Source = nil
	data.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig = nil

	_, diags := testArtifactApplyCreate(context.Background(), artifactResource, data)
	if !diags.HasError() {
		t.Fatal("expected Create to reject a container with no image source")
	}
	if got := diags.Errors()[0].Summary(); got != "Missing image source" {
		t.Fatalf("summary = %q, want %q", got, "Missing image source")
	}
}

// The same rule, reached through Update.
func TestArtifactUpdateEnforcesDeferredImageSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	artifactResource := &ArtifactResource{provider: &Provider{service: mockService}}

	state := artifactResourceModelWithSource("apply-missing-image", t.TempDir())
	state.Source = nil
	state.ArtifactID = types.StringValue("6ffab89d-de30-4fbf-a0f0-b6dfc00ae542")

	plan := artifactResourceModelWithSource("apply-missing-image", t.TempDir())
	plan.Source = nil
	plan.ArtifactID = state.ArtifactID
	plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig = nil

	_, diags := testArtifactApplyUpdate(context.Background(), artifactResource, plan, state)
	if !diags.HasError() {
		t.Fatal("expected Update to reject a container with no image source")
	}
	if got := diags.Errors()[0].Summary(); got != "Missing image source" {
		t.Fatalf("summary = %q, want %q", got, "Missing image source")
	}
}
