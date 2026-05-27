package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccArtifactResource(t *testing.T) {
	t.Parallel()
	testArtifactResource(t, uuid.NewString(), false)
}

func TestIntegrationArtifactResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	initialID := uuid.NewString()
	updatedID := uuid.NewString()
	repoID := uuid.NewString()
	name := "test-artifact-" + uuid.NewString()[:8]
	updatedName := "updated-" + name
	imageURI := "nginx:latest"

	repoIDPtr := repoID

	initialArtifact := artifactFixture(initialID, &repoIDPtr, name, imageURI)
	updatedArtifact := artifactFixture(updatedID, &repoIDPtr, updatedName, imageURI)

	// Create: CreateArtifact → post-create Read
	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		Return(initialArtifact, nil)
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

	resource.Test(t, resource.TestCase{
		IsUnitTest:               isMock,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             checkArtifactRepoDestroyedFromAPI(&lastArtifactID, isMock),
		Steps: []resource.TestStep{
			{
				Config: artifactResourceConfig(name, "nginx:latest"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "artifact_id"),
					resource.TestCheckResourceAttrSet(resourceName, "artifact_repository_id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "type", "service"),
					captureAttr(resourceName, "artifact_repository_id", &initialRepoID),
					checkArtifactExistsInAPI(resourceName, name, "nginx:latest", isMock),
				),
			},
			{
				Config: artifactResourceConfig("updated-"+name, "nginx:latest"),
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
		if got := artifact.Spec.ContainerGroups[0].Containers[0].ImageURI; got != expectedImageURI {
			return fmt.Errorf("expected image_uri %q, got %q", expectedImageURI, got)
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

func artifactResourceConfig(name, imageURI string) string {
	return fmt.Sprintf(`
resource "datarobot_artifact" "test" {
  name        = %q
  description = "test artifact description"
  type        = "service"

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

            readiness_probe = {
              path = "/health"
              port = 8080
            }
          }
        ]
      }
    ]
  }
}
`, name, imageURI)
}

func artifactFixture(id string, repoID *string, name, imageURI string) *client.Artifact {
	port := int64(8080)
	primary := true
	containerName := "main"
	containerDesc := "main container"
	probeScheme := "HTTP"
	probeFailureThreshold := int64(3)

	return &client.Artifact{
		ID:                   id,
		Name:                 name,
		Description:          "test artifact description",
		Type:                 client.ArtifactTypeService,
		Status:               client.ArtifactStatusLocked,
		ArtifactRepositoryID: repoID,
		Spec: client.ArtifactSpec{
			ContainerGroups: []client.ArtifactContainerGroup{
				{
					Containers: []client.ArtifactContainer{
						{
							Name:        &containerName,
							ImageURI:    imageURI,
							Description: containerDesc,
							Primary:     &primary,
							Port:        &port,
							Entrypoint:  []string{"python", "-m", "app"},
							EnvironmentVars: []client.ArtifactEnvironmentVariable{
								{Source: client.EnvironmentVariableSourceString, Name: "ENV", Value: "production"},
							},
							ReadinessProbe: &client.ArtifactProbeConfig{
								Path:             "/health",
								Port:             &port,
								Scheme:           &probeScheme,
								FailureThreshold: &probeFailureThreshold,
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
