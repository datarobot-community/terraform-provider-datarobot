package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &MemorySpaceResource{}
var _ resource.ResourceWithImportState = &MemorySpaceResource{}

func NewMemorySpaceResource() resource.Resource {
	return &MemorySpaceResource{}
}

type MemorySpaceResource struct {
	provider *Provider
}

func (r *MemorySpaceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_memory_space"
}

func (r *MemorySpaceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Memory Space is a DataRobot concept that serves as a logical container for Chat Histories (Sessions) and persistent Memories. Feature should be enabled before use with `ENABLE_AGENTIC_MEMORY_API` flag.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Memory Space.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A human-readable description.",
				Optional:            true,
			},
			"llm_model_name": schema.StringAttribute{
				MarkdownDescription: "An LLM model name associated with the memory space (maximum 200 characters). Non-reasoning models are recommended. Reasoning-capable models are significantly slower for fact extraction without producing meaningfully better results.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
			},
			"llm_base_url": schema.StringAttribute{
				MarkdownDescription: "The chat API URL used for memory extraction. The memory service uses the DataRobot LLM gateway by default; set this only when the default does not work — for example, in air-gapped environments or when the required LLM model is not provided by the gateway and cannot be added.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 2083),
				},
			},
			"custom_instructions": schema.StringAttribute{
				MarkdownDescription: "Custom prompt instructions for fact extraction (maximum 10,000 characters). ``None`` means the default memory extraction prompt is used.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(10000),
				},
			},
		},
	}
}

func (r *MemorySpaceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MemorySpaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MemorySpaceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	enabled, err := r.provider.service.IsFeatureFlagEnabled(ctx, "ENABLE_AGENTIC_MEMORY_API")
	if err != nil {
		resp.Diagnostics.AddError("Error checking feature flag", err.Error())
		return
	}
	if !enabled {
		resp.Diagnostics.AddError(
			"Feature not enabled",
			"The ENABLE_AGENTIC_MEMORY_API feature flag is not enabled. Please enable it in your DataRobot account settings to use Memory Spaces.",
		)
		return
	}

	apiReq := &client.MemorySpaceRequest{}
	if !data.Description.IsNull() {
		desc := data.Description.ValueString()
		apiReq.Description = &desc
	}
	if !data.LLMModelName.IsNull() {
		v := data.LLMModelName.ValueString()
		apiReq.LLMModelName = &v
	}
	if !data.LLMBaseURL.IsNull() {
		v := data.LLMBaseURL.ValueString()
		apiReq.LLMBaseURL = &v
	}
	if !data.CustomInstructions.IsNull() {
		v := data.CustomInstructions.ValueString()
		apiReq.CustomInstructions = &v
	}

	traceAPICall("CreateMemorySpace")
	createResp, err := r.provider.service.CreateMemorySpace(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Memory Space", err.Error())
		return
	}
	data.ID = types.StringValue(createResp.MemorySpaceID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *MemorySpaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MemorySpaceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetMemorySpace")
	memorySpace, err := r.provider.service.GetMemorySpace(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Memory Space not found",
				fmt.Sprintf("Memory Space with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Memory Space with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	if memorySpace.Description != "" {
		data.Description = types.StringValue(memorySpace.Description)
	} else if !data.Description.IsNull() {
		data.Description = types.StringNull()
	}

	if memorySpace.LLMModelName != "" {
		data.LLMModelName = types.StringValue(memorySpace.LLMModelName)
	} else if !data.LLMModelName.IsNull() {
		data.LLMModelName = types.StringNull()
	}

	if memorySpace.LLMBaseURL != "" {
		data.LLMBaseURL = types.StringValue(memorySpace.LLMBaseURL)
	} else if !data.LLMBaseURL.IsNull() {
		data.LLMBaseURL = types.StringNull()
	}

	if memorySpace.CustomInstructions != "" {
		data.CustomInstructions = types.StringValue(memorySpace.CustomInstructions)
	} else if !data.CustomInstructions.IsNull() {
		data.CustomInstructions = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MemorySpaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MemorySpaceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	desc := data.Description.ValueString()
	llmModelName := data.LLMModelName.ValueString()
	llmBaseURL := data.LLMBaseURL.ValueString()
	customInstructions := data.CustomInstructions.ValueString()
	apiReq := &client.MemorySpaceRequest{
		Description:        &desc,
		LLMModelName:       &llmModelName,
		LLMBaseURL:         &llmBaseURL,
		CustomInstructions: &customInstructions,
	}

	traceAPICall("UpdateMemorySpace")
	_, err := r.provider.service.UpdateMemorySpace(ctx, data.ID.ValueString(), apiReq)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Memory Space not found",
				fmt.Sprintf("Memory Space with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error updating Memory Space",
				fmt.Sprintf("Unable to update Memory Space, got error: %s", err),
			)
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *MemorySpaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MemorySpaceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteMemorySpace")
	err := r.provider.service.DeleteMemorySpace(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Memory Space", err.Error())
			return
		}
	}
}

func (r *MemorySpaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
