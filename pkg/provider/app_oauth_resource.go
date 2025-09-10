package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/data.

import (
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/application"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewAppOAuthResource returns the new implementation (application package).
func NewAppOAuthResource() resource.Resource { return application.NewAppOAuthResource() }

// NOTE: No legacy types or logic retained here.
