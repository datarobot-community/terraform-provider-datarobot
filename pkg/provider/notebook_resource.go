package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/notebook.

import (
	notebookres "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/notebook"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewNotebookResource returns the new implementation (notebook package).
func NewNotebookResource() resource.Resource { return notebookres.NewNotebookResource() }

// NOTE: No legacy types or logic retained here.
