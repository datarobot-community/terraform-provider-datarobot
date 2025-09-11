package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/data.

import (
	collaboration "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/collaboration"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewUseCaseResource returns the new implementation (applications package).
func NewUseCaseResource() resource.Resource { return collaboration.NewUseCaseResource() }

// NOTE: No legacy types or logic retained here.
