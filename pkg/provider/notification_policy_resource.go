package provider

import (
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/govern"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// This file delegates constructor for backwards compatibility.
func NewNotificationPolicyResource() resource.Resource { return govern.NewNotificationPolicyResource() }
