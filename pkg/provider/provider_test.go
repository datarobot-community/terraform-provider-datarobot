package provider

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/google/uuid"
	tf_provider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/joho/godotenv"
)

var testAccProvider tf_provider.Provider
var cl *client.Client
var nameSalt string

var globalTestCfg *struct {
	UserName string
	UserID   string
	Endpoint string
	ApiKey   string
}

func init() {
	// check if .env file exist and try to load it
	// if not, log the warningn and continue load from environment variables
	// in case if tests run in the CI/CD pipeline
	if _, err := os.Stat("../../.env"); os.IsNotExist(err) {
		log.Println("Warning: .env file not found, defaulting to system environment variables")
	} else {
		err := godotenv.Load("../../.env")
		if err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		} else {
			log.Println("Loaded .env file successfully")
		}
	}
	// Initialize globalTestCfg
	globalTestCfg = &struct {
		UserName string
		UserID   string
		Endpoint string
		ApiKey   string
	}{}

	// Get environment variables
	nameSalt = uuid.New().String()
	apiKey := os.Getenv(DataRobotApiKeyEnvVar)
	cfg := client.NewConfiguration(apiKey)
	if endpoint := os.Getenv(DataRobotEndpointEnvVar); endpoint != "" {
		cfg.Endpoint = endpoint
	}
	globalTestCfg.ApiKey = os.Getenv(DataRobotApiKeyEnvVar)
	globalTestCfg.Endpoint = cfg.Endpoint
	// TODO: obtain this by requesting user data
	globalTestCfg.UserID = os.Getenv(DatarobotUserIDEnvVar)
	globalTestCfg.UserName = os.Getenv(DataRobotUserNameEnvVar)
	// END TODO

	cfg.UserAgent = UserAgent
	cl = client.NewClient(cfg)
	testAccProvider = New("test")()
}

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
//
//nolint:all
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"datarobot": providerserver.NewProtocol6WithError(New("test")()),
}

//nolint:all
func testAccPreCheck(t *testing.T) {
	if globalTestCfg.ApiKey == "" {
		t.Fatalf("%s must be set for acceptance testing", DataRobotApiKeyEnvVar)
	}
}

// testAccArtifactBuildPreCheck is required for acceptance tests that trigger artifact image builds.
// Set DATAROBOT_SKIP_ARTIFACT_BUILD_ACC=1 to skip when the environment has no Image Build Service.
func testAccArtifactBuildPreCheck(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	if os.Getenv("DATAROBOT_SKIP_ARTIFACT_BUILD_ACC") == "1" {
		t.Skip("Skipping artifact build acceptance test: DATAROBOT_SKIP_ARTIFACT_BUILD_ACC=1")
	}
}

// mockAPIKey sets a fake API key for a mock (non-acceptance) test and restores
// the original on cleanup. globalTestCfg is shared package state; leaving it as
// "fake" leaks into the parallel acceptance tests that bake globalTestCfg.ApiKey
// into their provider config (e.g. the data source tests), causing spurious 401s.
func mockAPIKey(t *testing.T) {
	t.Helper()
	orig := globalTestCfg.ApiKey
	t.Cleanup(func() { globalTestCfg.ApiKey = orig })
	globalTestCfg.ApiKey = "fake"
}

// testAccFeatureFlagPreCheck skips the test unless at least one of flagNames is
// enabled on the server under test.
func testAccFeatureFlagPreCheck(t *testing.T, flagNames ...string) {
	t.Helper()
	testAccPreCheck(t)
	svc := client.NewService(cl)

	userInfo, infoErr := svc.GetUserInfo(context.Background())
	if infoErr != nil {
		t.Logf("GetUserInfo error: %v", infoErr)
	} else {
		t.Logf("permissions: %v", userInfo.Permissions)
	}

	for _, flagName := range flagNames {
		enabled, err := svc.IsFeatureFlagEnabled(context.Background(), flagName)
		if err != nil {
			// Keep going: another flag in the list may still grant access.
			t.Logf("Feature flag check error for %q: %v", flagName, err)
			continue
		}
		t.Logf("Feature flag %q = %v", flagName, enabled)
		if enabled {
			return
		}
	}

	t.Skipf("Skipping test: none of the feature flags %v could be confirmed enabled on this server", flagNames)
}
