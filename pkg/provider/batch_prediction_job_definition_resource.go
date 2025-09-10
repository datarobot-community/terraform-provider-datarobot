package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/data.

import (
	prediction "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/prediction"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewBatchPredictionJobDefinitionResource returns the new implementation (predictions package).
func NewBatchPredictionJobDefinitionResource() resource.Resource { return prediction.NewBatchPredictionJobDefinitionResource() }

// NOTE: No legacy types or logic retained here.
