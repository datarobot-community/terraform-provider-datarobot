package provider

// Legacy shim delegating to domain implementation in pkg/resources/genai.
// DO NOT add legacy logic here.

import (
	genaires "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/genai"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func NewCustomModelLLMValidationResource() resource.Resource { return genaires.NewCustomModelLLMValidationResource() }

