package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CustomModelFromVectorDatabaseResource{}
var _ resource.ResourceWithImportState = &CustomModelFromVectorDatabaseResource{}
var _ resource.ResourceWithConfigValidators = &CustomModelFromVectorDatabaseResource{}

func NewCustomModelFromVectorDatabaseResource() resource.Resource {
	return &CustomModelFromVectorDatabaseResource{}
}

// CustomModelFromVectorDatabaseResource packages a vector database into a deployable custom
// model (the "send to custom model workshop" operation). Unlike datarobot_custom_model, the
// model source is a vector database rather than an LLM blueprint or files, so the deployed
// model serves the built-in RAG (retrieval + generation) chat-completions API.
type CustomModelFromVectorDatabaseResource struct {
	provider *Provider
}

func (r *CustomModelFromVectorDatabaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_model_from_vector_database"
}

func (r *CustomModelFromVectorDatabaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A custom model packaged from a vector database (the vector database " +
			"\"send to custom model workshop\" operation). Changing the source vector database or any " +
			"compute setting forces a new custom model to be created.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the latest Custom Model version.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vector_database_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the source Vector Database for the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The name of the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The description of the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_bundle_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "A single identifier that represents a bundle of resources: Memory, CPU, GPU, etc. " +
					"Cannot be used together with memory_mb.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"replicas": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The number of replicas to deploy for the Custom Model.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"network_egress_policy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The network access policy of the Custom Model (e.g. PUBLIC or NONE).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: CustomModelNetworkEgressPolicyValidators(),
			},
			"memory_mb": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "The maximum memory in MB that may be allocated to the Custom Model. " +
					"Cannot be used together with resource_bundle_id.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *CustomModelFromVectorDatabaseResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("resource_bundle_id"),
			path.MatchRoot("memory_mb"),
		),
	}
}

func (r *CustomModelFromVectorDatabaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CustomModelFromVectorDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CustomModelFromVectorDatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resources := buildCustomModelVersionResources(plan)

	traceAPICall("CreateCustomModelVersionFromVectorDatabase")
	customModelVersion, statusID, err := r.provider.service.CreateCustomModelVersionFromVectorDatabase(
		ctx,
		plan.VectorDatabaseID.ValueString(),
		&client.CreateCustomModelVersionFromVectorDatabaseRequest{Resources: resources},
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Custom Model from Vector Database", err.Error())
		return
	}
	if customModelVersion == nil || customModelVersion.CustomModelID == "" {
		resp.Diagnostics.AddError(
			"Error creating Custom Model from Vector Database",
			"the workshop endpoint did not return a custom model id")
		return
	}
	customModelID := customModelVersion.CustomModelID

	if statusID != "" {
		traceAPICall("WaitForVectorDatabaseCustomModelPackaging")
		if err = waitForGenAITaskStatusToComplete(ctx, r.provider.service, statusID); err != nil {
			resp.Diagnostics.AddError("Error waiting for Custom Model packaging to complete", err.Error())
			return
		}
	}

	if IsKnown(plan.Name) || IsKnown(plan.Description) {
		traceAPICall("UpdateCustomModel")
		if _, err = r.provider.service.UpdateCustomModel(ctx, customModelID, &client.UpdateCustomModelRequest{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
		}); err != nil {
			resp.Diagnostics.AddError("Error updating Custom Model", err.Error())
			return
		}
	}

	customModel, err := r.waitForCustomModelReady(ctx, customModelID)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Custom Model to be ready", err.Error())
		return
	}

	loadCustomModelFromVectorDatabaseToState(*customModel, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *CustomModelFromVectorDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CustomModelFromVectorDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetCustomModel")
	customModel, err := r.provider.service.GetCustomModel(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Custom Model not found",
				fmt.Sprintf("Custom Model with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Custom Model with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}

	loadCustomModelFromVectorDatabaseToState(*customModel, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomModelFromVectorDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CustomModelFromVectorDatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CustomModelFromVectorDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only name/description are mutable in place; every other attribute is RequiresReplace.
	if !plan.Name.Equal(state.Name) || !plan.Description.Equal(state.Description) {
		traceAPICall("UpdateCustomModel")
		if _, err := r.provider.service.UpdateCustomModel(ctx, state.ID.ValueString(), &client.UpdateCustomModelRequest{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
		}); err != nil {
			resp.Diagnostics.AddError("Error updating Custom Model", err.Error())
			return
		}
	}

	traceAPICall("GetCustomModel")
	customModel, err := r.provider.service.GetCustomModel(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting Custom Model", err.Error())
		return
	}

	loadCustomModelFromVectorDatabaseToState(*customModel, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *CustomModelFromVectorDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CustomModelFromVectorDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteCustomModel")
	if err := r.provider.service.DeleteCustomModel(ctx, data.ID.ValueString()); err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Custom Model", err.Error())
			return
		}
	}
}

func (r *CustomModelFromVectorDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *CustomModelFromVectorDatabaseResource) waitForCustomModelReady(ctx context.Context, customModelID string) (*client.CustomModel, error) {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		traceAPICall("IsCustomModelReady")
		ready, err := r.provider.service.IsCustomModelReady(ctx, customModelID)
		if err != nil {
			return backoff.Permanent(err)
		}
		if !ready {
			return errors.New("custom model is not ready")
		}
		return nil
	}

	if err := backoff.Retry(operation, expBackoff); err != nil {
		return nil, err
	}

	traceAPICall("GetCustomModel")
	return r.provider.service.GetCustomModel(ctx, customModelID)
}

// buildCustomModelVersionResources maps the optional compute settings onto the workshop
// "resources" payload, returning nil when none are set so the API applies its defaults.
func buildCustomModelVersionResources(plan CustomModelFromVectorDatabaseResourceModel) *client.CustomModelVersionResources {
	resources := &client.CustomModelVersionResources{}
	set := false
	if IsKnown(plan.ResourceBundleID) {
		resources.ResourceBundleID = plan.ResourceBundleID.ValueStringPointer()
		set = true
	}
	if IsKnown(plan.Replicas) {
		replicas := plan.Replicas.ValueInt64()
		resources.Replicas = &replicas
		set = true
	}
	if IsKnown(plan.NetworkEgressPolicy) {
		resources.NetworkEgressPolicy = plan.NetworkEgressPolicy.ValueStringPointer()
		set = true
	}
	if IsKnown(plan.MemoryMB) {
		maxMem := plan.MemoryMB.ValueInt64() * 1024 * 1024
		resources.MaximumMemory = &maxMem
		set = true
	}
	if !set {
		return nil
	}
	return resources
}

func loadCustomModelFromVectorDatabaseToState(customModel client.CustomModel, state *CustomModelFromVectorDatabaseResourceModel) {
	state.ID = types.StringValue(customModel.ID)
	state.VersionID = types.StringValue(customModel.LatestVersion.ID)
	state.Name = types.StringValue(customModel.Name)
	if customModel.Description != "" {
		state.Description = types.StringValue(customModel.Description)
	} else {
		state.Description = types.StringNull()
	}
	// Resolve unset (unknown) Optional+Computed fields to null when the API omits them, else they
	// stay unknown after apply. A user-set (known) value is left untouched.
	if customModel.LatestVersion.Replicas != nil {
		state.Replicas = types.Int64Value(*customModel.LatestVersion.Replicas)
	} else if state.Replicas.IsUnknown() {
		state.Replicas = types.Int64Null()
	}
	if customModel.LatestVersion.NetworkEgressPolicy != nil {
		state.NetworkEgressPolicy = types.StringValue(*customModel.LatestVersion.NetworkEgressPolicy)
	} else if state.NetworkEgressPolicy.IsUnknown() {
		state.NetworkEgressPolicy = types.StringNull()
	}
	if customModel.LatestVersion.ResourceBundleID != nil {
		state.ResourceBundleID = types.StringValue(*customModel.LatestVersion.ResourceBundleID)
	} else if state.ResourceBundleID.IsUnknown() {
		state.ResourceBundleID = types.StringNull()
	}
	if customModel.LatestVersion.MaximumMemory != nil {
		state.MemoryMB = types.Int64Value(int64(*customModel.LatestVersion.MaximumMemory / (1024 * 1024)))
	} else if state.MemoryMB.IsUnknown() {
		state.MemoryMB = types.Int64Null()
	}
}
