package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/data.

import (
	auth "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/auth"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewAAzureCredentialResource returns the new implementation (auth package).
func NewAzureCredentialResource() resource.Resource { return auth.NewAzureCredentialResource() }

// NOTE: No legacy types or logic retained here.
