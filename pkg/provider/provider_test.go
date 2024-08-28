package provider

import (
	"os"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	tf_provider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProvider tf_provider.Provider
var cl *client.Client

func init() {
	apiKey := os.Getenv(DataRobotApiKeyEnvVar)
	cfg := client.NewConfiguration(apiKey)
	if endpoint := os.Getenv(DataRobotEndpointEnvVar); endpoint != "" {
		cfg.Endpoint = endpoint
	}
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
	if os.Getenv(DataRobotApiKeyEnvVar) == "" {
		t.Fatalf("%s must be set for acceptance testing", DataRobotApiKeyEnvVar)
	}
}
