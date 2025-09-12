package provider

import (
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/data"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// This file delegates constructor for backwards compatibility.
func NewDatasetFromDatasourceResource() resource.Resource { return data.NewDatasetFromDatasourceResource() }
