package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"

	genairesources "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/genai"
)

// Shim: all logic moved to pkg/resources/genai/custom_model_resource.go
// This file delegates constructor for backwards compatibility.
func NewCustomModelResource() resource.Resource { return genairesources.NewCustomModelResource() }
