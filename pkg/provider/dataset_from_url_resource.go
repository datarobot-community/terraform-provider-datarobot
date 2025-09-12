package provider

// Legacy shim delegating to domain implementation in pkg/resources/data.
// DO NOT add legacy logic here.

import (
	datares "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/data"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func NewDatasetFromURLResource() resource.Resource { return datares.NewDatasetFromURLResource() }

