package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/data.

import (
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/application"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewApplicationSourceFromTemplateResource returns the new implementation (application package).
func NewApplicationSourceFromTemplateResource() resource.Resource { return application.NewApplicationSourceFromTemplateResource() }

// NOTE: No legacy types or logic retained here.
