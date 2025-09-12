package provider

import (
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/deployment"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// This file delegates constructor for backwards compatibility.
func NewDeploymentResource() resource.Resource { return deployment.NewDeploymentResource() }
