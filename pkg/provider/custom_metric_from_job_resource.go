package provider

// Legacy shim delegating to domain implementation in pkg/resources/deployment.
// DO NOT add legacy logic here.

import (
	deployres "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/deployment"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func NewCustomMetricFromJobResource() resource.Resource { return deployres.NewCustomMetricFromJobResource() }

