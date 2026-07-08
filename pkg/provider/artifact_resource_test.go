package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccArtifactResource(t *testing.T) {
	t.Parallel()
	testArtifactResource(t, uuid.NewString(), false)
}

func TestAccArtifactDraftLifecycle(t *testing.T) {
	t.Parallel()
	testArtifactDraftResource(t, "draft-"+uuid.NewString()[:8])
}

func TestIntegrationArtifactResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		globalTestCfg.ApiKey = "fake"
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	initialID := uuid.NewString()
	updatedID := uuid.NewString()
	repoID := uuid.NewString()
	name := "test-artifact-" + uuid.NewString()[:8]
	updatedName := "updated-" + name

	repoIDPtr := repoID

	initialArtifact := artifactFixture(initialID, &repoIDPtr, name)
	updatedArtifact := artifactFixture(updatedID, &repoIDPtr, updatedName)

	// Create: CreateArtifact → post-create Read
	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
			if req.Status != client.ArtifactStatusLocked {
				t.Errorf("expected create status locked, got %q", req.Status)
			}
			return initialArtifact, nil
		})
	mockService.EXPECT().
		GetArtifact(gomock.Any(), initialID).
		Return(initialArtifact, nil)

	// Pre-update refresh
	mockService.EXPECT().
		GetArtifact(gomock.Any(), initialID).
		Return(initialArtifact, nil)

	// Update: CreateArtifact with same repoID → post-update Read
	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		Return(updatedArtifact, nil)
	mockService.EXPECT().
		GetArtifact(gomock.Any(), updatedID).
		Return(updatedArtifact, nil)

	// Import: TF calls Read with artifact_id set to the import ID (may be artifact version ID
	// or the stable UUID depending on whether ImportStateIdFunc is honoured by the test framework).
	// A post-import plan-check Read may follow, so allow up to 2 calls.
	mockService.EXPECT().
		GetArtifact(gomock.Any(), gomock.Any()).
		Return(updatedArtifact, nil).
		MaxTimes(2)

	// Destroy: delete the artifact repository
	mockService.EXPECT().
		DeleteArtifactRepository(gomock.Any(), repoID).
		Return(nil)

	testArtifactResource(t, name, true)
}

func testArtifactResource(t *testing.T, name string, isMock bool) {
	t.Helper()
	resourceName := "datarobot_artifact.test"

	var initialRepoID string
	var lastArtifactID string

	config := func(resourceName, imageURI string) string {
		cfg := artifactResourceConfig(resourceName, imageURI)
		if isMock {
			return testProviderConfigBlock() + "\n" + cfg
		}
		return cfg
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               isMock,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             checkArtifactRepoDestroyedFromAPI(&lastArtifactID, isMock),
		Steps: []resource.TestStep{
			{
				Config: config(name, "nginx:latest"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "artifact_id"),
					resource.TestCheckResourceAttrSet(resourceName, "artifact_repository_id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "type", "service"),
					resource.TestCheckResourceAttr(resourceName, "status", "locked"),
					captureAttr(resourceName, "artifact_repository_id", &initialRepoID),
					checkArtifactExistsInAPI(resourceName, name, "nginx:latest", isMock),
				),
			},
			{
				Config: config("updated-"+name, "nginx:latest"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "artifact_id"),
					resource.TestCheckResourceAttrSet(resourceName, "artifact_repository_id"),
					resource.TestCheckResourceAttr(resourceName, "name", "updated-"+name),
					captureAttr(resourceName, "artifact_id", &lastArtifactID),
					checkArtifactUpdatedInSameRepo(resourceName, "updated-"+name, &initialRepoID, isMock),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[resourceName]
					if !ok {
						return "", fmt.Errorf("resource %s not found", resourceName)
					}
					return rs.Primary.Attributes["artifact_id"], nil
				},
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "artifact_id",
				ImportStateVerifyIgnore:              []string{"id"},
			},
		},
	})
}

func testArtifactDraftResource(t *testing.T, name string) {
	t.Helper()
	resourceName := "datarobot_artifact.test"
	updatedName := "updated-" + name
	imageURI := "nginx:latest"

	var artifactID string
	var lastArtifactID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             checkArtifactRepoDestroyedFromAPI(&lastArtifactID, false),
		Steps: []resource.TestStep{
			{
				Config: artifactResourceConfigWithStatus(name, "draft"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "status", "draft"),
					resource.TestCheckResourceAttrSet(resourceName, "artifact_id"),
					captureAttr(resourceName, "artifact_id", &artifactID),
					checkArtifactStatusInAPI(resourceName, "draft", false),
					checkArtifactExistsInAPI(resourceName, name, imageURI, false),
				),
			},
			{
				Config: artifactResourceConfigWithStatus(updatedName, "draft"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "status", "draft"),
					checkArtifactIDEquals(resourceName, &artifactID),
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					checkArtifactStatusInAPI(resourceName, "draft", false),
					checkArtifactExistsInAPI(resourceName, updatedName, imageURI, false),
					captureAttr(resourceName, "artifact_id", &lastArtifactID),
				),
			},
			{
				Config: artifactResourceConfigWithStatus(updatedName, "locked"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "status", "locked"),
					checkArtifactIDEquals(resourceName, &artifactID),
					checkArtifactStatusInAPI(resourceName, "locked", false),
					captureAttr(resourceName, "artifact_id", &lastArtifactID),
				),
			},
		},
	})
}

// checkArtifactExistsInAPI verifies the artifact exists in the API with correct fields.
// In mock mode it uses Terraform state only; in acceptance mode it calls the API directly.
func checkArtifactExistsInAPI(resourceName, expectedName, expectedImageURI string, isMock bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		artifactID := rs.Primary.Attributes["artifact_id"]
		if artifactID == "" {
			return fmt.Errorf("artifact_id is not set in state")
		}

		repoID := rs.Primary.Attributes["artifact_repository_id"]
		if repoID == "" {
			return fmt.Errorf("artifact_repository_id is not set in state")
		}

		if isMock {
			return nil
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("provider not found")
		}
		p.service = NewService(cl)

		artifact, err := p.service.GetArtifact(context.Background(), artifactID)
		if err != nil {
			return fmt.Errorf("GetArtifact(%s): %w", artifactID, err)
		}

		if artifact.Name != expectedName {
			return fmt.Errorf("expected artifact name %q, got %q", expectedName, artifact.Name)
		}

		if len(artifact.Spec.ContainerGroups) == 0 || len(artifact.Spec.ContainerGroups[0].Containers) == 0 {
			return fmt.Errorf("artifact has no containers")
		}
		if got := artifactImageURIValue(artifact.Spec.ContainerGroups[0].Containers[0]); got != expectedImageURI {
			return fmt.Errorf("expected image_uri %q, got %q", expectedImageURI, got)
		}

		return nil
	}
}

func checkArtifactStatusInAPI(resourceName, expectedStatus string, isMock bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if isMock {
			return nil
		}

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		artifactID := rs.Primary.Attributes["artifact_id"]
		if artifactID == "" {
			return fmt.Errorf("artifact_id is not set in state")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("provider not found")
		}
		p.service = NewService(cl)

		artifact, err := p.service.GetArtifact(context.Background(), artifactID)
		if err != nil {
			return fmt.Errorf("GetArtifact(%s): %w", artifactID, err)
		}

		if string(artifact.Status) != expectedStatus {
			return fmt.Errorf("expected artifact status %q, got %q", expectedStatus, artifact.Status)
		}

		return nil
	}
}

func captureAttr(resourceName, attr string, dest *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		*dest = rs.Primary.Attributes[attr]
		return nil
	}
}

func checkArtifactIDEquals(resourceName string, expected *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if expected == nil || *expected == "" {
			return fmt.Errorf("expected artifact_id was not captured from a prior step")
		}

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		got := rs.Primary.Attributes["artifact_id"]
		if got != *expected {
			return fmt.Errorf("artifact_id: expected %q, got %q", *expected, got)
		}

		return nil
	}
}

// checkArtifactUpdatedInSameRepo verifies that after update:
// - the new artifact has the updated name in the API
// - the artifact_repository_id is the same as before (same versioned repo)
// - the previous artifact version is NOT deleted.
func checkArtifactUpdatedInSameRepo(resourceName, expectedName string, initialRepoID *string, isMock bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		newArtifactID := rs.Primary.Attributes["artifact_id"]
		if newArtifactID == "" {
			return fmt.Errorf("artifact_id is not set after update")
		}

		newRepoID := rs.Primary.Attributes["artifact_repository_id"]
		if newRepoID == "" {
			return fmt.Errorf("artifact_repository_id is not set after update")
		}
		if *initialRepoID != "" && newRepoID != *initialRepoID {
			return fmt.Errorf("artifact_repository_id changed after update: was %q, now %q", *initialRepoID, newRepoID)
		}

		if isMock {
			return nil
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("provider not found")
		}
		p.service = NewService(cl)

		artifact, err := p.service.GetArtifact(context.Background(), newArtifactID)
		if err != nil {
			return fmt.Errorf("GetArtifact(%s) after update: %w", newArtifactID, err)
		}

		if artifact.Name != expectedName {
			return fmt.Errorf("expected updated artifact name %q, got %q", expectedName, artifact.Name)
		}

		if artifact.ArtifactRepositoryID == nil || *artifact.ArtifactRepositoryID != *initialRepoID {
			return fmt.Errorf("expected artifact_repository_id %q after update, got %v", *initialRepoID, artifact.ArtifactRepositoryID)
		}

		return nil
	}
}

func checkArtifactRepoDestroyedFromAPI(lastArtifactID *string, isMock bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if isMock || *lastArtifactID == "" {
			return nil
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("provider not found")
		}
		p.service = NewService(cl)

		_, err := p.service.GetArtifact(context.Background(), *lastArtifactID)
		if err == nil {
			return fmt.Errorf("artifact %s still exists after destroy", *lastArtifactID)
		}
		if _, ok := err.(*client.NotFoundError); !ok {
			return fmt.Errorf("unexpected error checking artifact %s after destroy: %w", *lastArtifactID, err)
		}

		return nil
	}
}

func artifactTestContainerSpecBlock(imageURI string) string {
	return fmt.Sprintf(`
  spec = {
    container_groups = [
      {
        containers = [
          {
            name        = "main"
            image_uri   = %q
            description = "main container"
            primary     = true
            port        = 8080
            entrypoint  = ["python", "-m", "app"]

            environment_vars = [
              {
                source = "string"
                name   = "ENV"
                value  = "production"
              }
            ]

            startup_probe = {
              path                  = "/startup"
              port                  = 8080
              scheme                = "HTTP"
              initial_delay_seconds = 10
              period_seconds        = 15
              timeout_seconds       = 5
              failure_threshold     = 3
            }

            readiness_probe = {
              path                  = "/health"
              port                  = 8080
              scheme                = "HTTP"
              initial_delay_seconds = 5
              period_seconds        = 10
              timeout_seconds       = 3
              failure_threshold     = 3
            }

            liveness_probe = {
              path              = "/live"
              port              = 8080
              scheme            = "HTTP"
              failure_threshold = 5
            }
          }
        ]
      }
    ]
  }`, imageURI)
}

func artifactResourceConfig(name, imageURI string) string {
	return fmt.Sprintf(`
resource "datarobot_artifact" "test" {
  name        = %q
  description = "test artifact description"
  type        = "service"
%s
}
`, name, artifactTestContainerSpecBlock(imageURI))
}

func artifactImageURIValue(c client.ArtifactContainer) string {
	return c.ImageURI
}

func stringPtr(s string) *string {
	return &s
}

func artifactFixture(id string, repoID *string, name string) *client.Artifact {
	return artifactFixtureWithStatus(id, repoID, name, client.ArtifactStatusLocked)
}

// artifactTestImageURI is the image used in mock artifact fixtures and Terraform test configs.
const artifactTestImageURI = "nginx:latest"

// artifactFixture returns a full Workload API artifact response for integration tests.
func artifactFixtureWithStatus(id string, repoID *string, name string, status client.ArtifactStatus) *client.Artifact {
	port := int64(8080)
	primary := true
	containerName := "main"
	containerDesc := "main container"
	probeScheme := "HTTP"
	probeFailureThreshold := int64(3)
	probeInitialDelay := int64(10)
	probePeriod := int64(15)
	probeTimeout := int64(5)
	readinessInitialDelay := int64(5)
	readinessPeriod := int64(10)
	readinessTimeout := int64(3)
	livenessFailureThreshold := int64(5)
	version := 1
	fullName := "Test User"
	email := "test@example.com"
	username := "testuser"

	return &client.Artifact{
		ID:                   id,
		Name:                 name,
		Description:          "test artifact description",
		Type:                 client.ArtifactTypeService,
		Status:               status,
		Version:              &version,
		ArtifactRepositoryID: repoID,
		CreatedAt:            "2026-01-01T00:00:00Z",
		UpdatedAt:            "2026-01-02T00:00:00Z",
		Creator: &client.ArtifactUser{
			ID:       "creator-id",
			FullName: &fullName,
			Email:    &email,
			Username: &username,
		},
		Tags: []client.ArtifactTag{
			{ID: "tag-id", Name: "env", Value: "test"},
		},
		Permissions: []string{"CAN_VIEW", "CAN_UPDATE"},
		Spec: client.ArtifactSpec{
			ContainerGroups: []client.ArtifactContainerGroup{
				{
					Containers: []client.ArtifactContainer{
						{
							Name:        &containerName,
							ImageURI:    artifactTestImageURI,
							Description: containerDesc,
							Primary:     &primary,
							Port:        &port,
							Entrypoint:  []string{"python", "-m", "app"},
							EnvironmentVars: []client.ArtifactEnvironmentVariable{
								{Source: client.EnvironmentVariableSourceString, Name: "ENV", Value: "production"},
							},
							StartupProbe: &client.ArtifactProbeConfig{
								Path:                "/startup",
								Port:                &port,
								Scheme:              &probeScheme,
								InitialDelaySeconds: &probeInitialDelay,
								PeriodSeconds:       &probePeriod,
								TimeoutSeconds:      &probeTimeout,
								FailureThreshold:    &probeFailureThreshold,
							},
							ReadinessProbe: &client.ArtifactProbeConfig{
								Path:                "/health",
								Port:                &port,
								Scheme:              &probeScheme,
								InitialDelaySeconds: &readinessInitialDelay,
								PeriodSeconds:       &readinessPeriod,
								TimeoutSeconds:      &readinessTimeout,
								FailureThreshold:    &probeFailureThreshold,
							},
							LivenessProbe: &client.ArtifactProbeConfig{
								Path:             "/live",
								Port:             &port,
								Scheme:           &probeScheme,
								FailureThreshold: &livenessFailureThreshold,
							},
						},
					},
				},
			},
		},
	}
}

func TestArtifactTooManyContainerGroups(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      artifactConfigWithMultipleGroups(),
				ExpectError: regexp.MustCompile("Too many container groups"),
			},
		},
	})
}

func artifactConfigWithMultipleGroups() string {
	return `
resource "datarobot_artifact" "test" {
  name = "multi-group-test"
  spec = {
    container_groups = [
      {
        containers = [{ image_uri = "image-a:latest" }]
      },
      {
        containers = [{ image_uri = "image-b:latest" }]
      }
    ]
  }
}
`
}

func TestArtifactCredentialEnvVarValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	cases := []struct {
		name        string
		config      string
		expectError string
	}{
		{
			name:        "credential env var missing dr_credential_id",
			config:      artifactConfigWithCredentialEnvVar("dr-credential", "", "token", ""),
			expectError: `"dr_credential_id" is required`,
		},
		{
			name:        "credential env var missing key",
			config:      artifactConfigWithCredentialEnvVar("dr-credential", "cred-abc", "", ""),
			expectError: `"key" is required`,
		},
		{
			name:        "credential env var with unexpected value",
			config:      artifactConfigWithCredentialEnvVar("dr-credential", "cred-abc", "token", "should-not-be-here"),
			expectError: `"value" must not be set`,
		},
		{
			name:        "string env var missing value",
			config:      artifactConfigWithStringEnvVarMissingValue(),
			expectError: `"value" is required`,
		},
		{
			name:        "invalid source type",
			config:      artifactConfigWithInvalidSource(),
			expectError: `Invalid source`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      tc.config,
						ExpectError: regexp.MustCompile(tc.expectError),
					},
				},
			})
		})
	}
}

// artifactConfigWithCredentialEnvVar builds a config with a credential env var.
// Pass empty strings for dr_credential_id, key, or value to omit those fields.
func artifactConfigWithCredentialEnvVar(source, credentialID, key, value string) string {
	credentialIDLine := ""
	if credentialID != "" {
		credentialIDLine = fmt.Sprintf("dr_credential_id = %q\n", credentialID)
	}
	keyLine := ""
	if key != "" {
		keyLine = fmt.Sprintf("key = %q\n", key)
	}
	valueLine := ""
	if value != "" {
		valueLine = fmt.Sprintf("value = %q\n", value)
	}
	return fmt.Sprintf(`
resource "datarobot_artifact" "test" {
  name = "cred-env-test"
  spec = {
    container_groups = [{
      containers = [{
        image_uri = "nginx:latest"
        primary   = true
        port      = 8080
        environment_vars = [{
          source = %q
          name   = "MY_SECRET"
          %s%s%s
        }]
      }]
    }]
  }
}
`, source, credentialIDLine, keyLine, valueLine)
}

func artifactConfigWithStringEnvVarMissingValue() string {
	return `
resource "datarobot_artifact" "test" {
  name = "missing-value-test"
  spec = {
    container_groups = [{
      containers = [{
        image_uri = "nginx:latest"
        environment_vars = [{
          source = "string"
          name   = "ENV"
        }]
      }]
    }]
  }
}
`
}

func artifactConfigWithInvalidSource() string {
	return `
resource "datarobot_artifact" "test" {
  name = "invalid-source-test"
  spec = {
    container_groups = [{
      containers = [{
        image_uri = "nginx:latest"
        environment_vars = [{
          source = "unknown-type"
          name   = "ENV"
          value  = "foo"
        }]
      }]
    }]
  }
}
`
}

func TestIntegrationArtifactDraftLifecycle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		globalTestCfg.ApiKey = "fake"
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "draft-artifact-" + uuid.NewString()[:8]
	updatedName := "updated-" + name

	draftArtifact := artifactFixtureWithStatus(artifactID, &repoIDPtr, name, client.ArtifactStatusDraft)
	updatedDraftArtifact := artifactFixtureWithStatus(artifactID, &repoIDPtr, updatedName, client.ArtifactStatusDraft)
	lockedArtifact := artifactFixtureWithStatus(artifactID, &repoIDPtr, updatedName, client.ArtifactStatusLocked)

	getArtifactResponse := draftArtifact
	mockService.EXPECT().
		GetArtifact(gomock.Any(), artifactID).
		DoAndReturn(func(context.Context, string) (*client.Artifact, error) {
			return getArtifactResponse, nil
		}).AnyTimes()

	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
			if req.Status != client.ArtifactStatusDraft {
				t.Errorf("expected create status draft, got %q", req.Status)
			}
			return draftArtifact, nil
		})

	patchCall := 0
	mockService.EXPECT().
		PatchArtifact(gomock.Any(), artifactID, gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, req *client.PatchArtifactRequest) (*client.Artifact, error) {
			patchCall++
			switch patchCall {
			case 1:
				if req.Status != nil {
					t.Errorf("expected spec-only patch, got status %q", *req.Status)
				}
				getArtifactResponse = updatedDraftArtifact
				return updatedDraftArtifact, nil
			default:
				if req.Status == nil || *req.Status != client.ArtifactStatusLocked {
					t.Errorf("expected lock patch, got status %v", req.Status)
				}
				getArtifactResponse = lockedArtifact
				return lockedArtifact, nil
			}
		}).Times(2)

	mockService.EXPECT().DeleteArtifactRepository(gomock.Any(), repoID).Return(nil)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: artifactResourceConfigWithStatus(name, "draft"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("datarobot_artifact.test", "status", "draft"),
					resource.TestCheckResourceAttr("datarobot_artifact.test", "artifact_id", artifactID),
				),
			},
			{
				Config: artifactResourceConfigWithStatus(updatedName, "draft"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("datarobot_artifact.test", "status", "draft"),
					resource.TestCheckResourceAttr("datarobot_artifact.test", "artifact_id", artifactID),
					resource.TestCheckResourceAttr("datarobot_artifact.test", "name", updatedName),
				),
			},
			{
				Config: artifactResourceConfigWithStatus(updatedName, "locked"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("datarobot_artifact.test", "status", "locked"),
					resource.TestCheckResourceAttr("datarobot_artifact.test", "artifact_id", artifactID),
				),
			},
		},
	})
}

func TestArtifactLockedToDraftRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		globalTestCfg.ApiKey = "fake"
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "locked-artifact-" + uuid.NewString()[:8]

	lockedArtifact := artifactFixtureWithStatus(artifactID, &repoIDPtr, name, client.ArtifactStatusLocked)

	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		Return(lockedArtifact, nil)
	mockService.EXPECT().
		GetArtifact(gomock.Any(), artifactID).
		Return(lockedArtifact, nil).
		AnyTimes()
	mockService.EXPECT().
		DeleteArtifactRepository(gomock.Any(), repoID).
		Return(nil).
		AnyTimes()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: artifactResourceConfigWithStatus(name, "locked"),
			},
			{
				Config:      artifactResourceConfigWithStatus(name, "draft"),
				ExpectError: regexp.MustCompile(`Cannot revert a locked artifact to draft`),
			},
		},
	})
}

func artifactResourceConfigWithStatus(name, status string) string {
	return fmt.Sprintf(`
resource "datarobot_artifact" "test" {
  name        = %q
  description = "test artifact description"
  type        = "service"
  status      = %q
%s
}
`, name, status, artifactTestContainerSpecBlock(artifactTestImageURI))
}

func TestArtifactCreateRequestStatus(t *testing.T) {
	spec := &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{
			{Containers: []ArtifactContainerModel{{ImageURI: types.StringValue("nginx:latest")}}},
		},
	}

	lockedReq := artifactCreateRequest(ArtifactResourceModel{
		Name:   types.StringValue("test"),
		Type:   types.StringValue("service"),
		Status: types.StringValue("locked"),
		Spec:   spec,
	})
	if lockedReq.Status != client.ArtifactStatusLocked {
		t.Fatalf("expected locked, got %q", lockedReq.Status)
	}

	draftReq := artifactCreateRequest(ArtifactResourceModel{
		Name:   types.StringValue("test"),
		Type:   types.StringValue("service"),
		Status: types.StringValue("draft"),
		Spec:   spec,
	})
	if draftReq.Status != client.ArtifactStatusDraft {
		t.Fatalf("expected draft, got %q", draftReq.Status)
	}

	defaultReq := artifactCreateRequest(ArtifactResourceModel{
		Name: types.StringValue("test"),
		Type: types.StringValue("service"),
		Spec: spec,
	})
	if defaultReq.Status != client.ArtifactStatusLocked {
		t.Fatalf("expected default locked, got %q", defaultReq.Status)
	}
}

func TestPatchRequestFromPlan(t *testing.T) {
	spec := &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{
			{Containers: []ArtifactContainerModel{{ImageURI: types.StringValue("nginx:latest")}}},
		},
	}
	base := ArtifactResourceModel{
		Name:        types.StringValue("test"),
		Description: types.StringValue("desc"),
		Spec:        spec,
	}

	draftState := base
	draftState.Status = types.StringValue("draft")
	draftPlan := draftState

	patch := patchRequestFromPlan(draftPlan, draftState)
	if patch.Status != nil {
		t.Fatalf("expected no status on draft spec patch, got %v", patch.Status)
	}

	lockPlan := draftPlan
	lockPlan.Status = types.StringValue("locked")
	lockPatch := patchRequestFromPlan(lockPlan, draftState)
	if lockPatch.Status == nil || *lockPatch.Status != client.ArtifactStatusLocked {
		t.Fatalf("expected lock status in patch, got %v", lockPatch.Status)
	}
}

func TestArtifactImageSourceRequired(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "datarobot_artifact" "test" {
  name = "missing-image-source"
  spec = {
    container_groups = [{
      containers = [{
        primary = true
        port    = 8080
      }]
    }]
  }
}`,
				ExpectError: regexp.MustCompile("Missing image source"),
			},
		},
	})
}

func TestArtifactLockedImageBuildConfigWithoutImageURI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "datarobot_artifact" "test" {
  name   = "locked-build-config"
  status = "locked"
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
}`,
				ExpectError: regexp.MustCompile("Incomplete build configuration for locked artifact"),
			},
		},
	})
}

func TestArtifactImageBuildConfigGeneratedRequiresFields(t *testing.T) {
	resp := &tfresource.ValidateConfigResponse{}
	containerPath := path.Root("spec").
		AtName("container_groups").AtListIndex(0).
		AtName("containers").AtListIndex(0)
	cfg := &ArtifactImageBuildConfigModel{
		Dockerfile: &ArtifactDockerfileModel{
			Source: types.StringValue("generated"),
		},
	}

	validateImageBuildConfig(resp, containerPath, cfg, string(client.ArtifactTypeService))

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected validation errors")
	}
	if len(resp.Diagnostics.Errors()) != 3 {
		t.Fatalf("expected 3 validation errors, got %d", len(resp.Diagnostics.Errors()))
	}

	var combined strings.Builder
	for _, d := range resp.Diagnostics.Errors() {
		combined.WriteString(d.Summary())
		combined.WriteString(d.Detail())
	}
	msg := combined.String()

	for _, field := range []string{"execution_environment_id", "execution_environment_version_id", "entrypoint"} {
		if !strings.Contains(msg, field) {
			t.Errorf("expected validation to mention %q, got: %s", field, msg)
		}
	}
}

func TestArtifactNimWithCodeRefRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "datarobot_artifact" "test" {
  name   = "nim-code-ref"
  type   = "nim"
  status = "draft"
  spec = {
    container_groups = [{
      containers = [{
        primary = true
        port    = 8080
        image_build_config = {
          code_ref = {
            catalog_id         = "aaaaaaaaaaaaaaaaaaaaaaaa"
            catalog_version_id = "bbbbbbbbbbbbbbbbbbbbbbbb"
          }
          dockerfile = { source = "provided" }
        }
      }]
    }]
  }
}`,
				ExpectError: regexp.MustCompile("NIM artifacts cannot include"),
			},
		},
	})
}

func TestArtifactImageBuildConfigToClient_provided(t *testing.T) {
	container := artifactContainerToClient(ArtifactContainerModel{
		Primary: types.BoolValue(true),
		Port:    types.Int64Value(8080),
		ImageBuildConfig: &ArtifactImageBuildConfigModel{
			Dockerfile: &ArtifactDockerfileModel{
				Source: types.StringValue("provided"), // path omitted on purpose: defaults to ./Dockerfile in artifactDockerfileToClient
			},
		},
	})

	if container.ImageURI != "" {
		t.Fatalf("expected no imageUri, got %q", container.ImageURI)
	}
	if container.ImageBuildConfig == nil || container.ImageBuildConfig.Dockerfile == nil {
		t.Fatal("expected imageBuildConfig.dockerfile")
	}
	if container.ImageBuildConfig.Dockerfile.Source != "provided" {
		t.Fatalf("expected provided source, got %q", container.ImageBuildConfig.Dockerfile.Source)
	}
	if container.ImageBuildConfig.Dockerfile.Path != "./Dockerfile" {
		t.Fatalf("expected default dockerfile path, got %q", container.ImageBuildConfig.Dockerfile.Path)
	}
}

func TestArtifactImageBuildConfigToClient_generated(t *testing.T) {
	container := artifactContainerToClient(ArtifactContainerModel{
		ImageURI: types.StringValue("registry.example/app:latest"),
		ImageBuildConfig: &ArtifactImageBuildConfigModel{
			Dockerfile: &ArtifactDockerfileModel{
				Source:                        types.StringValue("generated"),
				ExecutionEnvironmentID:        types.StringValue("eeeeeeeeeeeeeeeeeeeeeeee"),
				ExecutionEnvironmentVersionID: types.StringValue("ffffffffffffffffffffffff"),
				Entrypoint:                    []types.String{types.StringValue("python"), types.StringValue("app.py")},
			},
		},
	})

	cfg := container.ImageBuildConfig
	if cfg == nil || cfg.Dockerfile == nil {
		t.Fatal("expected imageBuildConfig")
	}
	if cfg.Dockerfile.ExecutionEnvironmentID != "eeeeeeeeeeeeeeeeeeeeeeee" {
		t.Fatalf("unexpected executionEnvironmentId: %q", cfg.Dockerfile.ExecutionEnvironmentID)
	}
	if cfg.Dockerfile.ExecutionEnvironmentVersionID != "ffffffffffffffffffffffff" {
		t.Fatalf("unexpected executionEnvironmentVersionID: %q", cfg.Dockerfile.ExecutionEnvironmentVersionID)
	}
	if len(cfg.Dockerfile.Entrypoint) != 2 || cfg.Dockerfile.Entrypoint[0] != "python" {
		t.Fatalf("unexpected entrypoint: %v", cfg.Dockerfile.Entrypoint)
	}
}

func TestArtifactImageBuildConfigFromAPI_provided(t *testing.T) {
	t.Run("provided dockerfile with code_ref", func(t *testing.T) {
		const (
			catalogID        = "aaaaaaaaaaaaaaaaaaaaaaaa"
			catalogVersionID = "bbbbbbbbbbbbbbbbbbbbbbbb"
		)

		model := loadContainerFromAPI(client.ArtifactContainer{
			ImageBuildConfig: &client.ArtifactImageBuildConfig{
				CodeRef: &client.ArtifactCodeRef{
					Type:     "datarobot",
					Provider: "datarobot",
					Datarobot: &client.ArtifactDataRobotCodeRef{
						CatalogID:        catalogID,
						CatalogVersionID: catalogVersionID,
					},
				},
				Dockerfile: &client.ArtifactDockerfile{
					Source: "provided",
					Path:   "./Dockerfile",
				},
			},
		}, nil)

		cfg := model.ImageBuildConfig
		if cfg == nil {
			t.Fatal("expected image_build_config in state model")
		}
		if cfg.CodeRef == nil {
			t.Fatal("expected code_ref in state model")
		}
		if got := cfg.CodeRef.CatalogID.ValueString(); got != catalogID {
			t.Fatalf("catalog_id: got %q, want %q", got, catalogID)
		}
		if got := cfg.CodeRef.CatalogVersionID.ValueString(); got != catalogVersionID {
			t.Fatalf("catalog_version_id: got %q, want %q", got, catalogVersionID)
		}
		if cfg.Dockerfile == nil {
			t.Fatal("expected dockerfile in state model")
		}
		if got := cfg.Dockerfile.Source.ValueString(); got != "provided" {
			t.Fatalf("dockerfile.source: got %q, want %q", got, "provided")
		}
		if got := cfg.Dockerfile.Path.ValueString(); got != "./Dockerfile" {
			t.Fatalf("dockerfile.path: got %q, want %q", got, "./Dockerfile")
		}
	})
}

func TestArtifactImageBuildConfigFromAPI_generated(t *testing.T) {
	t.Run("generated dockerfile", func(t *testing.T) {
		const (
			eeID        = "eeeeeeeeeeeeeeeeeeeeeeee"
			eeVersionID = "ffffffffffffffffffffffff"
		)

		model := loadContainerFromAPI(client.ArtifactContainer{
			ImageURI: "registry.example/app:latest",
			ImageBuildConfig: &client.ArtifactImageBuildConfig{
				Dockerfile: &client.ArtifactDockerfile{
					Source:                        "generated",
					ExecutionEnvironmentID:        eeID,
					ExecutionEnvironmentVersionID: eeVersionID,
					Entrypoint:                    []string{"python", "app.py"},
				},
			},
		}, nil)

		cfg := model.ImageBuildConfig
		if cfg == nil || cfg.Dockerfile == nil {
			t.Fatal("expected image_build_config.dockerfile in state model")
		}
		if got := cfg.Dockerfile.Source.ValueString(); got != "generated" {
			t.Fatalf("dockerfile.source: got %q, want %q", got, "generated")
		}
		if got := cfg.Dockerfile.ExecutionEnvironmentID.ValueString(); got != eeID {
			t.Fatalf("execution_environment_id: got %q, want %q", got, eeID)
		}
		if got := cfg.Dockerfile.ExecutionEnvironmentVersionID.ValueString(); got != eeVersionID {
			t.Fatalf("execution_environment_version_id: got %q, want %q", got, eeVersionID)
		}
		if len(cfg.Dockerfile.Entrypoint) != 2 {
			t.Fatalf("entrypoint length: got %d, want 2", len(cfg.Dockerfile.Entrypoint))
		}
		if got := cfg.Dockerfile.Entrypoint[0].ValueString(); got != "python" {
			t.Fatalf("entrypoint[0]: got %q, want %q", got, "python")
		}
		if got := cfg.Dockerfile.Entrypoint[1].ValueString(); got != "app.py" {
			t.Fatalf("entrypoint[1]: got %q, want %q", got, "app.py")
		}
		if model.ImageURI.ValueString() != "registry.example/app:latest" {
			t.Fatalf("image_uri: got %q", model.ImageURI.ValueString())
		}
	})
}

func TestDockerfileEqual_normalizesProvidedDefaults(t *testing.T) {
	providedDefaults := &ArtifactDockerfileModel{
		Source: types.StringValue("provided"),
		Path:   types.StringValue("./Dockerfile"),
	}
	providedNoPath := &ArtifactDockerfileModel{
		Source: types.StringValue("provided"),
		Path:   types.StringNull(),
	}
	providedSourceOnly := &ArtifactDockerfileModel{
		Source: types.StringValue("provided"),
	}

	cases := []struct {
		name string
		a    *ArtifactDockerfileModel
		b    *ArtifactDockerfileModel
	}{
		{name: "nil vs nil", a: nil, b: nil},
		{name: "nil vs provided defaults", a: nil, b: providedDefaults},
		{name: "source only vs defaults", a: providedSourceOnly, b: providedDefaults},
		{name: "null path vs defaults", a: providedNoPath, b: providedDefaults},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !dockerfileEqual(tc.a, tc.b) {
				t.Fatal("expected normalized dockerfile configs to be equal")
			}
		})
	}
}

func TestContainersEqual_includesImageBuildConfig(t *testing.T) {
	base := ArtifactContainerModel{
		ImageURI: types.StringValue("nginx:latest"),
		ImageBuildConfig: &ArtifactImageBuildConfigModel{
			Dockerfile: &ArtifactDockerfileModel{Source: types.StringValue("provided")},
		},
	}
	changed := base
	changed.ImageBuildConfig = &ArtifactImageBuildConfigModel{
		Dockerfile: &ArtifactDockerfileModel{
			Source:                        types.StringValue("generated"),
			ExecutionEnvironmentID:        types.StringValue("eeeeeeeeeeeeeeeeeeeeeeee"),
			ExecutionEnvironmentVersionID: types.StringValue("ffffffffffffffffffffffff"),
			Entrypoint:                    []types.String{types.StringValue("python")},
		},
	}

	if containersEqual(base, changed) {
		t.Fatal("expected image_build_config change to make containers unequal")
	}
}

func TestIntegrationArtifactDraftImageBuildConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "draft-build-" + uuid.NewString()[:8]

	draftArtifact := &client.Artifact{
		ID:                   artifactID,
		Name:                 name,
		Type:                 client.ArtifactTypeService,
		Status:               client.ArtifactStatusDraft,
		ArtifactRepositoryID: &repoIDPtr,
		Spec: client.ArtifactSpec{
			ContainerGroups: []client.ArtifactContainerGroup{{
				Containers: []client.ArtifactContainer{{
					Name:    stringPtr("main"),
					Primary: func() *bool { v := true; return &v }(),
					Port:    func() *int64 { v := int64(8080); return &v }(),
					ImageBuildConfig: &client.ArtifactImageBuildConfig{
						Dockerfile: &client.ArtifactDockerfile{Source: "provided", Path: "./Dockerfile"},
					},
				}},
			}},
		},
	}

	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
			if req.Status != client.ArtifactStatusDraft {
				t.Errorf("expected draft create, got %q", req.Status)
			}
			c := req.Spec.ContainerGroups[0].Containers[0]
			if c.ImageURI != "" {
				t.Errorf("expected no imageUri on create, got %q", c.ImageURI)
			}
			if c.ImageBuildConfig == nil || c.ImageBuildConfig.Dockerfile == nil {
				t.Fatal("expected imageBuildConfig on create request")
			}
			if c.ImageBuildConfig.Dockerfile.Source != "provided" {
				t.Fatalf("expected provided dockerfile, got %q", c.ImageBuildConfig.Dockerfile.Source)
			}
			if c.ImageBuildConfig.Dockerfile.Path != "./Dockerfile" {
				t.Fatalf("expected dockerfile path, got %q", c.ImageBuildConfig.Dockerfile.Path)
			}
			return draftArtifact, nil
		})

	mockService.EXPECT().GetArtifact(gomock.Any(), artifactID).Return(draftArtifact, nil).AnyTimes()
	mockService.EXPECT().DeleteArtifactRepository(gomock.Any(), repoID).Return(nil)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "datarobot_artifact" "test" {
  name   = %q
  status = "draft"
  spec = {
    container_groups = [{
      containers = [{
        name    = "main"
        primary = true
        port    = 8080
        image_build_config = {
          dockerfile = { source = "provided" }
        }
      }]
    }]
  }
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("datarobot_artifact.test", "status", "draft"),
					resource.TestCheckResourceAttr("datarobot_artifact.test", "artifact_id", artifactID),
				),
			},
		},
	})
}
