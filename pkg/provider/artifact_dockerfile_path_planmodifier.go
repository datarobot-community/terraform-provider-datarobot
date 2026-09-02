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

	source, sourceUnknown, diags := dockerfileSourceFromConfig(ctx, req.Config, req.Path.ParentPath())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// An unresolved `source` means we don't yet know whether the eventual value
	// will be "generated" (path must end up null) or "provided" (path defaults
	// to ./Dockerfile). Guessing "provided" here would plan a concrete value
	// that a later "generated" result can't be reconciled with. Leave path
	// unknown, same as the ConfigValue guard above.
	if sourceUnknown {
		return
	}

	resp.PlanValue = plannedDockerfilePath(source, req.PlanValue)
}

func plannedDockerfilePath(source string, planValue types.String) types.String {
	if source == "generated" {
		return types.StringNull()
	}

	if !planValue.IsNull() && !planValue.IsUnknown() {
		return planValue
	}

	// planValue is only null/unknown here because the config omitted `path`
	// (an unresolved reference was already handled by the ConfigValue guard
	// above). An omitted path always resolves to the documented default,
	// even if a prior apply had a custom path in state - falling back to
	// state would make the default permanently unreachable once a custom
	// path had ever been set, since deleting the config line could never
	// undo it.
	return types.StringValue(artifactDockerfileDefaultPath)
}

func dockerfileSourceFromConfig(ctx context.Context, config tfsdk.Config, dockerfilePath path.Path) (source string, unknown bool, diags diag.Diagnostics) {
	sourcePath := dockerfilePath.AtName("source")
	var sourceVal types.String
	diags.Append(config.GetAttribute(ctx, sourcePath, &sourceVal)...)
	if diags.HasError() {
		return "", false, diags
	}
	if sourceVal.IsUnknown() {
		return "", true, diags
	}
	if sourceVal.IsNull() || sourceVal.ValueString() == "" {
		return "provided", false, diags
	}

	return sourceVal.ValueString(), false, diags
}
