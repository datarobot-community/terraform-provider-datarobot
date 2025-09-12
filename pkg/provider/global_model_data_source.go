package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/data.

import (
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/govern"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

// NewGlobalModelDataSource returns the new implementation (govern package).
func NewGlobalModelDataSource() datasource.DataSource { return govern.NewGlobalModelDataSource() }

// NOTE: No legacy types or logic retained here.
