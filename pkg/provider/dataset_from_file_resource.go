package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/data.

import (
	datares "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/data"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewDatasetFromFileResource returns the new implementation (data package).
func NewDatasetFromFileResource() resource.Resource { return datares.NewDatasetFromFileResource() }

// NOTE: No legacy types or logic retained here.
