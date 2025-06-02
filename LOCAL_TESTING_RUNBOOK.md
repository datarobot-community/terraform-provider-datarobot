# Local Terraform Provider Testing Runbook

This guide explains how to test your Terraform provider changes locally before submitting a PR.

## Prerequisites

- Go development environment set up
- DataRobot staging|app environment access
- Environment variables configured (`.env.staging`|`.env.prod` file)

## Method 1: Filesystem Mirror (Recommended)

This is the most reliable method for testing built provider binaries.

### Step 1: Build the Provider

```bash
# Build the provider binary
go build -o terraform-provider-datarobot

# Verify the build succeeded
ls -la terraform-provider-datarobot
```

### Step 2: Set Up Filesystem Mirror Directory

```bash
# Determine your platform (darwin_arm64 for Apple Silicon, darwin_amd64 for Intel Mac)
PLATFORM="darwin_arm64"  # or "darwin_amd64" for Intel Macs

# Create the filesystem mirror directory structure
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/datarobot-community/datarobot/999.0.0/${PLATFORM}

# Copy the provider binary to the filesystem mirror
cp terraform-provider-datarobot ~/.terraform.d/plugins/registry.terraform.io/datarobot-community/datarobot/999.0.0/${PLATFORM}/
```

### Step 3: Configure Terraform CLI

Create a `.terraformrc` file in your project root:

```hcl
provider_installation {
  filesystem_mirror {
    path = "/Users/$(whoami)/.terraform.d/plugins"
  }
  direct {
    exclude = ["registry.terraform.io/datarobot-community/datarobot"]
  }
}
```

**Note:** The `.terraformrc` file is already in `.gitignore` so it won't be committed.

### Step 4: Create Test Configuration

Create a `main.tf` file for testing:

```hcl
terraform {
  required_providers {
    datarobot = {
      source  = "datarobot-community/datarobot"
      version = "999.0.0"  # Use your local version
    }
  }
}

provider "datarobot" {
  # Configuration will be read from environment variables:
  # DATAROBOT_ENDPOINT and DATAROBOT_API_TOKEN
}

# Test your new feature here
resource "datarobot_custom_application" "test" {
  source_version_id = "<valid-version-id>"
  name              = "Test App - Local Provider"
  
  # Test your new fields
  resources = {
    replicas                           = 2
    resource_label                     = "cpu.medium"
    session_affinity                   = true
    service_web_requests_on_root_path  = false
  }
  
  allow_auto_stopping = false
}
```

### Step 5: Test the Provider

```bash
# Source your environment variables
source .env.staging

# Clean any existing Terraform state
rm -rf .terraform .terraform.lock.hcl

# Initialize Terraform with your local provider
terraform init

# Verify the provider schema includes your changes
terraform providers schema -json | jq '.provider_schemas."registry.terraform.io/datarobot-community/datarobot".resource_schemas.datarobot_custom_application.block.attributes.resources'

# Test planning (should show your new fields)
terraform plan

# !!! ****** OPTIONAL ****** !!!
# Apply if everything looks good
# !!! ****** OPTIONAL ****** !!!
terraform apply
```

## Running Acceptance Tests

```bash
# Set environment variables
export DATAROBOT_ENDPOINT="https://staging.datarobot.com/api/v2"
export DATAROBOT_API_TOKEN="<your-token-here>"

# Run specific tests
TF_ACC=1 go test -v ./pkg/provider -run TestAccCustomApplicationResource -timeout 30m
```

## Troubleshooting

### Provider Not Found
- Ensure the directory structure matches exactly: `~/.terraform.d/plugins/registry.terraform.io/datarobot-community/datarobot/999.0.0/darwin_arm64/`
- Check that the binary is executable: `chmod +x ~/.terraform.d/plugins/registry.terraform.io/datarobot-community/datarobot/999.0.0/darwin_arm64/terraform-provider-datarobot`

### Schema Issues
- Rebuild the provider: `go build -o terraform-provider-datarobot`
- Copy the new binary to the filesystem mirror
- Clean and reinitialize: `rm -rf .terraform .terraform.lock.hcl && terraform init`

### Environment Variables
```bash
# Verify environment variables are set
echo $DATAROBOT_ENDPOINT
echo $DATAROBOT_API_TOKEN

# Source them if not set
source .env.staging
```

## Clean Up

After testing, clean up your test files:

```bash
# Remove test files (these are in .gitignore)
rm -f .terraformrc main.tf .terraform.lock.hcl
rm -rf .terraform/
```

## References

- [Terraform CLI Configuration](https://developer.hashicorp.com/terraform/cli/config/config-file)
- [Local Provider Testing](https://discuss.hashicorp.com/t/how-to-locally-test-a-new-provider-using-the-new-framework/49205)
- [Provider Development](https://www.daveperrett.com/articles/2021/08/19/working-with-local-terraform-providers)
