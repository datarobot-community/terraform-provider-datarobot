package provider

import (
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
