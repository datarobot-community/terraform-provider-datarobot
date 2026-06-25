package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &QuotaResource{}
var _ resource.ResourceWithImportState = &QuotaResource{}

func NewQuotaResource() resource.Resource {
	return &QuotaResource{}
}

// QuotaResource manages the usage quota on a DataRobot resource (e.g. a deployment).
// It replaces the imperative set_quota.py: the default rules cap request/token
// throughput per time window, and there is at most one quota per resource.
type QuotaResource struct {
	provider *Provider
}

func (r *QuotaResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_quota"
}

func (r *QuotaResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A usage quota governing a DataRobot resource (e.g. a Deployment). The " +
			"default rules cap throughput per time window and apply to every consumer. There is at " +
			"most one quota per resource; changing the resource it governs forces a new quota.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Quota.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("deployment"),
				MarkdownDescription: "The type of resource the quota governs. Defaults to `deployment`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the resource (e.g. the Deployment) the quota governs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default_rules": schema.SetNestedAttribute{
				Required:            true,
				MarkdownDescription: "The default rate-limit rules applied to all consumers.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"rule": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The metric the rule limits, e.g. `requests` or `token`.",
						},
						"limit": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "The maximum allowed for `rule` within `window`.",
						},
						"window": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The time window the limit applies to: `min`, `hour`, or `day`.",
							Validators: []validator.String{
								stringvalidator.OneOf("min", "hour", "day"),
							},
						},
					},
				},
			},
		},
	}
}

func (r *QuotaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	var ok bool
	if r.provider, ok = req.ProviderData.(*Provider); !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected  %T, got: %T. Please report this issue to the provider developers.", Provider{}, req.ProviderData),
		)
	}
}

func (r *QuotaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan QuotaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateQuota")
	quota, err := r.provider.service.CreateQuota(ctx, &client.CreateQuotaRequest{
		ResourceType: plan.ResourceType.ValueString(),
		ResourceID:   plan.ResourceID.ValueString(),
		DefaultRules: quotaRulesFromModel(plan.DefaultRules),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Quota", err.Error())
		return
	}

	loadQuotaToState(*quota, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *QuotaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data QuotaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ResourceID.IsNull() || data.ResourceID.IsUnknown() || data.ResourceID.ValueString() == "" {
		return
	}

	traceAPICall("GetQuotaForResource")
	quota, err := r.provider.service.GetQuotaForResource(ctx, data.ResourceType.ValueString(), data.ResourceID.ValueString())
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Quota not found",
				fmt.Sprintf("Quota for %s %s is not found. Removing from state.",
					data.ResourceType.ValueString(), data.ResourceID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Quota for %s %s", data.ResourceType.ValueString(), data.ResourceID.ValueString()),
				err.Error())
		}
		return
	}

	loadQuotaToState(*quota, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *QuotaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan QuotaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state QuotaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// resource_type / resource_id are RequiresReplace; only default_rules is mutable in place.
	traceAPICall("UpdateQuota")
	quota, err := r.provider.service.UpdateQuota(ctx, state.ID.ValueString(), &client.UpdateQuotaRequest{
		DefaultRules: quotaRulesFromModel(plan.DefaultRules),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating Quota", err.Error())
		return
	}

	loadQuotaToState(*quota, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *QuotaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data QuotaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteQuota")
	if err := r.provider.service.DeleteQuota(ctx, data.ID.ValueString()); err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Quota", err.Error())
			return
		}
	}
}

// ImportState keys on the governed resource, not the quota id, because the quota is a
// singleton per resource and Read looks it up that way. The import id is the resource
// id (resource_type defaults to "deployment"); pass "resourceType:resourceId" to import
// a quota on a non-deployment resource.
func (r *QuotaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resourceType := "deployment"
	resourceID := req.ID
	if parts := strings.SplitN(req.ID, ":", 2); len(parts) == 2 {
		resourceType, resourceID = parts[0], parts[1]
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_type"), resourceType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), resourceID)...)
}

func quotaRulesFromModel(rules []QuotaRuleModel) []client.QuotaRule {
	out := make([]client.QuotaRule, 0, len(rules))
	for _, rule := range rules {
		out = append(out, client.QuotaRule{
			Rule:   rule.Rule.ValueString(),
			Limit:  rule.Limit.ValueInt64(),
			Window: rule.Window.ValueString(),
		})
	}
	return out
}

func loadQuotaToState(quota client.Quota, state *QuotaResourceModel) {
	state.ID = types.StringValue(quota.ID)
	state.ResourceType = types.StringValue(quota.ResourceType)
	state.ResourceID = types.StringValue(quota.ResourceID)
	rules := make([]QuotaRuleModel, 0, len(quota.DefaultRules))
	for _, rule := range quota.DefaultRules {
		rules = append(rules, QuotaRuleModel{
			Rule:   types.StringValue(rule.Rule),
			Limit:  types.Int64Value(rule.Limit),
			Window: types.StringValue(rule.Window),
		})
	}
	state.DefaultRules = rules
}
