package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/data.

import (
	auth "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/auth"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewApiTokenCredentialResource returns the new implementation (auth package).
func NewApiTokenCredentialResource() resource.Resource { return auth.NewApiTokenCredentialResource() }

// NOTE: No legacy types or logic retained here.
