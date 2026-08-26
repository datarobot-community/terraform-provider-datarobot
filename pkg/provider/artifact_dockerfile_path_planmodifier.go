package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const artifactDockerfileDefaultPath = "./Dockerfile"

type artifactDockerfilePathPlanModifier struct{}

var _ planmodifier.String = artifactDockerfilePathPlanModifier{}

func (m artifactDockerfilePathPlanModifier) Description(_ context.Context) string {
	return "Defaults dockerfile path to ./Dockerfile when source is provided; leaves path null when source is generated."
}

func (m artifactDockerfilePathPlanModifier) MarkdownDescription(_ context.Context) string {
	return m.Description(context.Background())
}

func (m artifactDockerfilePathPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// A config value that is still unknown (e.g. a reference to another resource's
	// not-yet-applied attribute) must stay unknown here too; only Terraform can
	// resolve it at apply time. Overwriting it now with the default or the prior
	// state value would "lock in" a value before the real one is known, and the
	// UseStateForUnknown modifier that runs after this one can no longer help
	// because it only acts while the plan value is still unknown.
	if req.ConfigValue.IsUnknown() {
		return
	}

	source, diags := dockerfileSourceFromConfig(ctx, req.Config, req.Path.ParentPath())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.PlanValue = plannedDockerfilePath(source, req.PlanValue, req.StateValue)
}

func plannedDockerfilePath(source string, planValue, stateValue types.String) types.String {
	if source == "generated" {
		return types.StringNull()
	}

	if !planValue.IsNull() && !planValue.IsUnknown() {
		return planValue
	}

	if !stateValue.IsNull() && !stateValue.IsUnknown() {
		return stateValue
	}

	return types.StringValue(artifactDockerfileDefaultPath)
}

func dockerfileSourceFromConfig(ctx context.Context, config tfsdk.Config, dockerfilePath path.Path) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	sourcePath := dockerfilePath.AtName("source")
	var source types.String
	diags.Append(config.GetAttribute(ctx, sourcePath, &source)...)
	if diags.HasError() {
		return "", diags
	}
	if source.IsNull() || source.IsUnknown() || source.ValueString() == "" {
		return "provided", diags
	}

	return source.ValueString(), diags
}
