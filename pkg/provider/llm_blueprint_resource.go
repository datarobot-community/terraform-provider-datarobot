package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	defaultPromptType = "CHAT_HISTORY_AWARE"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &LLMBlueprintResource{}
var _ resource.ResourceWithImportState = &LLMBlueprintResource{}

func NewLLMBlueprintResource() resource.Resource {
	return &LLMBlueprintResource{}
}

type LLMBlueprintResource struct {
	provider *Provider
}

func (r *LLMBlueprintResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_llm_blueprint"
}

func (r *LLMBlueprintResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "LLMBlueprint",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the LLM Blueprint.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the LLM Blueprint.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the LLM Blueprint.",
				Optional:            true,
			},
			"playground_id": schema.StringAttribute{
				MarkdownDescription: "The id of the Playground for the LLM Blueprint.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vector_database_id": schema.StringAttribute{
				MarkdownDescription: "The id of the Vector Database for the LLM Blueprint.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"llm_id": schema.StringAttribute{
				MarkdownDescription: "The id of the LLM for the LLM Blueprint.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					// in order to generate an update to the custom model resource, we need to force a replace
					stringplanmodifier.RequiresReplace(),
				},
			},
			"llm_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The LLM settings for the LLM Blueprint.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"max_completion_length": schema.Int64Attribute{
						MarkdownDescription: "The maximum number of tokens allowed in the completion. The combined count of this value and prompt tokens must be below the model's maximum context size, where prompt token count is comprised of system prompt, user prompt, recent chat history, and vector database citations.",
						Optional:            true,
					},
					"temperature": schema.Float64Attribute{
						MarkdownDescription: "Controls the randomness of model output, where higher values return more diverse output and lower values return more deterministic results.",
						Optional:            true,
					},
					"top_p": schema.Float64Attribute{
						MarkdownDescription: "Threshold that controls the selection of words included in the response, based on a cumulative probability cutoff for token selection. Higher numbers return more diverse options for outputs.",
						Optional:            true,
					},
					"system_prompt": schema.StringAttribute{
						MarkdownDescription: "Guides the style of the LLM response. It is a 'universal' prompt, prepended to all individual prompts.",
						Optional:            true,
					},
				},
			},
			"prompt_type": schema.StringAttribute{
				MarkdownDescription: "The prompt type for the LLM Blueprint.",
				Optional:            true,
				Computed:            true,
				Default: stringdefault.StaticString(defaultPromptType),
				Validators: []validator.String{
					stringvalidator.OneOf(defaultPromptType, "ONE_TIME_PROMPT"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vector_database_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The Vector Database settings for the LLM Blueprint.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Object{
					objectvalidator.AlsoRequires(path.MatchRoot("vector_database_id")),
				},
				Attributes: map[string]schema.Attribute{
					"max_documents_retrieved_per_prompt": schema.Int64Attribute{
						MarkdownDescription: "The maximum number of documents to retrieve from the Vector Database.",
						Optional:            true,
						Validators: []validator.Int64{
							int64validator.Between(0, 10),
						},
					},
					"max_tokens": schema.Int64Attribute{
						MarkdownDescription: "The maximum number of tokens to retrieve from the Vector Database.",
						Optional:            true,
					},
				},
			},
		},
	}
}

func (r *LLMBlueprintResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *LLMBlueprintResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LLMBlueprintResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var llmID string
	if IsKnown(data.LLMID) {
		llmID = data.LLMID.ValueString()
		listResp, err := r.provider.service.ListLLMs(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Error getting LLM Blueprint", err.Error())
			return
		}

		// loop through resp.data and ensure llmID exists in the list
		llmExists := false
		for _, llm := range listResp.Data {
			if llm.ID == llmID {
				llmExists = true
				break
			}
		}

		if !llmExists {
			resp.Diagnostics.AddError(
				"Error getting LLM Blueprint",
				fmt.Sprintf("Unable to get LLM Blueprint, LLM ID does not exist: %s", llmID),
			)
			return
		}
	}

	createLLMBlueprintRequest := &client.CreateLLMBlueprintRequest{
		Name:             data.Name.ValueString(),
		Description:      data.Description.ValueString(),
		PlaygroundID:     data.PlaygroundID.ValueString(),
		VectorDatabaseID: data.VectorDatabaseID.ValueString(),
		LLMID:            data.LLMID.ValueString(),
		PromptType:       data.PromptType.ValueString(),
	}

	if data.LLMSettings != nil {
		createLLMBlueprintRequest.LLMSettings = &client.LLMSettings{}
		if IsKnown(data.LLMSettings.MaxCompletionLength) {
			createLLMBlueprintRequest.LLMSettings.MaxCompletionLength = data.LLMSettings.MaxCompletionLength.ValueInt64()
		}
		if IsKnown(data.LLMSettings.Temperature) {
			createLLMBlueprintRequest.LLMSettings.Temperature = data.LLMSettings.Temperature.ValueFloat64()
		}
		if IsKnown(data.LLMSettings.TopP) {
			createLLMBlueprintRequest.LLMSettings.TopP = data.LLMSettings.TopP.ValueFloat64()
		}
		if IsKnown(data.LLMSettings.SystemPrompt) {
			createLLMBlueprintRequest.LLMSettings.SystemPrompt = data.LLMSettings.SystemPrompt.ValueString()
		}
	}

	if data.VectorDatabaseSettings != nil {
		createLLMBlueprintRequest.VectorDatabaseSettings = &client.VectorDatabaseSettings{}
		if IsKnown(data.VectorDatabaseSettings.MaxDocumentsRetrievedPerPrompt) {
			createLLMBlueprintRequest.VectorDatabaseSettings.MaxDocumentsRetrievedPerPrompt = data.VectorDatabaseSettings.MaxDocumentsRetrievedPerPrompt.ValueInt64()
		}
		if IsKnown(data.VectorDatabaseSettings.MaxTokens) {
			createLLMBlueprintRequest.VectorDatabaseSettings.MaxTokens = data.VectorDatabaseSettings.MaxTokens.ValueInt64()
		}
	}

	traceAPICall("CreateLLMBlueprint")
	createResp, err := r.provider.service.CreateLLMBlueprint(ctx, createLLMBlueprintRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating LLM Blueprint", err.Error())
		return
	}

	data.ID = types.StringValue(createResp.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *LLMBlueprintResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LLMBlueprintResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetLLMBlueprint")
	llmBlueprint, err := r.provider.service.GetLLMBlueprint(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"LLM Blueprint not found",
				fmt.Sprintf("LLM Blueprint with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting LLM Blueprint with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}

	data.Name = types.StringValue(llmBlueprint.Name)
	if llmBlueprint.Description != "" {
		data.Description = types.StringValue(llmBlueprint.Description)
	}
	data.PlaygroundID = types.StringValue(llmBlueprint.PlaygroundID)
	data.LLMID = types.StringValue(llmBlueprint.LLMID)
	if llmBlueprint.VectorDatabaseID != "" {
		data.VectorDatabaseID = types.StringValue(llmBlueprint.VectorDatabaseID)
	}
	data.PromptType = types.StringValue(llmBlueprint.PromptType)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LLMBlueprintResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data LLMBlueprintResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("UpdateLLMBlueprint")
	_, err := r.provider.service.UpdateLLMBlueprint(ctx,
		data.ID.ValueString(),
		&client.UpdateLLMBlueprintRequest{
			Name:        data.Name.ValueString(),
			Description: data.Description.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"LLM Blueprint not found",
				fmt.Sprintf("LLM Blueprint with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error updating LLM Blueprint", err.Error())
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *LLMBlueprintResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LLMBlueprintResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteLLMBlueprint")
	err := r.provider.service.DeleteLLMBlueprint(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting LLM Blueprint", err.Error())
			return
		}
	}
}

func (r *LLMBlueprintResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
