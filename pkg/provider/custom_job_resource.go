package provider

import (
	genairesources "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/genai"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Shim: all logic moved to pkg/resources/genai/custom_job_resource.go
// This file delegates constructor for backwards compatibility.
func NewCustomJobResource() resource.Resource { return genairesources.NewCustomJobResource() }
