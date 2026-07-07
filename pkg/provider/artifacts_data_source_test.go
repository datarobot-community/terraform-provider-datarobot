package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccArtifactsDataSource(t *testing.T) {
	t.Parallel()
	name := "test-artifacts-ds-" + uuid.NewString()[:8]
	testArtifactsDataSource(t, name, "nginx:latest", false)
}

func TestIntegrationArtifactsDataSource(t *testing.T) {
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
	name := "test-artifacts-ds-" + uuid.NewString()[:8]
	imageURI := "nginx:latest"
	repoIDPtr := repoID
	artifact := artifactFixture(artifactID, &repoIDPtr, name)

	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		Return(artifact, nil)
	mockService.EXPECT().
		GetArtifact(gomock.Any(), artifactID).
		Return(artifact, nil).
		AnyTimes()
	mockService.EXPECT().
		ListArtifacts(gomock.Any(), gomock.Any()).
		Return([]client.Artifact{*artifact}, nil).
		AnyTimes()
	mockService.EXPECT().
		DeleteArtifactRepository(gomock.Any(), repoID).
		Return(nil)

	testArtifactsDataSource(t, name, imageURI, true)
}

func TestIntegrationArtifactsDataSourceWithStatusFilter(t *testing.T) {
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
	name := "test-artifacts-ds-" + uuid.NewString()[:8]
	imageURI := "nginx:latest"
	repoIDPtr := repoID
	artifact := artifactFixture(artifactID, &repoIDPtr, name)

	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		Return(artifact, nil)
	mockService.EXPECT().
		GetArtifact(gomock.Any(), artifactID).
		Return(artifact, nil).
		AnyTimes()
	mockService.EXPECT().
		ListArtifacts(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ interface{}, req *client.ListArtifactsRequest) ([]client.Artifact, error) {
			if req.Status != string(client.ArtifactStatusLocked) {
				t.Fatalf("expected status filter %q, got %q", client.ArtifactStatusLocked, req.Status)
			}
			return []client.Artifact{*artifact}, nil
		}).
		AnyTimes()
	mockService.EXPECT().
		DeleteArtifactRepository(gomock.Any(), repoID).
		Return(nil)

	dataSourceName := "data.datarobot_artifacts.locked"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: artifactsDataSourceConfigWithStatus(name, imageURI, "locked"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "status", "locked"),
					resource.TestCheckResourceAttr(dataSourceName, "artifacts.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "artifacts.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "artifacts.0.status", "locked"),
				),
			},
		},
	})
}

func TestIntegrationArtifactsDataSourceWithLimit(t *testing.T) {
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
	name := "test-artifacts-ds-" + uuid.NewString()[:8]
	imageURI := "nginx:latest"
	repoIDPtr := repoID
	artifact := artifactFixture(artifactID, &repoIDPtr, name)

	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		Return(artifact, nil)
	mockService.EXPECT().
		GetArtifact(gomock.Any(), artifactID).
		Return(artifact, nil).
		AnyTimes()
	mockService.EXPECT().
		ListArtifacts(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ interface{}, req *client.ListArtifactsRequest) ([]client.Artifact, error) {
			if req.Limit != 1 {
				t.Fatalf("expected limit 1, got %d", req.Limit)
			}
			return []client.Artifact{*artifact}, nil
		}).
		AnyTimes()
	mockService.EXPECT().
		DeleteArtifactRepository(gomock.Any(), repoID).
		Return(nil)

	dataSourceName := "data.datarobot_artifacts.limited"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: artifactsDataSourceConfigWithLimit(name, imageURI, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "limit", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "artifacts.#", "1"),
				),
			},
		},
	})
}

func TestIntegrationArtifactsDataSourceEmpty(t *testing.T) {
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

	mockService.EXPECT().
		ListArtifacts(gomock.Any(), gomock.Any()).
		Return([]client.Artifact{}, nil).
		AnyTimes()

	dataSourceName := "data.datarobot_artifacts.empty"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testProviderConfigBlock() + `
data "datarobot_artifacts" "empty" {
  status = "draft"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "artifacts.#", "0"),
				),
			},
		},
	})
}

func testArtifactsDataSource(t *testing.T, name, imageURI string, isMock bool) {
	t.Helper()

	dataSourceName := "data.datarobot_artifacts.all"
	resourceName := "datarobot_artifact.test"

	var steps []resource.TestStep
	if isMock {
		steps = []resource.TestStep{
			{
				Config: artifactsDataSourceConfig(name, imageURI),
				Check:  artifactsDataSourceMockChecks(dataSourceName, name, imageURI),
			},
		}
	} else {
		steps = []resource.TestStep{
			{
				Config: artifactResourceOnlyConfig(name, imageURI),
			},
			{
				Config: artifactsDataSourceConfig(name, imageURI),
				Check:  artifactsDataSourceAcceptanceChecks(dataSourceName, resourceName, name, imageURI),
			},
		}
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               isMock,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    steps,
	})
}

func artifactResourceOnlyConfig(name, imageURI string) string {
	return fmt.Sprintf(`
%s

%s
`, testProviderConfigBlock(), artifactResourceConfig(name, imageURI))
}

func artifactsDataSourceConfig(name, imageURI string) string {
	return fmt.Sprintf(`
%s

resource "datarobot_artifact" "test" {
  name        = %q
  description = "test artifact description"
  type        = "service"
%s
}

data "datarobot_artifacts" "all" {
  depends_on = [datarobot_artifact.test]
}
`, testProviderConfigBlock(), name, artifactTestContainerSpecBlock(imageURI))
}

func artifactsDataSourceConfigWithStatus(name, imageURI, status string) string {
	return fmt.Sprintf(`
%s

resource "datarobot_artifact" "test" {
  name        = %q
  description = "test artifact description"
  type        = "service"
%s
}

data "datarobot_artifacts" "locked" {
  status = %q
  depends_on = [datarobot_artifact.test]
}
`, testProviderConfigBlock(), name, artifactTestContainerSpecBlock(imageURI), status)
}

func artifactsDataSourceConfigWithLimit(name, imageURI string, limit int) string {
	return fmt.Sprintf(`
%s

resource "datarobot_artifact" "test" {
  name        = %q
  description = "test artifact description"
  type        = "service"
%s
}

data "datarobot_artifacts" "limited" {
  limit = %d
  depends_on = [datarobot_artifact.test]
}
`, testProviderConfigBlock(), name, artifactTestContainerSpecBlock(imageURI), limit)
}

func artifactsDataSourceMockChecks(dataSourceName, name, imageURI string) resource.TestCheckFunc {
	const containerPrefix = "artifacts.0.spec.container_groups.0.containers.0"

	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(dataSourceName, "artifacts.#", "1"),
		resource.TestCheckResourceAttr(dataSourceName, "artifacts.0.name", name),
		resource.TestCheckResourceAttr(dataSourceName, "artifacts.0.type", "service"),
		resource.TestCheckResourceAttrSet(dataSourceName, "artifacts.0.artifact_id"),
		resource.TestCheckResourceAttrSet(dataSourceName, "artifacts.0.artifact_repository_id"),
		resource.TestCheckResourceAttrSet(dataSourceName, "artifacts.0.created_at"),
		resource.TestCheckResourceAttrSet(dataSourceName, "artifacts.0.updated_at"),
		resource.TestCheckResourceAttr(dataSourceName, "artifacts.0.status", "locked"),
		resource.TestCheckResourceAttr(dataSourceName, "artifacts.0.version", "1"),
		resource.TestCheckResourceAttr(dataSourceName, "artifacts.0.creator.id", "creator-id"),
		resource.TestCheckResourceAttr(dataSourceName, "artifacts.0.tags.#", "1"),
		resource.TestCheckResourceAttr(dataSourceName, "artifacts.0.permissions.#", "2"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".name", "main"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".image_uri", imageURI),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".description", "main container"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".primary", "true"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".port", "8080"),
	)
}

func artifactsDataSourceAcceptanceChecks(dataSourceName, resourceName, name, imageURI string) resource.TestCheckFunc {
	return checkArtifactListedInDataSource(dataSourceName, resourceName, name, imageURI)
}

func checkArtifactListedInDataSource(dataSourceName, resourceName, expectedName, expectedImageURI string) resource.TestCheckFunc {
	const containerSuffix = ".spec.container_groups.0.containers.0"

	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		expectedArtifactID := rs.Primary.Attributes["artifact_id"]
		if expectedArtifactID == "" {
			return fmt.Errorf("artifact_id is not set on %s", resourceName)
		}

		ds, ok := s.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("data source %s not found in state", dataSourceName)
		}

		countStr, ok := ds.Primary.Attributes["artifacts.#"]
		if !ok {
			return fmt.Errorf("artifacts.# not found on %s", dataSourceName)
		}

		count, err := strconv.Atoi(countStr)
		if err != nil {
			return fmt.Errorf("invalid artifacts.# value %q: %w", countStr, err)
		}
		if count == 0 {
			return fmt.Errorf("expected created artifact %q in list, but artifacts.# is 0", expectedName)
		}

		for i := range count {
			prefix := fmt.Sprintf("artifacts.%d", i)
			attrs := ds.Primary.Attributes

			if attrs[prefix+".artifact_id"] != expectedArtifactID {
				continue
			}

			checks := map[string]string{
				prefix + ".name":                          expectedName,
				prefix + ".type":                          "service",
				prefix + containerSuffix + ".name":        "main",
				prefix + containerSuffix + ".image_uri":   expectedImageURI,
				prefix + containerSuffix + ".description": "main container",
				prefix + containerSuffix + ".primary":     "true",
				prefix + containerSuffix + ".port":        "8080",
			}

			for attr, expected := range checks {
				if attrs[attr] != expected {
					return fmt.Errorf("expected %s = %q, got %q", attr, expected, attrs[attr])
				}
			}

			for _, attr := range []string{
				prefix + ".artifact_repository_id",
				prefix + ".created_at",
				prefix + ".updated_at",
				prefix + ".status",
				prefix + ".version",
				prefix + ".creator.id",
			} {
				if attrs[attr] == "" {
					return fmt.Errorf("expected %s to be set", attr)
				}
			}

			return nil
		}

		return fmt.Errorf("artifact %q (id=%s) not found in %s list", expectedName, expectedArtifactID, dataSourceName)
	}
}
