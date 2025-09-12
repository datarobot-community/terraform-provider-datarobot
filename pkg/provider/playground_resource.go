package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/genai.

import (
	genaires "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/genai"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewPlaygroundResource returns the new implementation (genai package).
func NewPlaygroundResource() resource.Resource { return genaires.NewPlaygroundResource() }

// NOTE: No legacy types or logic retained here.
