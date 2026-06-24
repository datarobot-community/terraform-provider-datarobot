package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccArtifactDataSource(t *testing.T) {
	t.Parallel()
	name := "test-artifact-ds-" + uuid.NewString()[:8]
	testArtifactDataSource(t, name, "nginx:latest", false)
}

func TestIntegrationArtifactDataSource(t *testing.T) {
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
	name := "test-artifact-ds-" + uuid.NewString()[:8]
	imageURI := "nginx:latest"
	repoIDPtr := repoID
	artifact := artifactFixture(artifactID, &repoIDPtr, name, imageURI)

	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		Return(artifact, nil)
	mockService.EXPECT().
		GetArtifact(gomock.Any(), artifactID).
		Return(artifact, nil).
		AnyTimes()
	mockService.EXPECT().
		DeleteArtifactRepository(gomock.Any(), repoID).
		Return(nil)

	testArtifactDataSource(t, name, imageURI, true)
}

func testArtifactDataSource(t *testing.T, name, imageURI string, isMock bool) {
	t.Helper()

	dataSourceName := "data.datarobot_artifact.by_id"

	steps := []resource.TestStep{
		{
			Config: artifactDataSourceConfig(name, imageURI),
			Check: artifactDataSourceChecks(dataSourceName, name, imageURI, isMock),
		},
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               isMock,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    steps,
	})
}

func artifactDataSourceConfig(name, imageURI string) string {
	return fmt.Sprintf(`
%s

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
  }
}

data "datarobot_artifact" "by_id" {
  artifact_id = datarobot_artifact.test.artifact_id
}
`, testProviderConfigBlock(), name, imageURI)
}

func testProviderConfigBlock() string {
	return fmt.Sprintf(`provider "datarobot" {
  apikey = %q
}`, globalTestCfg.ApiKey)
}

func artifactDataSourceChecks(dataSourceName, name, imageURI string, isMock bool) resource.TestCheckFunc {
	const containerPrefix = "spec.container_groups.0.containers.0"

	checks := []resource.TestCheckFunc{
		// Top-level artifact fields
		resource.TestCheckResourceAttr(dataSourceName, "name", name),
		resource.TestCheckResourceAttr(dataSourceName, "description", "test artifact description"),
		resource.TestCheckResourceAttr(dataSourceName, "type", "service"),
		resource.TestCheckResourceAttrSet(dataSourceName, "artifact_id"),
		resource.TestCheckResourceAttrSet(dataSourceName, "artifact_repository_id"),
		resource.TestCheckResourceAttrSet(dataSourceName, "created_at"),
		resource.TestCheckResourceAttrSet(dataSourceName, "updated_at"),
		resource.TestCheckResourceAttrSet(dataSourceName, "status"),

		// Container spec
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".name", "main"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".image_uri", imageURI),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".description", "main container"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".primary", "true"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".port", "8080"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".entrypoint.#", "3"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".entrypoint.0", "python"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".entrypoint.1", "-m"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".entrypoint.2", "app"),

		// Environment variables
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".environment_vars.#", "1"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".environment_vars.0.source", "string"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".environment_vars.0.name", "ENV"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".environment_vars.0.value", "production"),

		// Probes
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".startup_probe.path", "/startup"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".startup_probe.port", "8080"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".startup_probe.scheme", "HTTP"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".startup_probe.initial_delay_seconds", "10"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".startup_probe.period_seconds", "15"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".startup_probe.timeout_seconds", "5"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".startup_probe.failure_threshold", "3"),

		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".readiness_probe.path", "/health"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".readiness_probe.port", "8080"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".readiness_probe.scheme", "HTTP"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".readiness_probe.initial_delay_seconds", "5"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".readiness_probe.period_seconds", "10"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".readiness_probe.timeout_seconds", "3"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".readiness_probe.failure_threshold", "3"),

		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".liveness_probe.path", "/live"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".liveness_probe.port", "8080"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".liveness_probe.scheme", "HTTP"),
		resource.TestCheckResourceAttr(dataSourceName, containerPrefix+".liveness_probe.failure_threshold", "5"),
	}

	if isMock {
		checks = append(checks,
			resource.TestCheckResourceAttr(dataSourceName, "status", "locked"),
			resource.TestCheckResourceAttr(dataSourceName, "version", "1"),
			resource.TestCheckResourceAttr(dataSourceName, "created_at", "2026-01-01T00:00:00Z"),
			resource.TestCheckResourceAttr(dataSourceName, "updated_at", "2026-01-02T00:00:00Z"),

			resource.TestCheckResourceAttr(dataSourceName, "creator.id", "creator-id"),
			resource.TestCheckResourceAttr(dataSourceName, "creator.full_name", "Test User"),
			resource.TestCheckResourceAttr(dataSourceName, "creator.email", "test@example.com"),
			resource.TestCheckResourceAttr(dataSourceName, "creator.username", "testuser"),

			resource.TestCheckResourceAttr(dataSourceName, "tags.#", "1"),
			resource.TestCheckResourceAttr(dataSourceName, "tags.0.id", "tag-id"),
			resource.TestCheckResourceAttr(dataSourceName, "tags.0.name", "env"),
			resource.TestCheckResourceAttr(dataSourceName, "tags.0.value", "test"),

			resource.TestCheckResourceAttr(dataSourceName, "permissions.#", "2"),
			resource.TestCheckResourceAttr(dataSourceName, "permissions.0", "CAN_VIEW"),
			resource.TestCheckResourceAttr(dataSourceName, "permissions.1", "CAN_UPDATE"),
		)
	} else {
		checks = append(checks,
			resource.TestCheckResourceAttrSet(dataSourceName, "creator.id"),
			resource.TestCheckResourceAttrSet(dataSourceName, "version"),
		)
	}

	return resource.ComposeAggregateTestCheckFunc(checks...)
}

func TestIntegrationArtifactDataSourceMissingID(t *testing.T) {
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

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testProviderConfigBlock() + `
data "datarobot_artifact" "missing_id" {}`,
				ExpectError: regexp.MustCompile(`artifact_id`),
			},
		},
	})
}

func TestIntegrationArtifactDataSourceNotFound(t *testing.T) {
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
		GetArtifact(gomock.Any(), "000000000000000000000000").
		Return(nil, client.NewNotFoundError("artifact")).
		AnyTimes()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testProviderConfigBlock() + `
data "datarobot_artifact" "not_found" {
  artifact_id = "000000000000000000000000"
}`,
				ExpectError: regexp.MustCompile(`(?i)not found`),
			},
		},
	})
}
