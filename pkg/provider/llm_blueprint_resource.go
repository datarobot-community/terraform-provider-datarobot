package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
					stringplanmodifier.RequiresReplaceIfConfigured(),
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

	var playgroundID string
	if IsKnown(data.PlaygroundID) {
		playgroundID = data.PlaygroundID.ValueString()
		_, err := r.provider.service.GetPlayground(ctx, playgroundID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting LLM Blueprint",
				fmt.Sprintf("Unable to get LLM Blueprint, got error: %s", err),
			)
			return
		}
	}

	var vectorDatabaseID string
	if IsKnown(data.VectorDatabaseID) {
		vectorDatabaseID = data.VectorDatabaseID.ValueString()
		_, err := r.provider.service.GetVectorDatabase(ctx, vectorDatabaseID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting LLM Blueprint",
				fmt.Sprintf("Unable to get LLM Blueprint, got error: %s", err),
			)
			return
		}
	}

	var llmID string
	if IsKnown(data.LLMID) {
		llmID = data.LLMID.ValueString()
		listResp, err := r.provider.service.ListLLMs(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting LLM Blueprint",
				fmt.Sprintf("Unable to get LLM Blueprint, got error: %s", err),
			)
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

	traceAPICall("CreateLLMBlueprint")
	createResp, err := r.provider.service.CreateLLMBlueprint(ctx, &client.CreateLLMBlueprintRequest{
		Name:             data.Name.ValueString(),
		Description:      data.Description.ValueString(),
		PlaygroundID:     data.PlaygroundID.ValueString(),
		VectorDatabaseID: data.VectorDatabaseID.ValueString(),
		LLMID:            data.LLMID.ValueString(),
	})
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
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"LLM Blueprint not found",
				fmt.Sprintf("LLM Blueprint with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error getting LLM Blueprint info", err.Error())
		}
		return
	}

	data.Name = types.StringValue(llmBlueprint.Name)
	data.Description = types.StringValue(llmBlueprint.Description)
	data.PlaygroundID = types.StringValue(llmBlueprint.PlaygroundID)
	data.LLMID = types.StringValue(llmBlueprint.LLMID)
	if llmBlueprint.VectorDatabaseID != "" {
		data.VectorDatabaseID = types.StringValue(llmBlueprint.VectorDatabaseID)
	}

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
			Name:             data.Name.ValueString(),
			Description:      data.Description.ValueString(),
			VectorDatabaseID: data.VectorDatabaseID.ValueString(),
			LLMID:            data.LLMID.ValueString(),
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
