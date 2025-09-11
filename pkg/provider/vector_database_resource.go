package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/data.

import (
	genai "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/genai"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewVectorDatabaseResource returns the new implementation (genai package).
func NewVectorDatabaseResource() resource.Resource { return genai.NewVectorDatabaseResource() }

// NOTE: No legacy types or logic retained here.
