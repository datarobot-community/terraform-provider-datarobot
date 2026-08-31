package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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

// TestAccArtifactSourceUploadWithBuild supersedes the former TestAccArtifactSourceUpload:
// the same upload flow, plus the build assertions. Running both meant two real image builds
// for one set of coverage.
func TestAccArtifactSourceUploadWithBuild(t *testing.T) {
	t.Parallel()
	testArtifactSourceUpload(t, false, true)
}

// TestAccArtifactLockedSourceWithBuild supersedes the former TestAccArtifactSourceUploadLocked,
// which had to be skipped: workload-api returns 422 when locking with image_build_config but
// no image_uri. The provider now triggers the build and waits, so image_uri is populated
// before the lock and the locked flow is covered here.
func TestAccArtifactLockedSourceWithBuild(t *testing.T) {
	t.Parallel()
	testArtifactSourceUploadLocked(t, false, true)
}

func TestIntegrationArtifactSourceUpload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

	testArtifactSourceUpload(t, true, false, mockService)
}

func TestIntegrationArtifactSourceUploadLocked(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

	testArtifactSourceUploadLocked(t, true, true, mockService)
}

func TestIntegrationArtifactResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

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
					checkArtifactStatusInAPI("draft", false),
					checkArtifactExistsInAPI(resourceName, name, imageURI, false),
				),
			},
			{
				Config: artifactResourceConfigWithStatus(updatedName, "draft"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "status", "draft"),
					checkArtifactIDEquals(resourceName, &artifactID),
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					checkArtifactStatusInAPI("draft", false),
					checkArtifactExistsInAPI(resourceName, updatedName, imageURI, false),
					captureAttr(resourceName, "artifact_id", &lastArtifactID),
				),
			},
			{
				Config: artifactResourceConfigWithStatus(updatedName, "locked"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "status", "locked"),
					checkArtifactIDEquals(resourceName, &artifactID),
					checkArtifactStatusInAPI("locked", false),
					captureAttr(resourceName, "artifact_id", &lastArtifactID),
				),
			},
		},
	})
}

func artifactCodeRefVersionAttr() string {
	return "spec.container_groups.0.containers.0.image_build_config.code_ref.catalog_version_id"
}

func artifactCodeRefCatalogAttr() string {
	return "spec.container_groups.0.containers.0.image_build_config.code_ref.catalog_id"
}

func artifactImageURIAttr() string {
	return "spec.container_groups.0.containers.0.image_uri"
}

func artifactBuildIDAttr() string {
	return "spec.container_groups.0.containers.0.build.artifact_image_build_id"
}

func artifactBuildCheckFuncs(resourceName string, isMock, enabled bool) []resource.TestCheckFunc {
	if !enabled {
		return nil
	}
	return []resource.TestCheckFunc{
		resource.TestCheckResourceAttrSet(resourceName, artifactImageURIAttr()),
		resource.TestCheckResourceAttrSet(resourceName, artifactBuildIDAttr()),
		checkArtifactImageBuiltInAPI(resourceName, isMock),
	}
}

func checkArtifactImageBuiltInAPI(resourceName string, isMock bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		imageURI := rs.Primary.Attributes[artifactImageURIAttr()]
		if imageURI == "" {
			return fmt.Errorf("image_uri is not set in state after build")
		}

		buildID := rs.Primary.Attributes[artifactBuildIDAttr()]
		if buildID == "" {
			return fmt.Errorf("build.artifact_image_build_id is not set in state after build")
		}

		if isMock {
			return nil
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

		if len(artifact.Spec.ContainerGroups) == 0 || len(artifact.Spec.ContainerGroups[0].Containers) == 0 {
			return fmt.Errorf("artifact has no containers")
		}
		container := artifact.Spec.ContainerGroups[0].Containers[0]
		if got := artifactImageURIValue(container); got == "" {
			return fmt.Errorf("image_uri not populated in API after build")
		}
		// build.* is asserted from Terraform state above: WAPI can leave container.build
		// empty or stale right after completion, which is why the provider pins it from
		// WaitForArtifactBuild. Asserting it here would race that same lag.

		return nil
	}
}

func checkArtifactSourceCatalogVersionChanged(resourceName string, previous *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if previous == nil || *previous == "" {
			return fmt.Errorf("previous catalog_version_id not captured")
		}

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		current := rs.Primary.Attributes[artifactCodeRefVersionAttr()]
		if current == "" {
			return fmt.Errorf("catalog_version_id is not set in state")
		}
		if current == *previous {
			return fmt.Errorf("catalog_version_id unchanged at %q after source update", current)
		}
		return nil
	}
}

func checkArtifactSourceCodeRefInAPI(isMock bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const resourceName = "datarobot_artifact.test"
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

		ref := client.ExtractCodeRef(artifact)
		if ref == nil || ref.CatalogID == "" || ref.CatalogVersionID == "" {
			return fmt.Errorf("expected code_ref on primary container in API")
		}

		stateCatalogID := rs.Primary.Attributes[artifactCodeRefCatalogAttr()]
		stateVersionID := rs.Primary.Attributes[artifactCodeRefVersionAttr()]
		if stateCatalogID != ref.CatalogID {
			return fmt.Errorf("state catalog_id %q != API %q", stateCatalogID, ref.CatalogID)
		}
		if stateVersionID != ref.CatalogVersionID {
			return fmt.Errorf("state catalog_version_id %q != API %q", stateVersionID, ref.CatalogVersionID)
		}

		return nil
	}
}

func testArtifactSourceUpload(t *testing.T, isMock bool, withBuildChecks bool, mockService ...*mock_client.MockService) {
	t.Helper()

	sourceDirV1 := writeArtifactSourceTree(t, map[string]string{
		"main.py":    "print('v1')",
		"Dockerfile": "FROM python:3.11-slim\nWORKDIR /app\nCOPY main.py .\nCMD [\"python\", \"main.py\"]\n",
	})
	sourceDirV2 := writeArtifactSourceTree(t, map[string]string{
		"main.py":    "print('v2')",
		"Dockerfile": "FROM python:3.11-slim\nWORKDIR /app\nCOPY main.py .\nCMD [\"python\", \"main.py\"]\n",
	})

	resourceName := "datarobot_artifact.test"
	name := "source-upload-" + uuid.NewString()[:8]

	var initialVersionID, artifactID, lastArtifactID string

	if isMock {
		if len(mockService) != 1 || mockService[0] == nil {
			t.Fatal("mock integration test requires a mock service")
		}
		svc := mockService[0]

		artifactID = uuid.NewString()
		repoID := uuid.NewString()
		repoIDPtr := repoID
		draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
		filesAPI := newSyncTestFilesAPI()

		var currentArtifact *client.Artifact
		patchCodeRef := func(_ context.Context, id, catalogID, versionID string) (*client.Artifact, error) {
			currentArtifact = artifactSourcePatchedArtifact(draftArtifact, catalogID, versionID)
			return currentArtifact, nil
		}

		svc.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil).Times(1)
		svc.EXPECT().FilesAPI().Return(filesAPI).Times(2)
		svc.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).DoAndReturn(patchCodeRef).Times(2)
		expectArtifactBuildAfterUploadFromLatest(svc, artifactID, &currentArtifact)
		expectArtifactBuildAfterUploadFromLatest(svc, artifactID, &currentArtifact)
		svc.EXPECT().PatchArtifact(gomock.Any(), artifactID, gomock.Any()).Return(draftArtifact, nil).Times(1)
		svc.EXPECT().GetArtifact(gomock.Any(), artifactID).DoAndReturn(
			func(_ context.Context, id string) (*client.Artifact, error) {
				if currentArtifact != nil {
					return artifactSourceBuiltForRead(currentArtifact), nil
				}
				return draftArtifact, nil
			},
		).AnyTimes()
		svc.EXPECT().DeleteArtifactRepository(gomock.Any(), repoID).Return(nil)
	}

	// Any non-mock run uploads source with image_build_config and a default
	// wait_for_build, so it needs the Image Build Service regardless of whether the
	// build attributes are asserted.
	preCheck := func() { testAccPreCheck(t) }
	if !isMock {
		preCheck = func() { testAccArtifactBuildPreCheck(t) }
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               isMock,
		PreCheck:                 preCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             checkArtifactRepoDestroyedFromAPI(&lastArtifactID, isMock),
		Steps: []resource.TestStep{
			{
				Config: artifactConfigWithSource(name, "draft", sourceDirV1),
				Check: resource.ComposeAggregateTestCheckFunc(
					append([]resource.TestCheckFunc{
						resource.TestCheckResourceAttr(resourceName, "status", "draft"),
						resource.TestCheckResourceAttrSet(resourceName, "artifact_id"),
						resource.TestCheckResourceAttrSet(resourceName, "source.dir_hash"),
						resource.TestCheckResourceAttrSet(resourceName, artifactCodeRefCatalogAttr()),
						resource.TestCheckResourceAttrSet(resourceName, artifactCodeRefVersionAttr()),
						captureAttr(resourceName, artifactCodeRefVersionAttr(), &initialVersionID),
						captureAttr(resourceName, "artifact_id", &artifactID),
						checkArtifactSourceCodeRefInAPI(isMock),
					}, artifactBuildCheckFuncs(resourceName, isMock, withBuildChecks)...)...,
				),
			},
			{
				Config: artifactConfigWithSource(name, "draft", sourceDirV2),
				Check: resource.ComposeAggregateTestCheckFunc(
					append([]resource.TestCheckFunc{
						checkArtifactSourceCatalogVersionChanged(resourceName, &initialVersionID),
						checkArtifactIDEquals(resourceName, &artifactID),
						checkArtifactSourceCodeRefInAPI(isMock),
						captureAttr(resourceName, "artifact_id", &lastArtifactID),
					}, artifactBuildCheckFuncs(resourceName, isMock, withBuildChecks)...)...,
				),
			},
		},
	})
}

func testArtifactSourceUploadLocked(t *testing.T, isMock bool, withBuildChecks bool, mockService ...*mock_client.MockService) {
	t.Helper()

	sourceDirV1 := writeArtifactSourceTree(t, map[string]string{
		"main.py":    "print('v1')",
		"Dockerfile": "FROM python:3.11-slim\nWORKDIR /app\nCOPY main.py .\nCMD [\"python\", \"main.py\"]\n",
	})
	sourceDirV2 := writeArtifactSourceTree(t, map[string]string{
		"main.py":    "print('v2')",
		"Dockerfile": "FROM python:3.11-slim\nWORKDIR /app\nCOPY main.py .\nCMD [\"python\", \"main.py\"]\n",
	})

	resourceName := "datarobot_artifact.test"
	name := "source-upload-locked-" + uuid.NewString()[:8]

	var initialVersionID, initialArtifactID, initialRepoID, lastArtifactID string

	if isMock {
		if len(mockService) != 1 || mockService[0] == nil {
			t.Fatal("mock integration test requires a mock service")
		}
		svc := mockService[0]

		draftArtifactID := uuid.NewString()
		lockedArtifactID := uuid.NewString()
		draftCloneID := uuid.NewString()
		newLockedArtifactID := uuid.NewString()
		repoID := uuid.NewString()
		repoIDPtr := repoID

		draftArtifact := artifactFixtureDraftWithBuildConfig(draftArtifactID, &repoIDPtr, name)
		draftClone := artifactFixtureDraftWithBuildConfig(draftCloneID, &repoIDPtr, name)
		filesAPI := newSyncTestFilesAPI()

		var latestArtifact *client.Artifact
		patchCodeRef := func(_ context.Context, id, catalogID, versionID string) (*client.Artifact, error) {
			var base *client.Artifact
			switch id {
			case draftArtifactID:
				base = draftArtifact
			case draftCloneID:
				base = draftClone
			default:
				t.Fatalf("unexpected PatchArtifactCodeRef artifact_id %q", id)
			}
			latestArtifact = artifactSourcePatchedArtifact(base, catalogID, versionID)
			return latestArtifact, nil
		}
		lockArtifact := func(lockedID string, status client.ArtifactStatus) func(context.Context, string, *client.PatchArtifactRequest) (*client.Artifact, error) {
			return func(_ context.Context, id string, req *client.PatchArtifactRequest) (*client.Artifact, error) {
				if req.Status == nil || *req.Status != status {
					t.Fatalf("expected lock patch on %q, got status %v", id, req.Status)
				}
				if latestArtifact == nil {
					t.Fatalf("expected patched artifact before lock on %q", id)
				}
				locked := *latestArtifact
				locked.ID = lockedID
				locked.Status = status
				latestArtifact = &locked
				return latestArtifact, nil
			}
		}

		gomock.InOrder(
			svc.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
					if req.Status != client.ArtifactStatusDraft {
						t.Fatalf("expected draft create before lock, got %q", req.Status)
					}
					return draftArtifact, nil
				}),
			svc.EXPECT().FilesAPI().Return(filesAPI),
			svc.EXPECT().PatchArtifactCodeRef(gomock.Any(), draftArtifactID, gomock.Any(), gomock.Any()).DoAndReturn(patchCodeRef),
		)
		expectArtifactBuildAfterUploadFromLatest(svc, draftArtifactID, &latestArtifact)
		svc.EXPECT().PatchArtifact(gomock.Any(), draftArtifactID, gomock.Any()).DoAndReturn(lockArtifact(lockedArtifactID, client.ArtifactStatusLocked))
		svc.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
				if req.Status != client.ArtifactStatusDraft {
					t.Fatalf("expected draft clone before lock, got %q", req.Status)
				}
				if req.ArtifactRepositoryID == nil || *req.ArtifactRepositoryID != repoID {
					t.Fatalf("expected repository %q, got %v", repoID, req.ArtifactRepositoryID)
				}
				return draftClone, nil
			})
		gomock.InOrder(
			svc.EXPECT().FilesAPI().Return(filesAPI),
			svc.EXPECT().PatchArtifactCodeRef(gomock.Any(), draftCloneID, gomock.Any(), gomock.Any()).DoAndReturn(patchCodeRef),
		)
		expectArtifactBuildAfterUploadFromLatest(svc, draftCloneID, &latestArtifact)
		svc.EXPECT().PatchArtifact(gomock.Any(), draftCloneID, gomock.Any()).DoAndReturn(lockArtifact(newLockedArtifactID, client.ArtifactStatusLocked))
		svc.EXPECT().GetArtifact(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, id string) (*client.Artifact, error) {
				if latestArtifact != nil && id == latestArtifact.ID {
					return artifactSourceBuiltForRead(latestArtifact), nil
				}
				if id == draftArtifactID {
					return artifactSourceBuiltForRead(draftArtifact), nil
				}
				if id == draftCloneID {
					return artifactSourceBuiltForRead(draftClone), nil
				}
				return nil, fmt.Errorf("GetArtifact(%s): not found", id)
			},
		).AnyTimes()
		svc.EXPECT().DeleteArtifactRepository(gomock.Any(), repoID).Return(nil)
	}

	// Any non-mock run uploads source with image_build_config and a default
	// wait_for_build, so it needs the Image Build Service regardless of whether the
	// build attributes are asserted.
	lockedPreCheck := func() { testAccPreCheck(t) }
	if !isMock {
		lockedPreCheck = func() { testAccArtifactBuildPreCheck(t) }
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               isMock,
		PreCheck:                 lockedPreCheck,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             checkArtifactRepoDestroyedFromAPI(&lastArtifactID, isMock),
		Steps: []resource.TestStep{
			{
				Config: artifactConfigWithSource(name, "locked", sourceDirV1),
				Check: resource.ComposeAggregateTestCheckFunc(
					append([]resource.TestCheckFunc{
						resource.TestCheckResourceAttr(resourceName, "status", "locked"),
						resource.TestCheckResourceAttrSet(resourceName, "artifact_id"),
						resource.TestCheckResourceAttrSet(resourceName, "artifact_repository_id"),
						resource.TestCheckResourceAttrSet(resourceName, "source.dir_hash"),
						resource.TestCheckResourceAttrSet(resourceName, artifactCodeRefCatalogAttr()),
						resource.TestCheckResourceAttrSet(resourceName, artifactCodeRefVersionAttr()),
						captureAttr(resourceName, artifactCodeRefVersionAttr(), &initialVersionID),
						captureAttr(resourceName, "artifact_id", &initialArtifactID),
						captureAttr(resourceName, "artifact_repository_id", &initialRepoID),
						checkArtifactStatusInAPI("locked", isMock),
						checkArtifactSourceCodeRefInAPI(isMock),
					}, artifactBuildCheckFuncs(resourceName, isMock, withBuildChecks)...)...,
				),
			},
			{
				Config: artifactConfigWithSource(name, "locked", sourceDirV2),
				Check: resource.ComposeAggregateTestCheckFunc(
					append([]resource.TestCheckFunc{
						resource.TestCheckResourceAttr(resourceName, "status", "locked"),
						checkArtifactSourceCatalogVersionChanged(resourceName, &initialVersionID),
						checkArtifactIDChanged(resourceName, &initialArtifactID),
						checkArtifactRepositoryIDEquals(resourceName, &initialRepoID),
						checkArtifactStatusInAPI("locked", isMock),
						checkArtifactSourceCodeRefInAPI(isMock),
						captureAttr(resourceName, "artifact_id", &lastArtifactID),
					}, artifactBuildCheckFuncs(resourceName, isMock, withBuildChecks)...)...,
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

func checkArtifactStatusInAPI(expectedStatus string, isMock bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const resourceName = "datarobot_artifact.test"
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

func checkArtifactIDChanged(resourceName string, previous *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if previous == nil || *previous == "" {
			return fmt.Errorf("previous artifact_id not captured")
		}

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		got := rs.Primary.Attributes["artifact_id"]
		if got == "" {
			return fmt.Errorf("artifact_id is not set in state")
		}
		if got == *previous {
			return fmt.Errorf("artifact_id unchanged at %q after locked source update", got)
		}
		return nil
	}
}

func checkArtifactRepositoryIDEquals(resourceName string, expected *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if expected == nil || *expected == "" {
			return fmt.Errorf("expected artifact_repository_id was not captured from a prior step")
		}

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		got := rs.Primary.Attributes["artifact_repository_id"]
		if got != *expected {
			return fmt.Errorf("artifact_repository_id: expected %q, got %q", *expected, got)
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
              },
              {
                source = "api-key"
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
              success_threshold     = 1
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
	probeSuccessThreshold := int64(1)
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
								{Source: client.EnvironmentVariableSourceAPIKey},
							},
							StartupProbe: &client.ArtifactProbeConfig{
								Path:                "/startup",
								Port:                &port,
								Scheme:              &probeScheme,
								InitialDelaySeconds: &probeInitialDelay,
								PeriodSeconds:       &probePeriod,
								TimeoutSeconds:      &probeTimeout,
								FailureThreshold:    &probeFailureThreshold,
								SuccessThreshold:    &probeSuccessThreshold,
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

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

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

func TestValidateArtifactEnvironmentVar(t *testing.T) {
	t.Parallel()

	evPath := path.Root("spec").
		AtName("container_groups").AtListIndex(0).
		AtName("containers").AtListIndex(0).
		AtName("environment_vars").AtListIndex(0)

	hasDetail := func(diags diag.Diagnostics, substr string) bool {
		for _, d := range diags.Errors() {
			if strings.Contains(d.Detail(), substr) {
				return true
			}
		}
		return false
	}

	t.Run("unknown string value defers validation", func(t *testing.T) {
		resp := &tfresource.ValidateConfigResponse{}
		validateArtifactEnvironmentVar(resp, evPath, ArtifactEnvironmentVariableModel{
			Source: types.StringValue(client.EnvironmentVariableSourceString),
			Name:   types.StringValue("DATAROBOT_API_TOKEN"),
			Value:  types.StringUnknown(),
		})
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors for unknown value, got: %v", resp.Diagnostics.Errors())
		}
	})

	t.Run("null string value is rejected", func(t *testing.T) {
		resp := &tfresource.ValidateConfigResponse{}
		validateArtifactEnvironmentVar(resp, evPath, ArtifactEnvironmentVariableModel{
			Source: types.StringValue(client.EnvironmentVariableSourceString),
			Name:   types.StringValue("ENV"),
			Value:  types.StringNull(),
		})
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected validation error for null value")
		}
		if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `"value" is required`) {
			t.Fatalf("unexpected error: %v", resp.Diagnostics.Errors()[0])
		}
	})

	t.Run("unknown credential fields defer validation", func(t *testing.T) {
		resp := &tfresource.ValidateConfigResponse{}
		validateArtifactEnvironmentVar(resp, evPath, ArtifactEnvironmentVariableModel{
			Source:         types.StringValue(client.EnvironmentVariableSourceCredential),
			Name:           types.StringValue("SECRET"),
			DrCredentialID: types.StringUnknown(),
			Key:            types.StringValue("token"),
		})
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors for unknown dr_credential_id, got: %v", resp.Diagnostics.Errors())
		}
	})

	t.Run("api-key without name is valid", func(t *testing.T) {
		resp := &tfresource.ValidateConfigResponse{}
		validateArtifactEnvironmentVar(resp, evPath, ArtifactEnvironmentVariableModel{
			Source: types.StringValue(client.EnvironmentVariableSourceAPIKey),
			Name:   types.StringNull(),
		})
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors for api-key source, got: %v", resp.Diagnostics.Errors())
		}
	})

	t.Run("api-key rejects value", func(t *testing.T) {
		resp := &tfresource.ValidateConfigResponse{}
		validateArtifactEnvironmentVar(resp, evPath, ArtifactEnvironmentVariableModel{
			Source: types.StringValue(client.EnvironmentVariableSourceAPIKey),
			Name:   types.StringNull(),
			Value:  types.StringValue("secret"),
		})
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected validation error for api-key with value")
		}
	})

	t.Run("unknown credential key defers validation", func(t *testing.T) {
		resp := &tfresource.ValidateConfigResponse{}
		validateArtifactEnvironmentVar(resp, evPath, ArtifactEnvironmentVariableModel{
			Source:         types.StringValue(client.EnvironmentVariableSourceCredential),
			Name:           types.StringValue("SECRET"),
			DrCredentialID: types.StringValue("cred-abc"),
			Key:            types.StringUnknown(),
		})
		if resp.Diagnostics.HasError() {
			t.Fatalf("expected no errors for unknown key, got: %v", resp.Diagnostics.Errors())
		}
	})

	// An unknown value defers only its own "required" check: a literal key is still
	// rejected, otherwise it would be silently dropped when mapped to the API.
	t.Run("unknown string value still reports unexpected key", func(t *testing.T) {
		resp := &tfresource.ValidateConfigResponse{}
		validateArtifactEnvironmentVar(resp, evPath, ArtifactEnvironmentVariableModel{
			Source: types.StringValue(client.EnvironmentVariableSourceString),
			Name:   types.StringValue("ENV"),
			Value:  types.StringUnknown(),
			Key:    types.StringValue("token"),
		})
		if !hasDetail(resp.Diagnostics, `"key" must not be set`) {
			t.Fatalf("expected unexpected-key error alongside unknown value, got: %v", resp.Diagnostics.Errors())
		}
		if hasDetail(resp.Diagnostics, `"value" is required`) {
			t.Fatalf("unknown value must not be reported as missing: %v", resp.Diagnostics.Errors())
		}
	})

	// An unknown dr_credential_id defers only its own "required" check: a missing key
	// is still reported, otherwise the provider would send an empty key to the API.
	t.Run("unknown credential id still reports its siblings", func(t *testing.T) {
		resp := &tfresource.ValidateConfigResponse{}
		validateArtifactEnvironmentVar(resp, evPath, ArtifactEnvironmentVariableModel{
			Source:         types.StringValue(client.EnvironmentVariableSourceCredential),
			Name:           types.StringValue("SECRET"),
			DrCredentialID: types.StringUnknown(),
			Key:            types.StringNull(),
			Value:          types.StringValue("literal"),
		})
		if !hasDetail(resp.Diagnostics, `"key" is required`) {
			t.Fatalf("expected missing-key error alongside unknown dr_credential_id, got: %v", resp.Diagnostics.Errors())
		}
		if !hasDetail(resp.Diagnostics, `"value" must not be set`) {
			t.Fatalf("expected unexpected-value error alongside unknown dr_credential_id, got: %v", resp.Diagnostics.Errors())
		}
		if hasDetail(resp.Diagnostics, `"dr_credential_id" is required`) {
			t.Fatalf("unknown dr_credential_id must not be reported as missing: %v", resp.Diagnostics.Errors())
		}
	})
}

func TestArtifactCredentialEnvVarValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

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
			name:        "string env var missing name",
			config:      artifactConfigWithStringEnvVarMissingName(),
			expectError: `"name" is required`,
		},
		{
			name:        "api-key env var with unexpected value",
			config:      artifactConfigWithAPIKeyEnvVar(`value = "should-not-be-here"`),
			expectError: `"value" must not be set`,
		},
		{
			name:        "api-key env var with unexpected dr_credential_id",
			config:      artifactConfigWithAPIKeyEnvVar(`dr_credential_id = "cred-abc"`),
			expectError: `"dr_credential_id" must not be set`,
		},
		{
			name:        "api-key env var with unexpected key",
			config:      artifactConfigWithAPIKeyEnvVar(`key = "token"`),
			expectError: `"key" must not be set`,
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

func artifactConfigWithStringEnvVarMissingName() string {
	return `
resource "datarobot_artifact" "test" {
  name = "missing-name-test"
  spec = {
    container_groups = [{
      containers = [{
        image_uri = "nginx:latest"
        environment_vars = [{
          source = "string"
          value  = "foo"
        }]
      }]
    }]
  }
}
`
}

// artifactConfigWithAPIKeyEnvVar builds a config with an api-key env var plus
// an extra attribute line that should be rejected by validation.

func artifactConfigWithAPIKeyEnvVar(extraLine string) string {
	return fmt.Sprintf(`
resource "datarobot_artifact" "test" {
  name = "api-key-env-test"
  spec = {
    container_groups = [{
      containers = [{
        image_uri = "nginx:latest"
        environment_vars = [{
          source = "api-key"
          %s
        }]
      }]
    }]
  }
}
`, extraLine)
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

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

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

func TestArtifactLockedToDraftCreatesNewDraft(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

	lockedArtifactID := uuid.NewString()
	draftArtifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "locked-artifact-" + uuid.NewString()[:8]

	lockedArtifact := artifactFixtureWithStatus(lockedArtifactID, &repoIDPtr, name, client.ArtifactStatusLocked)
	draftArtifact := artifactFixtureWithStatus(draftArtifactID, &repoIDPtr, name, client.ArtifactStatusDraft)

	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
			if req.Status != client.ArtifactStatusLocked {
				t.Fatalf("expected locked create status, got %q", req.Status)
			}
			return lockedArtifact, nil
		})
	mockService.EXPECT().
		GetArtifact(gomock.Any(), lockedArtifactID).
		Return(lockedArtifact, nil).
		AnyTimes()
	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
			if req.Status != client.ArtifactStatusDraft {
				t.Fatalf("expected draft create status, got %q", req.Status)
			}
			if req.ArtifactRepositoryID == nil || *req.ArtifactRepositoryID != repoID {
				t.Fatalf("expected artifact_repository_id %q, got %v", repoID, req.ArtifactRepositoryID)
			}
			return draftArtifact, nil
		})
	mockService.EXPECT().
		GetArtifact(gomock.Any(), draftArtifactID).
		Return(draftArtifact, nil).
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
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("datarobot_artifact.test", "status", "locked"),
					resource.TestCheckResourceAttr("datarobot_artifact.test", "artifact_id", lockedArtifactID),
				),
			},
			{
				Config: artifactResourceConfigWithStatus(name, "draft"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("datarobot_artifact.test", "status", "draft"),
					resource.TestCheckResourceAttr("datarobot_artifact.test", "artifact_id", draftArtifactID),
				),
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

	patch := patchRequestFromPlan(draftPlan, draftState, false)
	if patch.Status != nil {
		t.Fatalf("expected no status on draft spec patch, got %v", patch.Status)
	}

	lockPlan := draftPlan
	lockPlan.Status = types.StringValue("locked")
	lockPatch := patchRequestFromPlan(lockPlan, draftState, false)
	if lockPatch.Status == nil || *lockPatch.Status != client.ArtifactStatusLocked {
		t.Fatalf("expected lock status in patch, got %v", lockPatch.Status)
	}

	deferLockPatch := patchRequestFromPlan(lockPlan, draftState, true)
	if deferLockPatch.Status != nil {
		t.Fatalf("expected deferred lock to omit status, got %v", deferLockPatch.Status)
	}
}

func TestIntegrationArtifactInvalidStatus(t *testing.T) {
	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      artifactResourceConfigWithStatus("invalid-status", "not-a-status"),
				ExpectError: regexp.MustCompile(`Attribute status value must be one of`),
			},
		},
	})
}

func TestIntegrationArtifactLockedSpecCreatesNewVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

	initialID := uuid.NewString()
	updatedID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "locked-spec-" + uuid.NewString()[:8]

	initialArtifact := artifactFixtureWithStatusAndImage(initialID, &repoIDPtr, name, client.ArtifactStatusLocked, "nginx:latest")
	updatedArtifact := artifactFixtureWithStatusAndImage(updatedID, &repoIDPtr, name, client.ArtifactStatusLocked, "nginx:1.25")

	getArtifactResponse := initialArtifact
	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
			if req.Status != client.ArtifactStatusLocked {
				t.Errorf("expected create status locked, got %q", req.Status)
			}
			return initialArtifact, nil
		})
	mockService.EXPECT().
		GetArtifact(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string) (*client.Artifact, error) {
			return getArtifactResponse, nil
		}).AnyTimes()

	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
			if req.Status != client.ArtifactStatusLocked {
				t.Errorf("expected locked version create, got status %q", req.Status)
			}
			getArtifactResponse = updatedArtifact
			return updatedArtifact, nil
		})

	mockService.EXPECT().
		DeleteArtifactRepository(gomock.Any(), repoID).
		Return(nil)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: artifactResourceConfigWithStatusAndImage(name, "locked", "nginx:latest"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("datarobot_artifact.test", "status", "locked"),
					resource.TestCheckResourceAttr("datarobot_artifact.test", "artifact_id", initialID),
				),
			},
			{
				Config: artifactResourceConfigWithStatusAndImage(name, "locked", "nginx:1.25"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("datarobot_artifact.test", "status", "locked"),
					resource.TestCheckResourceAttr("datarobot_artifact.test", "artifact_id", updatedID),
					resource.TestCheckResourceAttrSet("datarobot_artifact.test", "id"),
				),
			},
		},
	})
}

func TestIntegrationArtifactDraftSpecPatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "draft-spec-" + uuid.NewString()[:8]

	draftArtifact := artifactFixtureWithStatusAndImage(artifactID, &repoIDPtr, name, client.ArtifactStatusDraft, "nginx:latest")
	updatedDraftArtifact := artifactFixtureWithStatusAndImage(artifactID, &repoIDPtr, name, client.ArtifactStatusDraft, "nginx:1.25")

	getArtifactResponse := draftArtifact
	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		Return(draftArtifact, nil)
	mockService.EXPECT().
		GetArtifact(gomock.Any(), artifactID).
		DoAndReturn(func(context.Context, string) (*client.Artifact, error) {
			return getArtifactResponse, nil
		}).AnyTimes()
	mockService.EXPECT().
		PatchArtifact(gomock.Any(), artifactID, gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, req *client.PatchArtifactRequest) (*client.Artifact, error) {
			if req.Status != nil {
				t.Errorf("expected spec-only patch, got status %v", req.Status)
			}
			if req.Spec == nil {
				t.Fatal("expected spec in patch request")
			}
			getArtifactResponse = updatedDraftArtifact
			return updatedDraftArtifact, nil
		})
	mockService.EXPECT().
		DeleteArtifactRepository(gomock.Any(), repoID).
		Return(nil)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: artifactResourceConfigWithStatusAndImage(name, "draft", "nginx:latest"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("datarobot_artifact.test", "status", "draft"),
					resource.TestCheckResourceAttr("datarobot_artifact.test", "artifact_id", artifactID),
				),
			},
			{
				Config: artifactResourceConfigWithStatusAndImage(name, "draft", "nginx:1.25"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("datarobot_artifact.test", "status", "draft"),
					resource.TestCheckResourceAttr("datarobot_artifact.test", "artifact_id", artifactID),
				),
			},
		},
	})
}

func TestArtifactImageSourceRequired(t *testing.T) {
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

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

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

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

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

func TestValidateArtifactSource(t *testing.T) {
	t.Parallel()

	validDir := t.TempDir()
	missingDir := filepath.Join(validDir, "does-not-exist")
	filePath := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	buildConfig := &ArtifactImageBuildConfigModel{
		Dockerfile: &ArtifactDockerfileModel{Source: types.StringValue("provided")},
	}
	specWithBuildConfig := &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{{
			Containers: []ArtifactContainerModel{{
				Primary:          types.BoolValue(true),
				Port:             types.Int64Value(8080),
				ImageBuildConfig: buildConfig,
			}},
		}},
	}
	specWithImageURIOnly := &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{{
			Containers: []ArtifactContainerModel{{
				Primary:  types.BoolValue(true),
				Port:     types.Int64Value(8080),
				ImageURI: types.StringValue("nginx:latest"),
			}},
		}},
	}
	specWithManualCodeRef := &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{{
			Containers: []ArtifactContainerModel{{
				Primary: types.BoolValue(true),
				Port:    types.Int64Value(8080),
				ImageBuildConfig: &ArtifactImageBuildConfigModel{
					CodeRef: artifactCodeRefObject(&ArtifactCodeRefModel{
						CatalogID:        types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaa"),
						CatalogVersionID: types.StringValue("bbbbbbbbbbbbbbbbbbbbbbbb"),
					}),
					Dockerfile: &ArtifactDockerfileModel{Source: types.StringValue("provided")},
				},
			}},
		}},
	}

	tests := []struct {
		name        string
		data        ArtifactResourceModel
		wantSummary string
	}{
		{
			name: "valid draft source",
			data: ArtifactResourceModel{
				Status: types.StringValue("draft"),
				Source: &ArtifactSourceModel{Dir: types.StringValue(validDir)},
				Spec:   specWithBuildConfig,
			},
		},
		{
			name: "missing dir",
			data: ArtifactResourceModel{
				Status: types.StringValue("draft"),
				Source: &ArtifactSourceModel{Dir: types.StringNull()},
				Spec:   specWithBuildConfig,
			},
			wantSummary: "Missing source directory",
		},
		{
			name: "dir not found",
			data: ArtifactResourceModel{
				Status: types.StringValue("draft"),
				Source: &ArtifactSourceModel{Dir: types.StringValue(missingDir)},
				Spec:   specWithBuildConfig,
			},
			wantSummary: "Source directory not found",
		},
		{
			name: "dir is file",
			data: ArtifactResourceModel{
				Status: types.StringValue("draft"),
				Source: &ArtifactSourceModel{Dir: types.StringValue(filePath)},
				Spec:   specWithBuildConfig,
			},
			wantSummary: "Invalid source directory",
		},
		{
			name: "nim artifact",
			data: ArtifactResourceModel{
				Type:   types.StringValue("nim"),
				Status: types.StringValue("draft"),
				Source: &ArtifactSourceModel{Dir: types.StringValue(validDir)},
				Spec:   specWithBuildConfig,
			},
			wantSummary: "Unsupported source on NIM artifacts",
		},
		{
			name: "agent artifact with source",
			data: ArtifactResourceModel{
				Type:   types.StringValue("agent"),
				Status: types.StringValue("draft"),
				Source: &ArtifactSourceModel{Dir: types.StringValue(validDir)},
				Spec:   specWithBuildConfig,
			},
		},
		{
			name: "missing spec",
			data: ArtifactResourceModel{
				Status: types.StringValue("draft"),
				Source: &ArtifactSourceModel{Dir: types.StringValue(validDir)},
			},
			wantSummary: "Missing image build target",
		},
		{
			name: "primary without image_build_config",
			data: ArtifactResourceModel{
				Status: types.StringValue("draft"),
				Source: &ArtifactSourceModel{Dir: types.StringValue(validDir)},
				Spec:   specWithImageURIOnly,
			},
			wantSummary: "Missing image build target",
		},
		{
			name: "manual code_ref conflict",
			data: ArtifactResourceModel{
				Status: types.StringValue("draft"),
				Source: &ArtifactSourceModel{Dir: types.StringValue(validDir)},
				Spec:   specWithManualCodeRef,
			},
			wantSummary: "Conflicting code_ref",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &tfresource.ValidateConfigResponse{}
			validateArtifactSource(resp, tt.data)

			if tt.wantSummary == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected no errors, got: %v", resp.Diagnostics.Errors())
				}
				return
			}

			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected validation error %q", tt.wantSummary)
			}
			if !strings.Contains(resp.Diagnostics.Errors()[0].Summary(), tt.wantSummary) {
				t.Fatalf("expected summary %q, got %q", tt.wantSummary, resp.Diagnostics.Errors()[0].Summary())
			}
		})
	}
}

func TestArtifactHasPrimaryImageBuildConfig(t *testing.T) {
	t.Parallel()

	buildConfig := &ArtifactImageBuildConfigModel{
		Dockerfile: &ArtifactDockerfileModel{Source: types.StringValue("provided")},
	}

	tests := []struct {
		name string
		spec *ArtifactSpecModel
		want bool
	}{
		{
			name: "explicit primary with build config",
			spec: &ArtifactSpecModel{
				ContainerGroups: []ArtifactContainerGroupModel{{
					Containers: []ArtifactContainerModel{{
						Primary:          types.BoolValue(true),
						ImageBuildConfig: buildConfig,
					}},
				}},
			},
			want: true,
		},
		{
			name: "sole container without primary flag",
			spec: &ArtifactSpecModel{
				ContainerGroups: []ArtifactContainerGroupModel{{
					Containers: []ArtifactContainerModel{{
						ImageBuildConfig: buildConfig,
					}},
				}},
			},
			want: true,
		},
		{
			name: "primary with image_uri only",
			spec: &ArtifactSpecModel{
				ContainerGroups: []ArtifactContainerGroupModel{{
					Containers: []ArtifactContainerModel{{
						Primary:  types.BoolValue(true),
						ImageURI: types.StringValue("nginx:latest"),
					}},
				}},
			},
			want: false,
		},
		{
			name: "build config on non-primary sidecar",
			spec: &ArtifactSpecModel{
				ContainerGroups: []ArtifactContainerGroupModel{{
					Containers: []ArtifactContainerModel{
						{Primary: types.BoolValue(true), Port: types.Int64Value(8080)},
						{Primary: types.BoolValue(false), ImageBuildConfig: buildConfig},
					},
				}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := artifactHasPrimaryImageBuildConfig(tt.spec); got != tt.want {
				t.Fatalf("artifactHasPrimaryImageBuildConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArtifactHasManualCodeRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec *ArtifactSpecModel
		want bool
	}{
		{
			name: "both catalog ids set",
			spec: &ArtifactSpecModel{
				ContainerGroups: []ArtifactContainerGroupModel{{
					Containers: []ArtifactContainerModel{{
						ImageBuildConfig: &ArtifactImageBuildConfigModel{
							CodeRef: artifactCodeRefObject(&ArtifactCodeRefModel{
								CatalogID:        types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaa"),
								CatalogVersionID: types.StringValue("bbbbbbbbbbbbbbbbbbbbbbbb"),
							}),
						},
					}},
				}},
			},
			want: true,
		},
		{
			name: "empty code_ref block",
			spec: &ArtifactSpecModel{
				ContainerGroups: []ArtifactContainerGroupModel{{
					Containers: []ArtifactContainerModel{{
						ImageBuildConfig: &ArtifactImageBuildConfigModel{
							CodeRef: artifactCodeRefObject(&ArtifactCodeRefModel{}),
						},
					}},
				}},
			},
			want: false,
		},
		{
			name: "no image_build_config",
			spec: &ArtifactSpecModel{
				ContainerGroups: []ArtifactContainerGroupModel{{
					Containers: []ArtifactContainerModel{{
						ImageURI: types.StringValue("nginx:latest"),
					}},
				}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := artifactHasManualCodeRef(tt.spec); got != tt.want {
				t.Fatalf("artifactHasManualCodeRef() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArtifactModifyPlanComputesSourceDirHash(t *testing.T) {
	validDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(validDir, "main.py"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             artifactConfigWithSource("plan-hash", "draft", validDir),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("datarobot_artifact.test", "source.dir_hash"),
				),
			},
		},
	})
}

// TestArtifactModifyPlanImageURIClearsWithoutSource guards the manual image_build_config
// path (no `source` block) that already shipped in a prior release: image_uri must still
// behave like a plain Optional attribute there. Making image_uri Computed for the new
// source-driven build feature must not silently keep a stale image_uri in state forever
// when a user drops it from config on a manually-managed image_build_config container.
func TestArtifactModifyPlanImageURIClearsWithoutSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "manual-image-uri-" + uuid.NewString()[:8]

	draftArtifact := &client.Artifact{
		ID:                   artifactID,
		Name:                 name,
		Type:                 client.ArtifactTypeService,
		Status:               client.ArtifactStatusDraft,
		ArtifactRepositoryID: &repoIDPtr,
		Spec: client.ArtifactSpec{
			ContainerGroups: []client.ArtifactContainerGroup{{
				Containers: []client.ArtifactContainer{{
					Name:     stringPtr("main"),
					Primary:  boolPtr(true),
					Port:     func() *int64 { v := int64(8080); return &v }(),
					ImageURI: "registry.example.com/app:manual",
					ImageBuildConfig: &client.ArtifactImageBuildConfig{
						Dockerfile: &client.ArtifactDockerfileConfig{Source: "provided", Path: "./Dockerfile"},
					},
				}},
			}},
		},
	}

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	mockService.EXPECT().GetArtifact(gomock.Any(), artifactID).Return(draftArtifact, nil).AnyTimes()
	mockService.EXPECT().DeleteArtifactRepository(gomock.Any(), repoID).Return(nil)

	configWithURI := fmt.Sprintf(`
resource "datarobot_artifact" "test" {
  name   = %q
  status = "draft"
  spec = {
    container_groups = [{
      containers = [{
        name      = "main"
        primary   = true
        port      = 8080
        image_uri = "registry.example.com/app:manual"
        image_build_config = {
          dockerfile = { source = "provided" }
        }
      }]
    }]
  }
}`, name)

	configWithoutURI := fmt.Sprintf(`
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
}`, name)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configWithURI,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("datarobot_artifact.test", "spec.container_groups.0.containers.0.image_uri", "registry.example.com/app:manual"),
				),
			},
			{
				// Dropping image_uri from config (no source configured) must still plan a
				// change, matching pre-Computed behavior — not silently keep the stale value.
				Config:             configWithoutURI,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestArtifactModifyPlanImageURIStaysKnownWithUnchangedSource checks the other side of the
// image_uri Computed scoping: with `source` configured and image_uri left unset in config,
// a re-plan against unchanged source content must NOT diff image_uri to null. If it did,
// apply would send a clearing PatchArtifact even though nothing changed — silently wiping
// out the last build's image_uri on every no-op plan/apply.
func TestArtifactModifyPlanImageURIStaysKnownWithUnchangedSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

	sourceDir := writeArtifactSourceTree(t, map[string]string{
		"main.py":    "print('v1')",
		"Dockerfile": "FROM python:3.11-slim\nWORKDIR /app\nCOPY main.py .\nCMD [\"python\", \"main.py\"]\n",
	})

	resourceName := "datarobot_artifact.test"
	name := "source-noop-image-uri-" + uuid.NewString()[:8]

	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
	filesAPI := newSyncTestFilesAPI()

	var currentArtifact *client.Artifact
	patchCodeRef := func(_ context.Context, id, catalogID, versionID string) (*client.Artifact, error) {
		currentArtifact = artifactSourcePatchedArtifact(draftArtifact, catalogID, versionID)
		return currentArtifact, nil
	}

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil).Times(1)
	mockService.EXPECT().FilesAPI().Return(filesAPI).Times(1)
	mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).DoAndReturn(patchCodeRef).Times(1)
	expectArtifactBuildAfterUploadFromLatest(mockService, artifactID, &currentArtifact)
	mockService.EXPECT().GetArtifact(gomock.Any(), artifactID).DoAndReturn(
		func(_ context.Context, id string) (*client.Artifact, error) {
			if currentArtifact != nil {
				return artifactSourceBuiltForRead(currentArtifact), nil
			}
			return draftArtifact, nil
		},
	).AnyTimes()
	mockService.EXPECT().DeleteArtifactRepository(gomock.Any(), repoID).Return(nil)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: artifactConfigWithSource(name, "draft", sourceDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "spec.container_groups.0.containers.0.image_uri"),
				),
			},
			{
				// Same source dir, unchanged content/dir_hash: nothing should diff.
				Config:             artifactConfigWithSource(name, "draft", sourceDir),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestArtifactSourceConfigValidation(t *testing.T) {

	validDir := t.TempDir()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

	tests := []struct {
		name        string
		config      string
		expectError *regexp.Regexp
	}{
		{
			name:        "nim with source",
			config:      artifactConfigWithSourceType("nim-source", "draft", "nim", validDir),
			expectError: regexp.MustCompile("Unsupported source on NIM artifacts"),
		},
		{
			name:        "source with manual code_ref",
			config:      artifactConfigWithSourceAndCodeRef(validDir),
			expectError: regexp.MustCompile("Conflicting code_ref"),
		},
		{
			name:        "missing source dir",
			config:      artifactConfigWithSource("missing-dir", "draft", filepath.Join(validDir, "missing")),
			expectError: regexp.MustCompile("Source directory not found"),
		},
		{
			name:        "primary without image_build_config",
			config:      artifactConfigWithSourceImageURIONly(validDir),
			expectError: regexp.MustCompile("Missing image build target"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      tt.config,
						ExpectError: tt.expectError,
					},
				},
			})
		})
	}
}

const (
	artifactSourceTestCatalogID = "aaaaaaaaaaaaaaaaaaaaaaaa"
	artifactSourceTestVersionID = "bbbbbbbbbbbbbbbbbbbbbbbb"
	artifactSourceTestBuildID   = "eeeeeeeeeeeeeeeeeeeeeeee"
	artifactSourceTestImageURI  = "registry.example.com/app:test"
)

func artifactFixtureDraftWithBuildConfig(id string, repoID *string, name string) *client.Artifact {
	port := int64(8080)
	primary := true
	containerName := "main"
	return &client.Artifact{
		ID:                   id,
		Name:                 name,
		Type:                 client.ArtifactTypeService,
		Status:               client.ArtifactStatusDraft,
		ArtifactRepositoryID: repoID,
		Spec: client.ArtifactSpec{
			ContainerGroups: []client.ArtifactContainerGroup{{
				Containers: []client.ArtifactContainer{{
					Name:    &containerName,
					Primary: &primary,
					Port:    &port,
					ImageBuildConfig: &client.ArtifactImageBuildConfig{
						Dockerfile: &client.ArtifactDockerfileConfig{Source: "provided", Path: "./Dockerfile"},
					},
				}},
			}},
		},
	}
}

func artifactSourcePatchedArtifact(base *client.Artifact, catalogID, versionID string) *client.Artifact {
	patched := *base
	primary := true
	patched.Spec = base.Spec
	groups := make([]client.ArtifactContainerGroup, len(base.Spec.ContainerGroups))
	for gi, group := range base.Spec.ContainerGroups {
		containers := make([]client.ArtifactContainer, len(group.Containers))
		for ci, container := range group.Containers {
			containers[ci] = container
			isPrimary := container.Primary != nil && *container.Primary
			if isPrimary || (container.Primary == nil && ci == 0) {
				if containers[ci].ImageBuildConfig == nil {
					containers[ci].ImageBuildConfig = &client.ArtifactImageBuildConfig{}
				}
				containers[ci].ImageBuildConfig.CodeRef = &client.ArtifactCodeRef{
					Type: "datarobot",
					DataRobot: client.ArtifactDataRobotCodeRef{
						CatalogID:        catalogID,
						CatalogVersionID: versionID,
					},
				}
				containers[ci].Primary = &primary
			}
		}
		groups[gi] = client.ArtifactContainerGroup{Containers: containers}
	}
	patched.Spec.ContainerGroups = groups
	return &patched
}

func artifactFixtureWithImageURI(base *client.Artifact) *client.Artifact {
	built := *base
	primary := true
	built.Spec = base.Spec
	groups := make([]client.ArtifactContainerGroup, len(base.Spec.ContainerGroups))
	for gi, group := range base.Spec.ContainerGroups {
		containers := make([]client.ArtifactContainer, len(group.Containers))
		for ci, container := range group.Containers {
			containers[ci] = container
			isPrimary := container.Primary != nil && *container.Primary
			if isPrimary || (container.Primary == nil && ci == 0) {
				containers[ci].ImageURI = artifactSourceTestImageURI
				containers[ci].Primary = &primary
				containers[ci].Build = &client.ArtifactContainerBuildInfo{
					ArtifactImageBuildID: artifactSourceTestBuildID,
					Status:               client.ArtifactBuildStatusCompleted,
					CreatedAt:            "2026-01-01T00:00:00Z",
				}
			}
		}
		groups[gi] = client.ArtifactContainerGroup{Containers: containers}
	}
	built.Spec.ContainerGroups = groups
	return &built
}

func expectArtifactBuildAfterUpload(mockService *mock_client.MockService, artifactID string, builtArtifact *client.Artifact) {
	expectArtifactBuildAfterUploadWithLatest(mockService, artifactID, builtArtifact, nil)
}

func expectArtifactBuildAfterUploadFromLatest(mockService *mock_client.MockService, artifactID string, latest **client.Artifact) {
	gomock.InOrder(
		mockService.EXPECT().
			TriggerArtifactBuild(gomock.Any(), artifactID).
			Return(&client.ArtifactBuildTriggerResponse{BuildIDs: []string{artifactSourceTestBuildID}}, nil),
		mockService.EXPECT().
			WaitForArtifactBuild(gomock.Any(), artifactID, artifactSourceTestBuildID, gomock.Any()).
			Return(&client.ArtifactBuild{ID: artifactSourceTestBuildID, Status: client.ArtifactBuildStatusCompleted}, nil),
		mockService.EXPECT().
			GetArtifact(gomock.Any(), artifactID).
			DoAndReturn(func(_ context.Context, id string) (*client.Artifact, error) {
				if latest == nil || *latest == nil {
					panic("expected artifact before build refresh")
				}
				built := artifactFixtureWithImageURI(*latest)
				*latest = built
				return built, nil
			}),
	)
}

func expectArtifactBuildAfterUploadWithLatest(mockService *mock_client.MockService, artifactID string, builtArtifact *client.Artifact, latest **client.Artifact) {
	gomock.InOrder(
		mockService.EXPECT().
			TriggerArtifactBuild(gomock.Any(), artifactID).
			Return(&client.ArtifactBuildTriggerResponse{BuildIDs: []string{artifactSourceTestBuildID}}, nil),
		mockService.EXPECT().
			WaitForArtifactBuild(gomock.Any(), artifactID, artifactSourceTestBuildID, gomock.Any()).
			Return(&client.ArtifactBuild{ID: artifactSourceTestBuildID, Status: client.ArtifactBuildStatusCompleted}, nil),
		mockService.EXPECT().
			GetArtifact(gomock.Any(), artifactID).
			DoAndReturn(func(_ context.Context, id string) (*client.Artifact, error) {
				if latest != nil {
					*latest = builtArtifact
				}
				return builtArtifact, nil
			}),
	)
}

func artifactSourceBuiltForRead(artifact *client.Artifact) *client.Artifact {
	if artifact == nil || client.ExtractCodeRef(artifact) == nil {
		return artifact
	}
	return artifactFixtureWithImageURI(artifact)
}

func expectArtifactBuildTriggerOnly(mockService *mock_client.MockService, artifactID string, builtArtifact *client.Artifact) {
	gomock.InOrder(
		mockService.EXPECT().
			TriggerArtifactBuild(gomock.Any(), artifactID).
			Return(&client.ArtifactBuildTriggerResponse{BuildIDs: []string{artifactSourceTestBuildID}}, nil),
		mockService.EXPECT().
			GetArtifact(gomock.Any(), artifactID).
			Return(builtArtifact, nil),
	)
}

func artifactResourceModelWithSource(name, dir string) ArtifactResourceModel {
	return ArtifactResourceModel{
		Name:   types.StringValue(name),
		Status: types.StringValue("draft"),
		Type:   types.StringValue("service"),
		Source: &ArtifactSourceModel{Dir: types.StringValue(dir)},
		Spec: &ArtifactSpecModel{
			ContainerGroups: []ArtifactContainerGroupModel{{
				Containers: []ArtifactContainerModel{{
					Name:    types.StringValue("main"),
					Primary: types.BoolValue(true),
					Port:    types.Int64Value(8080),
					Build:   artifactBuildNull(),
					ImageBuildConfig: &ArtifactImageBuildConfigModel{
						CodeRef: artifactCodeRefNull(),
						Dockerfile: &ArtifactDockerfileModel{
							Source: types.StringValue("provided"),
						},
					},
				}},
			}},
		},
	}
}

func testArtifactResourceSchemaFor(r *ArtifactResource) (schema.Schema, diag.Diagnostics) {
	resp := &tfresource.SchemaResponse{}
	r.Schema(context.Background(), tfresource.SchemaRequest{}, resp)
	return resp.Schema, resp.Diagnostics
}

// testArtifactApplyCreate exercises the real Create() implementation (not a parallel copy).

func testArtifactApplyCreate(ctx context.Context, r *ArtifactResource, data ArtifactResourceModel) (ArtifactResourceModel, diag.Diagnostics) {
	schema, diags := testArtifactResourceSchemaFor(r)
	if diags.HasError() {
		return data, diags
	}

	plan := tfsdk.Plan{Schema: schema}
	diags.Append(plan.Set(ctx, &data)...)
	if diags.HasError() {
		return data, diags
	}

	resp := &tfresource.CreateResponse{
		State: tfsdk.State{
			Schema: schema,
			Raw:    tftypes.NewValue(schema.Type().TerraformType(ctx), nil),
		},
	}
	r.Create(ctx, tfresource.CreateRequest{Plan: plan}, resp)

	var result ArtifactResourceModel
	if !resp.State.Raw.IsNull() {
		resp.Diagnostics.Append(resp.State.Get(ctx, &result)...)
	}
	return result, resp.Diagnostics
}

// testArtifactApplyRead exercises the real Read() implementation.

func testArtifactApplyRead(ctx context.Context, r *ArtifactResource, data ArtifactResourceModel) (ArtifactResourceModel, diag.Diagnostics, bool) {
	schema, diags := testArtifactResourceSchemaFor(r)
	if diags.HasError() {
		return data, diags, false
	}

	state := tfsdk.State{Schema: schema}
	diags.Append(state.Set(ctx, &data)...)
	if diags.HasError() {
		return data, diags, false
	}

	resp := &tfresource.ReadResponse{State: state}
	r.Read(ctx, tfresource.ReadRequest{State: state}, resp)

	if resp.State.Raw.IsNull() {
		return data, resp.Diagnostics, true
	}

	var result ArtifactResourceModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &result)...)
	return result, resp.Diagnostics, false
}

// testArtifactApplyUpdate exercises the real Update() implementation (not a parallel copy).

func testArtifactApplyUpdate(ctx context.Context, r *ArtifactResource, planModel, stateModel ArtifactResourceModel) (ArtifactResourceModel, diag.Diagnostics) {
	schema, diags := testArtifactResourceSchemaFor(r)
	if diags.HasError() {
		return planModel, diags
	}

	plan := tfsdk.Plan{Schema: schema}
	diags.Append(plan.Set(ctx, &planModel)...)
	if diags.HasError() {
		return planModel, diags
	}

	state := tfsdk.State{Schema: schema}
	diags.Append(state.Set(ctx, &stateModel)...)
	if diags.HasError() {
		return planModel, diags
	}

	resp := &tfresource.UpdateResponse{State: state}
	r.Update(ctx, tfresource.UpdateRequest{Plan: plan, State: state}, resp)

	var result ArtifactResourceModel
	if !resp.State.Raw.IsNull() {
		resp.Diagnostics.Append(resp.State.Get(ctx, &result)...)
	}
	return result, resp.Diagnostics
}

// testArtifactApplyModifyPlan exercises the real ModifyPlan() implementation, unlike the
// unit tests for artifactModifyPlanNeedsUnknownImageURI/applySourceManagedImageURIToPlan
// which call those helpers directly and never wire through the resource's config handling.
func testArtifactApplyModifyPlan(ctx context.Context, r *ArtifactResource, configModel, planModel, stateModel ArtifactResourceModel) (ArtifactResourceModel, diag.Diagnostics) {
	schema, diags := testArtifactResourceSchemaFor(r)
	if diags.HasError() {
		return planModel, diags
	}

	configPlan := tfsdk.Plan{Schema: schema}
	diags.Append(configPlan.Set(ctx, &configModel)...)
	if diags.HasError() {
		return planModel, diags
	}
	config := tfsdk.Config{Schema: schema, Raw: configPlan.Raw}

	plan := tfsdk.Plan{Schema: schema}
	diags.Append(plan.Set(ctx, &planModel)...)
	if diags.HasError() {
		return planModel, diags
	}

	state := tfsdk.State{Schema: schema}
	diags.Append(state.Set(ctx, &stateModel)...)
	if diags.HasError() {
		return planModel, diags
	}

	resp := &tfresource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, tfresource.ModifyPlanRequest{Config: config, Plan: plan, State: state}, resp)

	var result ArtifactResourceModel
	if !resp.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(resp.Plan.Get(ctx, &result)...)
	}
	return result, resp.Diagnostics
}

func diagErrorSummary(diags diag.Diagnostics) string {
	if !diags.HasError() {
		return ""
	}
	return diags.Errors()[0].Summary()
}

func TestArtifactResourceSourceCreateSuccess(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDir := writeArtifactSourceTree(t, map[string]string{"main.py": "app"})
	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-create-" + uuid.NewString()[:8]

	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
	patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	filesAPI := newSyncTestFilesAPI()

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	mockService.EXPECT().FilesAPI().Return(filesAPI)
	mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil)
	expectArtifactBuildAfterUpload(mockService, artifactID, artifactFixtureWithImageURI(patchedArtifact))

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	result, diags := testArtifactApplyCreate(context.Background(), resource, artifactResourceModelWithSource(name, sourceDir))
	if diags.HasError() {
		t.Fatalf("create: %s", diagErrorSummary(diags))
	}
	if result.ArtifactID.ValueString() != artifactID {
		t.Fatalf("artifact_id = %q, want %q", result.ArtifactID.ValueString(), artifactID)
	}
	if !IsKnown(result.Source.DirHash) {
		t.Fatal("expected source.dir_hash to be set after create")
	}
	codeRef := imageBuildConfigCodeRef(result.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig)
	if codeRef == nil {
		t.Fatal("expected code_ref after create")
	}
	if got := codeRef.CatalogID.ValueString(); got != artifactSourceTestCatalogID {
		t.Fatalf("catalog_id = %q, want %q", got, artifactSourceTestCatalogID)
	}
}

func TestArtifactResourceSourceCreateLockedSuccess(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDir := writeArtifactSourceTree(t, map[string]string{"main.py": "app"})
	draftArtifactID := uuid.NewString()
	lockedArtifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-create-locked-" + uuid.NewString()[:8]

	draftArtifact := artifactFixtureDraftWithBuildConfig(draftArtifactID, &repoIDPtr, name)
	patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	lockedArtifact := *patchedArtifact
	lockedArtifact.ID = lockedArtifactID
	lockedArtifact.Status = client.ArtifactStatusLocked
	filesAPI := newSyncTestFilesAPI()

	gomock.InOrder(
		mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
				if req.Status != client.ArtifactStatusDraft {
					t.Fatalf("expected draft create before lock, got %q", req.Status)
				}
				return draftArtifact, nil
			}),
		mockService.EXPECT().FilesAPI().Return(filesAPI),
		mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), draftArtifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil),
	)
	expectArtifactBuildAfterUpload(mockService, draftArtifactID, artifactFixtureWithImageURI(patchedArtifact))
	mockService.EXPECT().PatchArtifact(gomock.Any(), draftArtifactID, gomock.Any()).DoAndReturn(
		func(_ context.Context, id string, req *client.PatchArtifactRequest) (*client.Artifact, error) {
			if req.Status == nil || *req.Status != client.ArtifactStatusLocked {
				t.Fatalf("expected lock patch, got status %v", req.Status)
			}
			return &lockedArtifact, nil
		},
	)

	model := artifactResourceModelWithSource(name, sourceDir)
	model.Status = types.StringValue("locked")

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	result, diags := testArtifactApplyCreate(context.Background(), resource, model)
	if diags.HasError() {
		t.Fatalf("create: %s", diagErrorSummary(diags))
	}
	if result.ArtifactID.ValueString() != lockedArtifactID {
		t.Fatalf("artifact_id = %q, want %q", result.ArtifactID.ValueString(), lockedArtifactID)
	}
	if result.Status.ValueString() != "locked" {
		t.Fatalf("status = %q, want locked", result.Status.ValueString())
	}
}

func TestArtifactResourceSourceUpdateLockedSourceChangeCloneLock(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDirV1 := writeArtifactSourceTree(t, map[string]string{"main.py": "v1"})
	sourceDirV2 := writeArtifactSourceTree(t, map[string]string{"main.py": "v2"})
	lockedArtifactID := uuid.NewString()
	draftCloneID := uuid.NewString()
	newLockedArtifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-locked-update-" + uuid.NewString()[:8]

	lockedArtifact := artifactFixtureDraftWithBuildConfig(lockedArtifactID, &repoIDPtr, name)
	lockedArtifact.Status = client.ArtifactStatusLocked
	lockedArtifact.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig.CodeRef = &client.ArtifactCodeRef{
		Type: "datarobot",
		DataRobot: client.ArtifactDataRobotCodeRef{
			CatalogID:        artifactSourceTestCatalogID,
			CatalogVersionID: artifactSourceTestVersionID,
		},
	}

	draftClone := artifactFixtureDraftWithBuildConfig(draftCloneID, &repoIDPtr, name)
	patchedDraft := artifactSourcePatchedArtifact(draftClone, "cccccccccccccccccccccccc", "dddddddddddddddddddddddd")
	lockedResult := *patchedDraft
	lockedResult.ID = newLockedArtifactID
	lockedResult.Status = client.ArtifactStatusLocked

	filesAPI := newSyncTestFilesAPI()

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
			if req.Status != client.ArtifactStatusDraft {
				t.Fatalf("expected draft clone, got %q", req.Status)
			}
			if req.ArtifactRepositoryID == nil || *req.ArtifactRepositoryID != repoID {
				t.Fatalf("expected repository %q, got %v", repoID, req.ArtifactRepositoryID)
			}
			return draftClone, nil
		})
	mockService.EXPECT().FilesAPI().Return(filesAPI)
	mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), draftCloneID, gomock.Any(), gomock.Any()).Return(patchedDraft, nil)
	expectArtifactBuildAfterUpload(mockService, draftCloneID, artifactFixtureWithImageURI(patchedDraft))
	mockService.EXPECT().PatchArtifact(gomock.Any(), draftCloneID, gomock.Any()).DoAndReturn(
		func(_ context.Context, id string, req *client.PatchArtifactRequest) (*client.Artifact, error) {
			if req.Status == nil || *req.Status != client.ArtifactStatusLocked {
				t.Fatalf("expected lock patch, got status %v", req.Status)
			}
			return &lockedResult, nil
		})

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	state := artifactResourceModelWithSource(name, sourceDirV1)
	state.Status = types.StringValue("locked")
	state.ArtifactID = types.StringValue(lockedArtifactID)
	state.ArtifactRepositoryID = types.StringValue(repoID)
	_ = setImageBuildConfigCodeRef(state.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig, &ArtifactCodeRefModel{
		CatalogID:        types.StringValue(artifactSourceTestCatalogID),
		CatalogVersionID: types.StringValue(artifactSourceTestVersionID),
	})
	state.Source.DirHash = types.StringValue("hash-v1")

	plan := artifactResourceModelWithSource(name, sourceDirV2)
	plan.Status = types.StringValue("locked")
	plan.ArtifactID = state.ArtifactID
	plan.ArtifactRepositoryID = state.ArtifactRepositoryID
	plan.Source.DirHash = types.StringValue("hash-v2")

	updated, diags := testArtifactApplyUpdate(context.Background(), resource, plan, state)
	if diags.HasError() {
		t.Fatalf("update: %s", diagErrorSummary(diags))
	}
	if updated.ArtifactID.ValueString() != newLockedArtifactID {
		t.Fatalf("artifact_id = %q, want %q", updated.ArtifactID.ValueString(), newLockedArtifactID)
	}
	if updated.Status.ValueString() != "locked" {
		t.Fatalf("status = %q, want locked", updated.Status.ValueString())
	}
}

func TestArtifactResourceSourceUpdateLockedSpecChangeCloneLock(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDir := writeArtifactSourceTree(t, map[string]string{"main.py": "stable"})
	lockedArtifactID := uuid.NewString()
	draftCloneID := uuid.NewString()
	newLockedArtifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-locked-spec-" + uuid.NewString()[:8]

	draftClone := artifactFixtureDraftWithBuildConfig(draftCloneID, &repoIDPtr, name)
	patchedDraft := artifactSourcePatchedArtifact(draftClone, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	lockedResult := *patchedDraft
	lockedResult.ID = newLockedArtifactID
	lockedResult.Status = client.ArtifactStatusLocked
	port9090 := int64(9090)
	lockedResult.Spec.ContainerGroups[0].Containers[0].Port = &port9090

	filesAPI := newSyncTestFilesAPI()

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
			if req.Status != client.ArtifactStatusDraft {
				t.Fatalf("expected draft clone, got %q", req.Status)
			}
			if req.ArtifactRepositoryID == nil || *req.ArtifactRepositoryID != repoID {
				t.Fatalf("expected repository %q, got %v", repoID, req.ArtifactRepositoryID)
			}
			return draftClone, nil
		})
	mockService.EXPECT().FilesAPI().Return(filesAPI)
	mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), draftCloneID, gomock.Any(), gomock.Any()).Return(patchedDraft, nil)
	expectArtifactBuildAfterUpload(mockService, draftCloneID, artifactFixtureWithImageURI(patchedDraft))
	mockService.EXPECT().PatchArtifact(gomock.Any(), draftCloneID, gomock.Any()).DoAndReturn(
		func(_ context.Context, id string, req *client.PatchArtifactRequest) (*client.Artifact, error) {
			if req.Status == nil || *req.Status != client.ArtifactStatusLocked {
				t.Fatalf("expected lock patch, got status %v", req.Status)
			}
			return &lockedResult, nil
		})

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	dirHash, err := computeFolderHash(types.StringValue(sourceDir))
	if err != nil {
		t.Fatal(err)
	}

	state := artifactResourceModelWithSource(name, sourceDir)
	state.Status = types.StringValue("locked")
	state.ArtifactID = types.StringValue(lockedArtifactID)
	state.ArtifactRepositoryID = types.StringValue(repoID)
	_ = setImageBuildConfigCodeRef(state.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig, &ArtifactCodeRefModel{
		CatalogID:        types.StringValue(artifactSourceTestCatalogID),
		CatalogVersionID: types.StringValue(artifactSourceTestVersionID),
	})
	state.Source.DirHash = dirHash

	plan := artifactResourceModelWithSource(name, sourceDir)
	plan.Status = types.StringValue("locked")
	plan.ArtifactID = state.ArtifactID
	plan.ArtifactRepositoryID = state.ArtifactRepositoryID
	plan.Spec.ContainerGroups[0].Containers[0].Port = types.Int64Value(9090)
	plan.Source.DirHash = dirHash

	updated, diags := testArtifactApplyUpdate(context.Background(), resource, plan, state)
	if diags.HasError() {
		t.Fatalf("update: %s", diagErrorSummary(diags))
	}
	if updated.ArtifactID.ValueString() != newLockedArtifactID {
		t.Fatalf("artifact_id = %q, want %q", updated.ArtifactID.ValueString(), newLockedArtifactID)
	}
	if updated.Status.ValueString() != "locked" {
		t.Fatalf("status = %q, want locked", updated.Status.ValueString())
	}
	if got := updated.Spec.ContainerGroups[0].Containers[0].Port.ValueInt64(); got != 9090 {
		t.Fatalf("port = %d, want 9090", got)
	}
}

func TestArtifactResourceSourceUpdateDraftLockWithSourceChange(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDirV1 := writeArtifactSourceTree(t, map[string]string{"main.py": "v1"})
	sourceDirV2 := writeArtifactSourceTree(t, map[string]string{"main.py": "v2"})
	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-draft-lock-" + uuid.NewString()[:8]

	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
	patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	lockedArtifact := *patchedArtifact
	lockedArtifact.Status = client.ArtifactStatusLocked
	filesAPI1 := newSyncTestFilesAPI()
	filesAPI2 := newSyncTestFilesAPI()

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	gomock.InOrder(
		mockService.EXPECT().FilesAPI().Return(filesAPI1),
		mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil),
	)
	expectArtifactBuildAfterUpload(mockService, artifactID, artifactFixtureWithImageURI(patchedArtifact))
	mockService.EXPECT().PatchArtifact(gomock.Any(), artifactID, gomock.Any()).DoAndReturn(
		func(_ context.Context, id string, req *client.PatchArtifactRequest) (*client.Artifact, error) {
			if req.Status != nil {
				t.Fatalf("expected spec-only patch before lock, got status %v", req.Status)
			}
			return patchedArtifact, nil
		})
	gomock.InOrder(
		mockService.EXPECT().FilesAPI().Return(filesAPI2),
		mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil),
	)
	expectArtifactBuildAfterUpload(mockService, artifactID, artifactFixtureWithImageURI(patchedArtifact))
	mockService.EXPECT().PatchArtifact(gomock.Any(), artifactID, gomock.Any()).DoAndReturn(
		func(_ context.Context, id string, req *client.PatchArtifactRequest) (*client.Artifact, error) {
			if req.Status == nil || *req.Status != client.ArtifactStatusLocked {
				t.Fatalf("expected lock patch after source upload, got status %v", req.Status)
			}
			return &lockedArtifact, nil
		})

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	state, diags := testArtifactApplyCreate(context.Background(), resource, artifactResourceModelWithSource(name, sourceDirV1))
	if diags.HasError() {
		t.Fatalf("create: %s", diagErrorSummary(diags))
	}

	plan := artifactResourceModelWithSource(name, sourceDirV2)
	plan.Status = types.StringValue("locked")
	plan.ID = state.ID
	plan.ArtifactID = state.ArtifactID
	plan.ArtifactRepositoryID = state.ArtifactRepositoryID
	plan.Source.DirHash = types.StringValue("hash-v2")

	updated, diags := testArtifactApplyUpdate(context.Background(), resource, plan, state)
	if diags.HasError() {
		t.Fatalf("update: %s", diagErrorSummary(diags))
	}
	if updated.Status.ValueString() != "locked" {
		t.Fatalf("status = %q, want locked", updated.Status.ValueString())
	}
}

func TestArtifactResourceSourceCreateLockedLockFailure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDir := writeArtifactSourceTree(t, map[string]string{"main.py": "app"})
	draftArtifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-lock-fail-" + uuid.NewString()[:8]
	draftArtifact := artifactFixtureDraftWithBuildConfig(draftArtifactID, &repoIDPtr, name)
	patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	filesAPI := newSyncTestFilesAPI()

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	mockService.EXPECT().FilesAPI().Return(filesAPI)
	mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), draftArtifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil)
	expectArtifactBuildAfterUpload(mockService, draftArtifactID, artifactFixtureWithImageURI(patchedArtifact))
	mockService.EXPECT().PatchArtifact(gomock.Any(), draftArtifactID, gomock.Any()).Return(nil, fmt.Errorf("lock failed"))
	mockService.EXPECT().DeleteArtifactRepository(gomock.Any(), repoID).Return(nil)

	model := artifactResourceModelWithSource(name, sourceDir)
	model.Status = types.StringValue("locked")

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	_, diags := testArtifactApplyCreate(context.Background(), resource, model)
	if !diags.HasError() {
		t.Fatal("expected lock error")
	}
	if got := diagErrorSummary(diags); got != "Error locking Artifact after source upload" {
		t.Fatalf("error summary = %q", got)
	}
}

func TestArtifactResourceSourceUpdateLockedLockFailurePersistsDraftClone(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDirV1 := writeArtifactSourceTree(t, map[string]string{"main.py": "v1"})
	sourceDirV2 := writeArtifactSourceTree(t, map[string]string{"main.py": "v2"})
	lockedArtifactID := uuid.NewString()
	draftCloneID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-lock-update-fail-" + uuid.NewString()[:8]

	draftClone := artifactFixtureDraftWithBuildConfig(draftCloneID, &repoIDPtr, name)
	patchedDraft := artifactSourcePatchedArtifact(draftClone, "cccccccccccccccccccccccc", "dddddddddddddddddddddddd")
	filesAPI := newSyncTestFilesAPI()

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftClone, nil)
	mockService.EXPECT().FilesAPI().Return(filesAPI)
	mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), draftCloneID, gomock.Any(), gomock.Any()).Return(patchedDraft, nil)
	expectArtifactBuildAfterUpload(mockService, draftCloneID, artifactFixtureWithImageURI(patchedDraft))
	mockService.EXPECT().PatchArtifact(gomock.Any(), draftCloneID, gomock.Any()).Return(nil, fmt.Errorf("lock failed"))

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	state := artifactResourceModelWithSource(name, sourceDirV1)
	state.Status = types.StringValue("locked")
	state.ArtifactID = types.StringValue(lockedArtifactID)
	state.ArtifactRepositoryID = types.StringValue(repoID)
	state.Source.DirHash = types.StringValue("hash-v1")
	_ = setImageBuildConfigCodeRef(state.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig, &ArtifactCodeRefModel{
		CatalogID:        types.StringValue(artifactSourceTestCatalogID),
		CatalogVersionID: types.StringValue(artifactSourceTestVersionID),
	})

	plan := artifactResourceModelWithSource(name, sourceDirV2)
	plan.Status = types.StringValue("locked")
	plan.ArtifactID = state.ArtifactID
	plan.ArtifactRepositoryID = state.ArtifactRepositoryID
	plan.Source.DirHash = types.StringValue("hash-v2")

	updated, diags := testArtifactApplyUpdate(context.Background(), resource, plan, state)
	if !diags.HasError() {
		t.Fatal("expected lock error")
	}
	if got := diagErrorSummary(diags); got != "Error locking Artifact after source upload" {
		t.Fatalf("error summary = %q", got)
	}
	if updated.ArtifactID.ValueString() != draftCloneID {
		t.Fatalf("expected draft clone in state, artifact_id = %q", updated.ArtifactID.ValueString())
	}
	if updated.Status.ValueString() != "draft" {
		t.Fatalf("status = %q, want draft", updated.Status.ValueString())
	}
	if got := updated.Source.DirHash.ValueString(); got != "hash-v1" {
		t.Fatalf("dir_hash = %q, want prior hash-v1 so retry re-uploads", got)
	}
}

func TestArtifactResourceSourceCreateArtifactFailure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("create failed"))

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	_, diags := testArtifactApplyCreate(context.Background(), resource, artifactResourceModelWithSource("fail", writeArtifactSourceTree(t, map[string]string{"main.py": "x"})))
	if !diags.HasError() {
		t.Fatal("expected create error")
	}
	if got := diagErrorSummary(diags); got != "Error creating Artifact" {
		t.Fatalf("error summary = %q", got)
	}
}

func TestArtifactResourceSourceCreateUploadFailure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-upload-fail-" + uuid.NewString()[:8]
	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
	filesAPI := newSyncTestFilesAPI()
	filesAPI.uploadErr = fmt.Errorf("upload failed")

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	mockService.EXPECT().FilesAPI().Return(filesAPI)
	mockService.EXPECT().DeleteArtifactRepository(gomock.Any(), repoID).Return(nil)

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	_, diags := testArtifactApplyCreate(context.Background(), resource, artifactResourceModelWithSource(name, writeArtifactSourceTree(t, map[string]string{"main.py": "x"})))
	if !diags.HasError() {
		t.Fatal("expected upload error")
	}
	if got := diagErrorSummary(diags); got != "Error uploading artifact source" {
		t.Fatalf("error summary = %q", got)
	}
}

func TestArtifactResourceSourceCreatePatchCodeRefFailure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-patch-fail-" + uuid.NewString()[:8]
	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
	filesAPI := newSyncTestFilesAPI()

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	mockService.EXPECT().FilesAPI().Return(filesAPI)
	mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("patch failed"))
	mockService.EXPECT().DeleteArtifactRepository(gomock.Any(), repoID).Return(nil)

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	_, diags := testArtifactApplyCreate(context.Background(), resource, artifactResourceModelWithSource(name, writeArtifactSourceTree(t, map[string]string{"main.py": "x"})))
	if !diags.HasError() {
		t.Fatal("expected patch error")
	}
	if got := diagErrorSummary(diags); got != "Error uploading artifact source" {
		t.Fatalf("error summary = %q", got)
	}
}

// Create rolls back (deletes the artifact repository) when source upload or build fails:
// Terraform never records state on a failed create, so leaving the repo/artifact would
// orphan cloud resources and duplicate them on retry. Updates do not rollback on build failure.
func TestArtifactResourceSourceCreateBuildFailureRollback(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDir := writeArtifactSourceTree(t, map[string]string{"main.py": "app"})
	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-build-fail-" + uuid.NewString()[:8]

	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
	patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	filesAPI := newSyncTestFilesAPI()
	buildErr := &client.ArtifactBuildFailedError{BuildID: artifactSourceTestBuildID, Status: client.ArtifactBuildStatusFailed}

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	mockService.EXPECT().FilesAPI().Return(filesAPI)
	mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil)
	gomock.InOrder(
		mockService.EXPECT().
			TriggerArtifactBuild(gomock.Any(), artifactID).
			Return(&client.ArtifactBuildTriggerResponse{BuildIDs: []string{artifactSourceTestBuildID}}, nil),
		mockService.EXPECT().
			WaitForArtifactBuild(gomock.Any(), artifactID, artifactSourceTestBuildID, gomock.Any()).
			Return(&client.ArtifactBuild{ID: artifactSourceTestBuildID, Status: client.ArtifactBuildStatusFailed}, buildErr),
		mockService.EXPECT().
			BaseURL().
			Return("https://app.datarobot.com"),
		mockService.EXPECT().
			GetArtifactBuildLogs(gomock.Any(), artifactID, artifactSourceTestBuildID).
			Return("[2026-06-09 10:00:00] ERROR: docker build failed", nil),
	)
	mockService.EXPECT().DeleteArtifactRepository(gomock.Any(), repoID).Return(nil)

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	_, diags := testArtifactApplyCreate(context.Background(), resource, artifactResourceModelWithSource(name, sourceDir))
	if !diags.HasError() {
		t.Fatal("expected build error")
	}
	if got := diagErrorSummary(diags); got != "Error building artifact image" {
		t.Fatalf("error summary = %q", got)
	}
}

func TestArtifactResourceSourceCreateWaitForBuildFalse(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDir := writeArtifactSourceTree(t, map[string]string{"main.py": "app"})
	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-no-wait-" + uuid.NewString()[:8]

	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
	patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	builtArtifact := artifactFixtureWithImageURI(patchedArtifact)
	filesAPI := newSyncTestFilesAPI()

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	mockService.EXPECT().FilesAPI().Return(filesAPI)
	mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil)
	expectArtifactBuildTriggerOnly(mockService, artifactID, builtArtifact)

	model := artifactResourceModelWithSource(name, sourceDir)
	model.Source.WaitForBuild = types.BoolValue(false)

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	result, diags := testArtifactApplyCreate(context.Background(), resource, model)
	if diags.HasError() {
		t.Fatalf("create: %s", diagErrorSummary(diags))
	}
	if result.Status.ValueString() != "draft" {
		t.Fatalf("status = %q, want draft", result.Status.ValueString())
	}
}

func TestArtifactResourceSourceUpdateDraftNameOnlySkipsReupload(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDir := writeArtifactSourceTree(t, map[string]string{"main.py": "stable"})
	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-skip-upload-" + uuid.NewString()[:8]
	updatedName := "updated-" + name

	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
	patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	updatedArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, updatedName)
	updatedArtifact.Spec = patchedArtifact.Spec
	filesAPI := newSyncTestFilesAPI()

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	mockService.EXPECT().FilesAPI().Return(filesAPI).Times(1)
	mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil).Times(1)
	expectArtifactBuildAfterUpload(mockService, artifactID, artifactFixtureWithImageURI(patchedArtifact))
	mockService.EXPECT().PatchArtifact(gomock.Any(), artifactID, gomock.Any()).Return(updatedArtifact, nil)

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	state, diags := testArtifactApplyCreate(context.Background(), resource, artifactResourceModelWithSource(name, sourceDir))
	if diags.HasError() {
		t.Fatalf("create: %s", diagErrorSummary(diags))
	}

	plan := artifactResourceModelWithSource(updatedName, sourceDir)
	plan.ID = state.ID
	plan.ArtifactID = state.ArtifactID
	plan.ArtifactRepositoryID = state.ArtifactRepositoryID
	plan.Source.DirHash = state.Source.DirHash

	updated, diags := testArtifactApplyUpdate(context.Background(), resource, plan, state)
	if diags.HasError() {
		t.Fatalf("update: %s", diagErrorSummary(diags))
	}
	if updated.Name.ValueString() != updatedName {
		t.Fatalf("name = %q, want %q", updated.Name.ValueString(), updatedName)
	}
}

func TestArtifactResourceSourceUpdateDraftSourceChangeReuploads(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDirV1 := writeArtifactSourceTree(t, map[string]string{"main.py": "v1"})
	sourceDirV2 := writeArtifactSourceTree(t, map[string]string{"main.py": "v2"})
	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-reupload-" + uuid.NewString()[:8]

	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
	patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	filesAPI1 := newSyncTestFilesAPI()
	filesAPI2 := newSyncTestFilesAPI()

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	mockService.EXPECT().PatchArtifact(gomock.Any(), artifactID, gomock.Any()).Return(patchedArtifact, nil)
	gomock.InOrder(
		mockService.EXPECT().FilesAPI().Return(filesAPI1),
		mockService.EXPECT().FilesAPI().Return(filesAPI2),
	)
	gomock.InOrder(
		mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil),
		mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil),
	)
	expectArtifactBuildAfterUpload(mockService, artifactID, artifactFixtureWithImageURI(patchedArtifact))
	expectArtifactBuildAfterUpload(mockService, artifactID, artifactFixtureWithImageURI(patchedArtifact))

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	state, diags := testArtifactApplyCreate(context.Background(), resource, artifactResourceModelWithSource(name, sourceDirV1))
	if diags.HasError() {
		t.Fatalf("create: %s", diagErrorSummary(diags))
	}

	plan := artifactResourceModelWithSource(name, sourceDirV2)
	plan.ID = state.ID
	plan.ArtifactID = state.ArtifactID
	plan.ArtifactRepositoryID = state.ArtifactRepositoryID
	plan.Source.DirHash = types.StringValue("changed-hash")

	_, diags = testArtifactApplyUpdate(context.Background(), resource, plan, state)
	if diags.HasError() {
		t.Fatalf("update: %s", diagErrorSummary(diags))
	}
}

// TestArtifactResourceModifyPlanImageURIUnknownOnRebuild exercises the real ModifyPlan()
// method (not applySourceManagedImageURIToPlan directly) with a rebuilt image_uri that
// differs from the prior known state value. Without the applySourceManagedImageURIToPlan
// call in ModifyPlan, the framework's UseStateForUnknown plan modifier would carry the
// prior build's image_uri forward as "known", and Terraform would report "Provider
// produced inconsistent result after apply" once the real rebuilt value landed in state.
func TestArtifactResourceModifyPlanImageURIUnknownOnRebuild(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDirV1 := writeArtifactSourceTree(t, map[string]string{"main.py": "v1"})
	sourceDirV2 := writeArtifactSourceTree(t, map[string]string{"main.py": "v2"})
	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "modify-plan-image-uri-" + uuid.NewString()[:8]

	resource := &ArtifactResource{provider: &Provider{service: mockService}}

	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
	patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	filesAPI1 := newSyncTestFilesAPI()

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	mockService.EXPECT().FilesAPI().Return(filesAPI1)
	mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil)
	expectArtifactBuildAfterUpload(mockService, artifactID, artifactFixtureWithImageURI(patchedArtifact))

	state, diags := testArtifactApplyCreate(context.Background(), resource, artifactResourceModelWithSource(name, sourceDirV1))
	if diags.HasError() {
		t.Fatalf("create: %s", diagErrorSummary(diags))
	}
	priorImageURI := state.Spec.ContainerGroups[0].Containers[0].ImageURI
	if priorImageURI.ValueString() != artifactSourceTestImageURI {
		t.Fatalf("prior image_uri = %v, want %q", priorImageURI, artifactSourceTestImageURI)
	}

	// Simulates what Terraform Core hands ModifyPlan for an Optional+Computed attribute the
	// user never set in config: the raw plan carries the prior state value forward via
	// UseStateForUnknown, exactly like priorImageURI above.
	config := artifactResourceModelWithSource(name, sourceDirV2)
	planModel := artifactResourceModelWithSource(name, sourceDirV2)
	planModel.ID = state.ID
	planModel.ArtifactID = state.ArtifactID
	planModel.ArtifactRepositoryID = state.ArtifactRepositoryID
	planModel.Spec.ContainerGroups[0].Containers[0].ImageURI = priorImageURI

	plannedResult, diags := testArtifactApplyModifyPlan(context.Background(), resource, config, planModel, state)
	if diags.HasError() {
		t.Fatalf("modify plan: %s", diagErrorSummary(diags))
	}

	gotPlannedImageURI := plannedResult.Spec.ContainerGroups[0].Containers[0].ImageURI
	if !gotPlannedImageURI.IsUnknown() {
		t.Fatalf("planned spec.container_groups.0.containers.0.image_uri = %v, want unknown after a source change queues a rebuild", gotPlannedImageURI)
	}
}

func TestArtifactResourceSourceUpdateUploadFailure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDirV1 := writeArtifactSourceTree(t, map[string]string{"main.py": "v1"})
	sourceDirV2 := writeArtifactSourceTree(t, map[string]string{"main.py": "v2"})
	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-update-upload-fail-" + uuid.NewString()[:8]

	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
	patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	filesAPI1 := newSyncTestFilesAPI()
	filesAPI2 := newSyncTestFilesAPI()
	filesAPI2.uploadErr = fmt.Errorf("upload failed on update")

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	gomock.InOrder(
		mockService.EXPECT().FilesAPI().Return(filesAPI1),
		mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil),
	)
	expectArtifactBuildAfterUpload(mockService, artifactID, artifactFixtureWithImageURI(patchedArtifact))
	mockService.EXPECT().PatchArtifact(gomock.Any(), artifactID, gomock.Any()).Return(patchedArtifact, nil)
	mockService.EXPECT().FilesAPI().Return(filesAPI2)

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	state, diags := testArtifactApplyCreate(context.Background(), resource, artifactResourceModelWithSource(name, sourceDirV1))
	if diags.HasError() {
		t.Fatalf("create: %s", diagErrorSummary(diags))
	}

	plan := artifactResourceModelWithSource(name, sourceDirV2)
	plan.ID = state.ID
	plan.ArtifactID = state.ArtifactID
	plan.ArtifactRepositoryID = state.ArtifactRepositoryID
	plan.Source.DirHash = types.StringValue("changed-hash")

	_, diags = testArtifactApplyUpdate(context.Background(), resource, plan, state)
	if !diags.HasError() {
		t.Fatal("expected upload error on update")
	}
	if got := diagErrorSummary(diags); got != "Error uploading artifact source" {
		t.Fatalf("error summary = %q", got)
	}
}

func TestArtifactResourceSourceUpdatePatchCodeRefFailure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDirV1 := writeArtifactSourceTree(t, map[string]string{"main.py": "v1"})
	sourceDirV2 := writeArtifactSourceTree(t, map[string]string{"main.py": "v2"})
	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-update-patch-fail-" + uuid.NewString()[:8]

	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
	patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	filesAPI1 := newSyncTestFilesAPI()
	filesAPI2 := newSyncTestFilesAPI()

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	gomock.InOrder(
		mockService.EXPECT().FilesAPI().Return(filesAPI1),
		mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil),
	)
	expectArtifactBuildAfterUpload(mockService, artifactID, artifactFixtureWithImageURI(patchedArtifact))
	mockService.EXPECT().PatchArtifact(gomock.Any(), artifactID, gomock.Any()).Return(patchedArtifact, nil)
	gomock.InOrder(
		mockService.EXPECT().FilesAPI().Return(filesAPI2),
		mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("patch failed on update")),
	)

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	state, diags := testArtifactApplyCreate(context.Background(), resource, artifactResourceModelWithSource(name, sourceDirV1))
	if diags.HasError() {
		t.Fatalf("create: %s", diagErrorSummary(diags))
	}

	plan := artifactResourceModelWithSource(name, sourceDirV2)
	plan.ID = state.ID
	plan.ArtifactID = state.ArtifactID
	plan.ArtifactRepositoryID = state.ArtifactRepositoryID
	plan.Source.DirHash = types.StringValue("changed-hash")

	_, diags = testArtifactApplyUpdate(context.Background(), resource, plan, state)
	if !diags.HasError() {
		t.Fatal("expected patch error on update")
	}
	if got := diagErrorSummary(diags); got != "Error uploading artifact source" {
		t.Fatalf("error summary = %q", got)
	}
}

func TestArtifactResourceSourceUpdatePatchArtifactFailure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDir := writeArtifactSourceTree(t, map[string]string{"main.py": "stable"})
	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-patch-artifact-fail-" + uuid.NewString()[:8]
	updatedName := "updated-" + name

	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
	patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	filesAPI := newSyncTestFilesAPI()

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	mockService.EXPECT().FilesAPI().Return(filesAPI)
	mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil)
	expectArtifactBuildAfterUpload(mockService, artifactID, artifactFixtureWithImageURI(patchedArtifact))
	mockService.EXPECT().PatchArtifact(gomock.Any(), artifactID, gomock.Any()).Return(nil, fmt.Errorf("patch artifact failed"))

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	state, diags := testArtifactApplyCreate(context.Background(), resource, artifactResourceModelWithSource(name, sourceDir))
	if diags.HasError() {
		t.Fatalf("create: %s", diagErrorSummary(diags))
	}

	plan := artifactResourceModelWithSource(updatedName, sourceDir)
	plan.ID = state.ID
	plan.ArtifactID = state.ArtifactID
	plan.ArtifactRepositoryID = state.ArtifactRepositoryID
	plan.Source.DirHash = state.Source.DirHash

	_, diags = testArtifactApplyUpdate(context.Background(), resource, plan, state)
	if !diags.HasError() {
		t.Fatal("expected patch artifact error")
	}
	if got := diagErrorSummary(diags); got != "Error updating Artifact" {
		t.Fatalf("error summary = %q", got)
	}
}

func TestArtifactResourceSourceUpdateCreateArtifactFailure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDir := writeArtifactSourceTree(t, map[string]string{"main.py": "app"})
	lockedArtifactID := uuid.NewString()
	repoID := uuid.NewString()
	name := "source-create-version-fail-" + uuid.NewString()[:8]

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("create draft version failed"))

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	state := artifactResourceModelWithSource(name, sourceDir)
	state.Status = types.StringValue("locked")
	state.ArtifactID = types.StringValue(lockedArtifactID)
	state.ArtifactRepositoryID = types.StringValue(repoID)
	state.Source.DirHash = types.StringValue("old-hash")
	_ = setImageBuildConfigCodeRef(state.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig, &ArtifactCodeRefModel{
		CatalogID:        types.StringValue(artifactSourceTestCatalogID),
		CatalogVersionID: types.StringValue(artifactSourceTestVersionID),
	})

	plan := artifactResourceModelWithSource(name, sourceDir)
	plan.Status = types.StringValue("locked")
	plan.ArtifactID = state.ArtifactID
	plan.ArtifactRepositoryID = state.ArtifactRepositoryID
	plan.Source.DirHash = types.StringValue("new-hash")

	_, diags := testArtifactApplyUpdate(context.Background(), resource, plan, state)
	if !diags.HasError() {
		t.Fatal("expected create draft clone error")
	}
	if got := diagErrorSummary(diags); got != "Error creating draft Artifact for source update" {
		t.Fatalf("error summary = %q", got)
	}
}

func TestArtifactResourceSourceReadGetArtifactError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	artifactID := uuid.NewString()
	mockService.EXPECT().GetArtifact(gomock.Any(), artifactID).Return(nil, fmt.Errorf("read failed"))

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	_, diags, removed := testArtifactApplyRead(context.Background(), resource, ArtifactResourceModel{
		ArtifactID: types.StringValue(artifactID),
		Source:     &ArtifactSourceModel{Dir: types.StringValue(writeArtifactSourceTree(t, map[string]string{"main.py": "x"}))},
	})
	if removed {
		t.Fatal("expected resource to remain when read fails with API error")
	}
	if !diags.HasError() {
		t.Fatal("expected read error")
	}
	if !strings.Contains(diagErrorSummary(diags), "Error getting Artifact") {
		t.Fatalf("error summary = %q", diagErrorSummary(diags))
	}
}

func TestArtifactResourceSourceReadNotFoundRemovesState(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	artifactID := uuid.NewString()
	mockService.EXPECT().GetArtifact(gomock.Any(), artifactID).Return(nil, client.NewNotFoundError("artifact"))

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	_, diags, removed := testArtifactApplyRead(context.Background(), resource, ArtifactResourceModel{
		ArtifactID: types.StringValue(artifactID),
		Source:     &ArtifactSourceModel{Dir: types.StringValue(writeArtifactSourceTree(t, map[string]string{"main.py": "x"}))},
	})
	if !removed {
		t.Fatal("expected read not-found to mark resource removed")
	}
	if len(diags.Warnings()) == 0 {
		t.Fatal("expected not-found warning")
	}
	if !strings.Contains(diags.Warnings()[0].Summary(), "Artifact not found") {
		t.Fatalf("warning = %q", diags.Warnings()[0].Summary())
	}
}

func TestArtifactResourceSourceReadPreservesDirHash(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDir := writeArtifactSourceTree(t, map[string]string{
		"user_tools.py": "# @dr_mcp_tool(tags={\"user\"})\nasync def user_tool_example():\n    pass\n",
	})
	appliedHash, err := computeArtifactSourceDirHash(&ArtifactResourceModel{
		Source: &ArtifactSourceModel{Dir: types.StringValue(sourceDir)},
	})
	if err != nil {
		t.Fatal(err)
	}

	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	repoIDPtr := repoID
	name := "source-read-hash-" + uuid.NewString()[:8]

	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoIDPtr, name)
	patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	mockService.EXPECT().GetArtifact(gomock.Any(), artifactID).Return(patchedArtifact, nil)

	// Simulate a post-apply edit (uncommenting @dr_mcp_tool) before terraform plan.
	if err := os.WriteFile(filepath.Join(sourceDir, "user_tools.py"), []byte("@dr_mcp_tool(tags={\"user\"})\nasync def user_tool_example():\n    pass\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	currentHash, err := computeArtifactSourceDirHash(&ArtifactResourceModel{
		Source: &ArtifactSourceModel{Dir: types.StringValue(sourceDir)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if currentHash.Equal(appliedHash) {
		t.Fatal("expected dir_hash to change after uncommenting the decorator")
	}

	resource := &ArtifactResource{provider: &Provider{service: mockService}}
	result, diags, removed := testArtifactApplyRead(context.Background(), resource, ArtifactResourceModel{
		ArtifactID: types.StringValue(artifactID),
		Source: &ArtifactSourceModel{
			Dir:     types.StringValue(sourceDir),
			DirHash: appliedHash,
		},
	})
	if removed || diags.HasError() {
		t.Fatalf("read failed: removed=%v err=%v", removed, diagErrorSummary(diags))
	}
	if !result.Source.DirHash.Equal(appliedHash) {
		t.Fatalf("Read must preserve last-applied dir_hash so plan can detect local file changes, got %q want %q", result.Source.DirHash.ValueString(), appliedHash.ValueString())
	}

	plan := artifactResourceModelWithSource(name, sourceDir)
	plan.ArtifactID = types.StringValue(artifactID)
	plan.Source.DirHash = currentHash
	if !artifactSourceNeedsUpload(&plan, &result, artifactID, artifactID) {
		t.Fatal("expected source upload after local file edit when Read preserves last-applied dir_hash")
	}
}

func artifactConfigWithSource(name, status, dir string) string {
	return fmt.Sprintf(`
resource "datarobot_artifact" "test" {
  name   = %q
  status = %q
  source = { dir = %q }
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
}`, name, status, dir)
}

func artifactConfigWithSourceType(name, status, artifactType, dir string) string {
	return fmt.Sprintf(`
resource "datarobot_artifact" "test" {
  name   = %q
  type   = %q
  status = %q
  source = { dir = %q }
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
}`, name, artifactType, status, dir)
}

func artifactConfigWithSourceAndCodeRef(dir string) string {
	return fmt.Sprintf(`
resource "datarobot_artifact" "test" {
  name   = "source-code-ref-conflict"
  status = "draft"
  source = { dir = %q }
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
}`, dir)
}

func artifactConfigWithSourceImageURIONly(dir string) string {
	return fmt.Sprintf(`
resource "datarobot_artifact" "test" {
  name   = "source-image-uri-only"
  status = "draft"
  source = { dir = %q }
  spec = {
    container_groups = [{
      containers = [{
        primary   = true
        port      = 8080
        image_uri = "nginx:latest"
      }]
    }]
  }
}`, dir)
}

func TestDecodePlanArtifactModelUnknownCodeRef(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schema := testArtifactResourceSchema(t)
	dir := t.TempDir()
	stateCodeRef := &ArtifactCodeRefModel{
		CatalogID:        types.StringValue(artifactSourceTestCatalogID),
		CatalogVersionID: types.StringValue(artifactSourceTestVersionID),
	}

	tests := []struct {
		name      string
		planModel *ArtifactResourceModel
		state     *ArtifactResourceModel
		check     func(t *testing.T, decoded ArtifactResourceModel)
	}{
		{
			name: "create decodes unknown code_ref as null",
			planModel: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig()), func(m *ArtifactResourceModel) {
				m.Name = types.StringValue("create-decode")
			}),
			check: func(t *testing.T, decoded ArtifactResourceModel) {
				codeRef := imageBuildConfigCodeRef(decoded.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig)
				if codeRef != nil && (IsKnown(codeRef.CatalogID) || IsKnown(codeRef.CatalogVersionID)) {
					t.Fatalf("expected null code_ref on create, got %#v", codeRef)
				}
			},
		},
		{
			name: "update decodes unknown code_ref from state",
			planModel: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig()), func(m *ArtifactResourceModel) {
				m.Name = types.StringValue("update-decode")
				m.ArtifactID = types.StringValue("artifact-1")
				m.Source.DirHash = types.StringValue("hash-b")
			}),
			state: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithCodeRef(stateCodeRef)), func(m *ArtifactResourceModel) {
				m.Name = types.StringValue("update-decode")
				m.ArtifactID = types.StringValue("artifact-1")
				m.Source.DirHash = types.StringValue("hash-a")
			}),
			check: func(t *testing.T, decoded ArtifactResourceModel) {
				codeRef := imageBuildConfigCodeRef(decoded.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig)
				if codeRef == nil {
					t.Fatal("expected code_ref copied from state")
				}
				if got := codeRef.CatalogID.ValueString(); got != artifactSourceTestCatalogID {
					t.Fatalf("catalog_id = %q, want %q", got, artifactSourceTestCatalogID)
				}
				if got := codeRef.CatalogVersionID.ValueString(); got != artifactSourceTestVersionID {
					t.Fatalf("catalog_version_id = %q, want %q", got, artifactSourceTestVersionID)
				}
			},
		},
		{
			name: "update decodes unknown code_ref from primary after container reorder",
			planModel: testSourcePlanModel(t, dir, testDraftSourceSpec(testSidecarWithBuildConfig(), testPrimaryWithBuildConfig()), func(m *ArtifactResourceModel) {
				m.Name = types.StringValue("update-decode-reorder")
				m.ArtifactID = types.StringValue("artifact-1")
				m.Source.DirHash = types.StringValue("hash-a")
			}),
			state: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithCodeRef(stateCodeRef), testSidecarWithBuildConfig()), func(m *ArtifactResourceModel) {
				m.Name = types.StringValue("update-decode-reorder")
				m.ArtifactID = types.StringValue("artifact-1")
				m.Source.DirHash = types.StringValue("hash-a")
			}),
			check: func(t *testing.T, decoded ArtifactResourceModel) {
				codeRef := imageBuildConfigCodeRef(decoded.Spec.ContainerGroups[0].Containers[1].ImageBuildConfig)
				if codeRef == nil {
					t.Fatal("expected code_ref copied from primary in state")
				}
				if got := codeRef.CatalogID.ValueString(); got != artifactSourceTestCatalogID {
					t.Fatalf("catalog_id = %q, want %q", got, artifactSourceTestCatalogID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan := testArtifactPlanWithUnknownCodeRef(t, ctx, schema, tt.planModel)

			var decoded ArtifactResourceModel
			if diags := decodePlanArtifactModel(ctx, plan, tt.state, &decoded); diags.HasError() {
				t.Fatalf("decodePlanArtifactModel: %s", diagErrorSummary(diags))
			}

			tt.check(t, decoded)
		})
	}
}

func testArtifactPlanWithUnknownCodeRef(t *testing.T, ctx context.Context, schema schema.Schema, model *ArtifactResourceModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(ctx, model); diags.HasError() {
		t.Fatalf("plan.Set: %s", diagErrorSummary(diags))
	}

	gi, ci := primaryContainerIndex(model)
	if gi < 0 {
		t.Fatal("expected primary container with image_build_config")
	}
	codeRefPath := path.Root("spec").
		AtName("container_groups").AtListIndex(gi).
		AtName("containers").AtListIndex(ci).
		AtName("image_build_config").AtName("code_ref")
	if diags := plan.SetAttribute(ctx, codeRefPath, types.ObjectUnknown(artifactCodeRefObjectType.AttrTypes)); diags.HasError() {
		t.Fatalf("plan.SetAttribute(code_ref): %s", diagErrorSummary(diags))
	}

	return plan
}

func primaryContainerIndex(model *ArtifactResourceModel) (gi, ci int) {
	if model == nil || model.Spec == nil {
		return -1, -1
	}
	for groupIdx, group := range model.Spec.ContainerGroups {
		for containerIdx, container := range group.Containers {
			if container.ImageBuildConfig == nil {
				continue
			}
			if artifactContainerIsPrimary(container, group) {
				return groupIdx, containerIdx
			}
		}
	}
	return -1, -1
}

func testArtifactResourceSchema(t *testing.T) schema.Schema {
	t.Helper()

	schemaResponse := &tfresource.SchemaResponse{}
	NewArtifactResource().Schema(context.Background(), tfresource.SchemaRequest{}, schemaResponse)
	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("artifact schema: %s", diagErrorSummary(schemaResponse.Diagnostics))
	}
	return schemaResponse.Schema
}

func TestArtifactImageBuildConfigNonPrimaryRejected(t *testing.T) {
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
		Steps: []resource.TestStep{
			{
				Config: `
resource "datarobot_artifact" "test" {
  name   = "sidecar-build-config"
  status = "draft"
  spec = {
    container_groups = [{
      containers = [
        {
          name    = "primary"
          primary = true
          port    = 8080
          image_uri = "nginx:1.25"
        },
        {
          name    = "sidecar"
          primary = false
          image_build_config = {
            dockerfile = { source = "provided" }
          }
        },
      ]
    }]
  }
}`,
				ExpectError: regexp.MustCompile("Unsupported on non-primary container"),
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
					DataRobot: client.ArtifactDataRobotCodeRef{
						CatalogID:        catalogID,
						CatalogVersionID: catalogVersionID,
					},
				},
				Dockerfile: &client.ArtifactDockerfileConfig{
					Source: "provided",
					Path:   "./Dockerfile",
				},
			},
		}, nil)

		cfg := model.ImageBuildConfig
		if cfg == nil {
			t.Fatal("expected image_build_config in state model")
		}
		codeRef := imageBuildConfigCodeRef(cfg)
		if codeRef == nil {
			t.Fatal("expected code_ref in state model")
		}
		if got := codeRef.CatalogID.ValueString(); got != catalogID {
			t.Fatalf("catalog_id: got %q, want %q", got, catalogID)
		}
		if got := codeRef.CatalogVersionID.ValueString(); got != catalogVersionID {
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
				Dockerfile: &client.ArtifactDockerfileConfig{
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
		if !cfg.Dockerfile.Path.IsNull() {
			t.Fatalf("dockerfile.path: got %v, want null for generated dockerfile", cfg.Dockerfile.Path)
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

func TestArtifactNeedsNewVersion_dockerfileDefaults(t *testing.T) {
	base := ArtifactResourceModel{
		Name:                 types.StringValue("my-artifact"),
		Description:          types.StringValue("desc"),
		ArtifactRepositoryID: types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaa"),
		Spec: &ArtifactSpecModel{
			ContainerGroups: []ArtifactContainerGroupModel{{
				Containers: []ArtifactContainerModel{{
					Name:     types.StringValue("app"),
					ImageURI: types.StringValue("registry.example.com/app:v1"),
					Primary:  types.BoolValue(true),
					ImageBuildConfig: &ArtifactImageBuildConfigModel{
						CodeRef: artifactCodeRefObject(&ArtifactCodeRefModel{
							CatalogID:        types.StringValue("bbbbbbbbbbbbbbbbbbbbbbbb"),
							CatalogVersionID: types.StringValue("cccccccccccccccccccccccc"),
						}),
					},
				}},
			}},
		},
	}

	// Plan omits the nested dockerfile block; state was refreshed with API defaults.
	state := base
	state.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig.Dockerfile = &ArtifactDockerfileModel{
		Source: types.StringValue("provided"),
		Path:   types.StringValue("./Dockerfile"),
	}

	if artifactNeedsNewVersion(base, state) {
		t.Fatal("expected omitted dockerfile block to match API-provided defaults")
	}
}

func TestArtifactNeedsNewVersion_a2aEnabled(t *testing.T) {
	t.Parallel()

	base := ArtifactResourceModel{
		Name:                 types.StringValue("my-artifact"),
		Description:          types.StringValue("desc"),
		ArtifactRepositoryID: types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaa"),
		Spec: &ArtifactSpecModel{
			A2AEnabled: types.BoolValue(false),
			ContainerGroups: []ArtifactContainerGroupModel{{
				Containers: []ArtifactContainerModel{{
					ImageURI: types.StringValue("nginx:latest"),
					Primary:  types.BoolValue(true),
				}},
			}},
		},
	}

	unchanged := base
	if artifactNeedsNewVersion(unchanged, base) {
		t.Fatal("expected matching a2a_enabled not to force a new version")
	}

	enabled := base
	enabled.Spec = &ArtifactSpecModel{
		A2AEnabled:      types.BoolValue(true),
		ContainerGroups: base.Spec.ContainerGroups,
	}
	if !artifactNeedsNewVersion(enabled, base) {
		t.Fatal("expected a2a_enabled change to force a new version")
	}

	cleared := base
	cleared.Spec = &ArtifactSpecModel{
		A2AEnabled:      types.BoolNull(),
		ContainerGroups: base.Spec.ContainerGroups,
	}
	if artifactNeedsNewVersion(cleared, base) {
		t.Fatal("expected null and false a2a_enabled not to force a new version")
	}

	imported := base
	imported.Spec = &ArtifactSpecModel{
		A2AEnabled:      types.BoolNull(),
		ContainerGroups: base.Spec.ContainerGroups,
	}
	explicitFalse := base
	explicitFalse.Spec = &ArtifactSpecModel{
		A2AEnabled:      types.BoolValue(false),
		ContainerGroups: base.Spec.ContainerGroups,
	}
	if artifactNeedsNewVersion(explicitFalse, imported) {
		t.Fatal("expected adding explicit a2a_enabled = false not to force a new version")
	}

	on := base
	on.Spec = &ArtifactSpecModel{
		A2AEnabled:      types.BoolValue(true),
		ContainerGroups: base.Spec.ContainerGroups,
	}
	off := base
	off.Spec = &ArtifactSpecModel{
		A2AEnabled:      types.BoolNull(),
		ContainerGroups: base.Spec.ContainerGroups,
	}
	if !artifactNeedsNewVersion(off, on) {
		t.Fatal("expected turning a2a_enabled off to force a new version")
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

	if containersEqual(base, changed, false, false) {
		t.Fatal("expected image_build_config change to make containers unequal")
	}
}

func TestArtifactNeedsNewVersion_sidecarImageChangeWithSource(t *testing.T) {
	t.Parallel()

	base := ArtifactResourceModel{
		Name:                 types.StringValue("my-artifact"),
		Description:          types.StringValue("desc"),
		ArtifactRepositoryID: types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaa"),
		Source: &ArtifactSourceModel{
			Dir:     types.StringValue("/tmp/fake-dir"),
			DirHash: types.StringValue("hash-v1"),
		},
		Spec: &ArtifactSpecModel{
			ContainerGroups: []ArtifactContainerGroupModel{{
				Containers: []ArtifactContainerModel{
					{
						Name:    types.StringValue("main"),
						Primary: types.BoolValue(true),
						ImageBuildConfig: &ArtifactImageBuildConfigModel{
							CodeRef: artifactCodeRefObject(&ArtifactCodeRefModel{
								CatalogID:        types.StringValue("bbbbbbbbbbbbbbbbbbbbbbbb"),
								CatalogVersionID: types.StringValue("cccccccccccccccccccccccc"),
							}),
						},
					},
					{
						Name:     types.StringValue("sidecar"),
						Primary:  types.BoolValue(false),
						ImageURI: types.StringValue("sidecar:v1"),
					},
				},
			}},
		},
	}

	state := base
	plan := base
	plan.Spec = &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{{
			Containers: []ArtifactContainerModel{
				base.Spec.ContainerGroups[0].Containers[0],
				{
					Name:     types.StringValue("sidecar"),
					Primary:  types.BoolValue(false),
					ImageURI: types.StringValue("sidecar:v2"),
				},
			},
		}},
	}

	if !artifactNeedsNewVersion(plan, state) {
		t.Fatal("expected changing sidecar image_uri on locked artifact with source to need new version")
	}
}

func TestArtifactContainerToClient_routes(t *testing.T) {
	container := artifactContainerToClient(ArtifactContainerModel{
		ImageURI: types.StringValue("registry.example/mcp:latest"),
		Primary:  types.BoolValue(true),
		Port:     types.Int64Value(8080),
		Routes: []ArtifactContainerRouteModel{
			{
				Path: types.StringValue("/status"),
				Auth: types.StringValue(client.RouteAuthDisabled),
			},
		},
	})

	if len(container.Routes) != 1 {
		t.Fatalf("routes length: got %d, want 1", len(container.Routes))
	}
	if container.Routes[0].Path != "/status" {
		t.Fatalf("unexpected route path: %q", container.Routes[0].Path)
	}
	if container.Routes[0].Auth != client.RouteAuthDisabled {
		t.Fatalf("unexpected route auth: %q", container.Routes[0].Auth)
	}
}

func TestArtifactContainerToClient_routesOmittedWhenUnset(t *testing.T) {
	container := artifactContainerToClient(ArtifactContainerModel{
		ImageURI: types.StringValue("registry.example/mcp:latest"),
		Primary:  types.BoolValue(true),
		Port:     types.Int64Value(8080),
	})

	if container.Routes != nil {
		t.Fatalf("expected no routes, got %v", container.Routes)
	}
}

func TestLoadContainerFromAPI_routes(t *testing.T) {
	model := loadContainerFromAPI(client.ArtifactContainer{
		ImageURI: "registry.example/mcp:latest",
		Routes: []client.ArtifactContainerRoute{
			{Path: "/status", Auth: client.RouteAuthDisabled},
		},
	}, nil)

	if len(model.Routes) != 1 {
		t.Fatalf("routes length: got %d, want 1", len(model.Routes))
	}
	if got := model.Routes[0].Path.ValueString(); got != "/status" {
		t.Fatalf("unexpected route path: %q", got)
	}
	if got := model.Routes[0].Auth.ValueString(); got != client.RouteAuthDisabled {
		t.Fatalf("unexpected route auth: %q", got)
	}
}

func TestContainersEqual_routes(t *testing.T) {
	base := ArtifactContainerModel{
		ImageURI: types.StringValue("registry.example/mcp:latest"),
		Primary:  types.BoolValue(true),
		Port:     types.Int64Value(8080),
	}
	changed := base
	changed.Routes = []ArtifactContainerRouteModel{
		{
			Path: types.StringValue("/status"),
			Auth: types.StringValue(client.RouteAuthDisabled),
		},
	}

	if containersEqual(base, changed, false, false) {
		t.Fatal("expected routes change to make containers unequal")
	}
	if !containersEqual(changed, changed, false, false) {
		t.Fatal("expected identical routes to be equal")
	}

	authChanged := changed
	authChanged.Routes = []ArtifactContainerRouteModel{
		{
			Path: types.StringValue("/status"),
			Auth: types.StringValue(client.RouteAuthRequired),
		},
	}
	if containersEqual(changed, authChanged, false, false) {
		t.Fatal("expected auth-only route change to make containers unequal")
	}
}

func TestLoadContainerFromAPI_routesEmptyListRoundTrip(t *testing.T) {
	// `routes = []` is dropped from the request by omitempty and comes back absent.
	// Without the prior-state fallback the applied state would be null and Terraform
	// would fail with "Provider produced inconsistent result after apply".
	prior := &ArtifactContainerModel{Routes: []ArtifactContainerRouteModel{}}

	model := loadContainerFromAPI(client.ArtifactContainer{
		ImageURI: "registry.example/mcp:latest",
	}, prior)

	if model.Routes == nil {
		t.Fatal("expected empty routes list to be preserved, got nil")
	}
	if len(model.Routes) != 0 {
		t.Fatalf("routes length: got %d, want 0", len(model.Routes))
	}
}

func TestLoadContainerFromAPI_routesUnsetStaysNull(t *testing.T) {
	model := loadContainerFromAPI(client.ArtifactContainer{
		ImageURI: "registry.example/mcp:latest",
	}, &ArtifactContainerModel{})

	if model.Routes != nil {
		t.Fatalf("expected nil routes, got %v", model.Routes)
	}
}

func TestLoadContainerIntoDataSourceModel_routesEmptyList(t *testing.T) {
	// Data source lists render as null when nil, which breaks `length(...routes)`
	// in a config. environment_vars normalizes the same way.
	model := loadContainerIntoDataSourceModel(client.ArtifactContainer{
		ImageURI: "registry.example/mcp:latest",
	})

	if model.Routes == nil {
		t.Fatal("expected empty routes list, got nil")
	}
	if len(model.Routes) != 0 {
		t.Fatalf("routes length: got %d, want 0", len(model.Routes))
	}
}

func TestIntegrationArtifactDraftImageBuildConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

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
						Dockerfile: &client.ArtifactDockerfileConfig{Source: "provided", Path: "./Dockerfile"},
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

func artifactResourceConfigWithStatusAndImage(name, status, imageURI string) string {
	return fmt.Sprintf(`
resource "datarobot_artifact" "test" {
  name        = %q
  description = "test artifact description"
  type        = "service"
  status      = %q
%s
}
`, name, status, artifactTestContainerSpecBlock(imageURI))
}

func artifactFixtureWithStatusAndImage(id string, repoID *string, name string, status client.ArtifactStatus, imageURI string) *client.Artifact {
	artifact := artifactFixtureWithStatus(id, repoID, name, status)
	artifact.Spec.ContainerGroups[0].Containers[0].ImageURI = imageURI
	return artifact
}

func TestArtifactSpecToClientA2AEnabled(t *testing.T) {
	t.Parallel()

	spec := ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{},
		A2AEnabled:      types.BoolValue(true),
	}

	got := artifactSpecToClient(spec, client.ArtifactTypeAgent)
	if got.A2AEnabled == nil || !*got.A2AEnabled {
		t.Fatalf("agent spec A2AEnabled = %v, want true", got.A2AEnabled)
	}

	got = artifactSpecToClient(spec, client.ArtifactTypeService)
	if got.A2AEnabled != nil {
		t.Fatalf("service spec A2AEnabled = %v, want nil", got.A2AEnabled)
	}

	omitted := ArtifactSpecModel{ContainerGroups: []ArtifactContainerGroupModel{}}
	got = artifactSpecToClient(omitted, client.ArtifactTypeAgent)
	if got.A2AEnabled == nil || *got.A2AEnabled {
		t.Fatalf("omitted a2a_enabled A2AEnabled = %v, want false", got.A2AEnabled)
	}

	disabled := ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{},
		A2AEnabled:      types.BoolValue(false),
	}
	got = artifactSpecToClient(disabled, client.ArtifactTypeAgent)
	if got.A2AEnabled == nil || *got.A2AEnabled {
		t.Fatalf("explicit false A2AEnabled = %v, want false", got.A2AEnabled)
	}
}

func TestLoadArtifactSpecFromAPIA2AEnabled(t *testing.T) {
	t.Parallel()

	trueVal := true
	falseVal := false
	apiSpec := client.ArtifactSpec{ContainerGroups: []client.ArtifactContainerGroup{}}

	t.Run("null prior and API false stay null", func(t *testing.T) {
		apiSpec.A2AEnabled = &falseVal
		got := loadArtifactSpecFromAPI(apiSpec, nil)
		if !got.A2AEnabled.IsNull() {
			t.Fatalf("A2AEnabled = %v, want null", got.A2AEnabled)
		}
	})

	t.Run("null prior and API true surfaces true", func(t *testing.T) {
		apiSpec.A2AEnabled = &trueVal
		got := loadArtifactSpecFromAPI(apiSpec, nil)
		if got.A2AEnabled.IsNull() || !got.A2AEnabled.ValueBool() {
			t.Fatalf("A2AEnabled = %v, want true", got.A2AEnabled)
		}
	})

	t.Run("omitted config stays null after API false", func(t *testing.T) {
		apiSpec.A2AEnabled = &falseVal
		prior := &ArtifactSpecModel{A2AEnabled: types.BoolNull()}
		got := loadArtifactSpecFromAPI(apiSpec, prior)
		if !got.A2AEnabled.IsNull() {
			t.Fatalf("A2AEnabled = %v, want null", got.A2AEnabled)
		}
	})

	t.Run("omitted config surfaces API true as drift", func(t *testing.T) {
		apiSpec.A2AEnabled = &trueVal
		prior := &ArtifactSpecModel{A2AEnabled: types.BoolNull()}
		got := loadArtifactSpecFromAPI(apiSpec, prior)
		if got.A2AEnabled.IsNull() || !got.A2AEnabled.ValueBool() {
			t.Fatalf("A2AEnabled = %v, want true", got.A2AEnabled)
		}
	})

	t.Run("prior false round-trips API false", func(t *testing.T) {
		apiSpec.A2AEnabled = &falseVal
		prior := &ArtifactSpecModel{A2AEnabled: types.BoolValue(false)}
		got := loadArtifactSpecFromAPI(apiSpec, prior)
		if got.A2AEnabled.IsNull() || got.A2AEnabled.ValueBool() {
			t.Fatalf("A2AEnabled = %v, want false", got.A2AEnabled)
		}
	})

	t.Run("prior true round-trips API true", func(t *testing.T) {
		apiSpec.A2AEnabled = &trueVal
		prior := &ArtifactSpecModel{A2AEnabled: types.BoolValue(true)}
		got := loadArtifactSpecFromAPI(apiSpec, prior)
		if got.A2AEnabled.IsNull() || !got.A2AEnabled.ValueBool() {
			t.Fatalf("A2AEnabled = %v, want true", got.A2AEnabled)
		}
	})
}

func TestValidateArtifactA2AEnabled(t *testing.T) {
	t.Parallel()

	specEnabled := &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{},
		A2AEnabled:      types.BoolValue(true),
	}

	tests := []struct {
		name        string
		data        ArtifactResourceModel
		wantSummary string
	}{
		{
			name: "agent with a2a_enabled",
			data: ArtifactResourceModel{
				Type: types.StringValue(string(client.ArtifactTypeAgent)),
				Spec: specEnabled,
			},
		},
		{
			name: "service with a2a_enabled",
			data: ArtifactResourceModel{
				Type: types.StringValue(string(client.ArtifactTypeService)),
				Spec: specEnabled,
			},
			wantSummary: "Unsupported a2a_enabled",
		},
		{
			name: "nim with a2a_enabled",
			data: ArtifactResourceModel{
				Type: types.StringValue(string(client.ArtifactTypeNim)),
				Spec: specEnabled,
			},
			wantSummary: "Unsupported a2a_enabled",
		},
		{
			name: "service without a2a_enabled",
			data: ArtifactResourceModel{
				Type: types.StringValue(string(client.ArtifactTypeService)),
				Spec: &ArtifactSpecModel{ContainerGroups: []ArtifactContainerGroupModel{}},
			},
		},
		{
			name: "unknown type with a2a_enabled",
			data: ArtifactResourceModel{
				Type: types.StringUnknown(),
				Spec: specEnabled,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &tfresource.ValidateConfigResponse{}
			validateArtifactA2AEnabled(resp, tt.data)

			if tt.wantSummary == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected no errors, got: %v", resp.Diagnostics.Errors())
				}
				return
			}

			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected validation error %q", tt.wantSummary)
			}
			if !strings.Contains(resp.Diagnostics.Errors()[0].Summary(), tt.wantSummary) {
				t.Fatalf("expected summary %q, got %q", tt.wantSummary, resp.Diagnostics.Errors()[0].Summary())
			}
		})
	}
}

func TestArtifactA2AEnabledConfigValidation(t *testing.T) {
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
		Steps: []resource.TestStep{
			{
				Config: `
resource "datarobot_artifact" "test" {
  name = "service-a2a"
  type = "service"
  spec = {
    a2a_enabled = true
    container_groups = [{
      containers = [{
        image_uri = "nginx:latest"
        primary   = true
        port      = 8080
      }]
    }]
  }
}`,
				ExpectError: regexp.MustCompile("Unsupported a2a_enabled"),
			},
		},
	})
}
