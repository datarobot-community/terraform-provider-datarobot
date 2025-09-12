package provider

// Code-generated legacy shim. DO NOT RESTORE original implementation.
// Maintained only temporarily to avoid import churn during refactor.
// Delegates to new implementation under pkg/resources/data.

import (
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/environment"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

// NewExecutionEnvironmentDataSource returns the new implementation (environment package).
func NewExecutionEnvironmentDataSource() datasource.DataSource { return environment.NewExecutionEnvironmentDataSource() }

// NOTE: No legacy types or logic retained here.
