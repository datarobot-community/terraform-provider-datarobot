package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/data.

import (
	application "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/application"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewCustomApplicationFromEnvironmentResource returns the new implementation (applications package).
func NewCustomApplicationFromEnvironmentResource() resource.Resource { return application.NewCustomApplicationFromEnvironmentResource() }

// NOTE: No legacy types or logic retained here.
