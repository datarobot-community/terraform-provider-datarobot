package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/auth.

import (
	authres "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/auth"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewGoogleCloudCredentialResource returns the new implementation (auth package).
func NewGoogleCloudCredentialResource() resource.Resource { return authres.NewGoogleCloudCredentialResource() }

// NOTE: No legacy types or logic retained here.
