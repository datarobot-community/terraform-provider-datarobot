package provider

import (
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/environment"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// This file delegates constructor for backwards compatibility.
func NewExecutionEnvironmentResource() resource.Resource { return environment.NewExecutionEnvironmentResource() }
