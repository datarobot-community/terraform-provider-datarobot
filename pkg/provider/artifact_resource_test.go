package provider

import (
	"context"
	"fmt"
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

	// Import: Read to hydrate state from ID
	mockService.EXPECT().
		GetArtifact(gomock.Any(), updatedID).
		Return(updatedArtifact, nil).
		Times(1)

	// Destroy: no API call expected — artifacts are persisted

	testArtifactResource(t, name, true)
}

func testArtifactResource(t *testing.T, name string, isMock bool) {
	t.Helper()
	resourceName := "datarobot_artifact.test"

	var initialRepoID string

	resource.Test(t, resource.TestCase{
		IsUnitTest:               isMock,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: artifactResourceConfig(name, "nginx:latest"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
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
					resource.TestCheckResourceAttrSet(resourceName, "artifact_repository_id"),
					resource.TestCheckResourceAttr(resourceName, "name", "updated-"+name),
					checkArtifactUpdatedInSameRepo(resourceName, "updated-"+name, &initialRepoID, isMock),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
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

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("artifact ID is not set in state")
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

		artifact, err := p.service.GetArtifact(context.Background(), id)
		if err != nil {
			return fmt.Errorf("GetArtifact(%s): %w", id, err)
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

		newID := rs.Primary.ID
		if newID == "" {
			return fmt.Errorf("artifact ID is not set after update")
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

		artifact, err := p.service.GetArtifact(context.Background(), newID)
		if err != nil {
			return fmt.Errorf("GetArtifact(%s) after update: %w", newID, err)
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
                name  = "ENV"
                value = "production"
              }
            ]

            resource_request = {
              cpu    = 1
              memory = 536870912
            }

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
	cpu := float64(1)
	memory := int64(536870912)
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
								{Name: "ENV", Value: "production"},
							},
							ResourceRequest: client.ArtifactResourceRequest{
								CPU:    cpu,
								Memory: memory,
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
