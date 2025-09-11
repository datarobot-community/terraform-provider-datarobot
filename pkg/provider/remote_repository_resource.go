package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/data.

import (
	administration "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/administration"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewRemoteRepositoryResource returns the new implementation (applications package).
func NewRemoteRepositoryResource() resource.Resource { return administration.NewRemoteRepositoryResource() }

// NOTE: No legacy types or logic retained here.
