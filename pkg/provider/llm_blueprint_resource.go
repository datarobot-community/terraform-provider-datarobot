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
				Required:            true,
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
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan LLMBlueprintResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	var playgroundID string
	if IsKnown(plan.PlaygroundID) {
		playgroundID = plan.PlaygroundID.ValueString()
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
	if IsKnown(plan.VectorDatabaseID) {
		vectorDatabaseID = plan.VectorDatabaseID.ValueString()
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
	if IsKnown(plan.LLMID) {
		llmID = plan.LLMID.ValueString()
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

	name := plan.Name.ValueString()
	description := plan.Description.ValueString()
	traceAPICall("CreateLLMBlueprint")
	createResp, err := r.provider.service.CreateLLMBlueprint(ctx, &client.CreateLLMBlueprintRequest{
		Name:             name,
		Description:      description,
		PlaygroundID:     playgroundID,
		VectorDatabaseID: vectorDatabaseID,
		LLMID:            llmID,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating LLM Blueprint",
			fmt.Sprintf("Unable to create LLM Blueprint, got error: %s", err),
		)
		return
	}

	var state LLMBlueprintResourceModel
	loadLLMBlueprintToTerraformState(
		createResp.ID,
		name,
		description,
		playgroundID,
		vectorDatabaseID,
		llmID,
		&state,
	)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *LLMBlueprintResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state LLMBlueprintResourceModel
	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() {
		return
	}

	id := state.ID.ValueString()

	traceAPICall("GetLLMBlueprint")
	llmBlueprint, err := r.provider.service.GetLLMBlueprint(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"LLM Blueprint not found",
				fmt.Sprintf("LLM Blueprint with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting LLM Blueprint info",
				fmt.Sprintf("Unable to get LLM Blueprint, got error: %s", err),
			)
		}
		return
	}

	loadLLMBlueprintToTerraformState(
		llmBlueprint.ID,
		llmBlueprint.Name,
		llmBlueprint.Description,
		llmBlueprint.PlaygroundID,
		llmBlueprint.VectorDatabaseID,
		llmBlueprint.LLMID,
		&state,
	)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *LLMBlueprintResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan LLMBlueprintResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state LLMBlueprintResourceModel

	// Read Terraform state data into the model
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the only fields that can be updated don't change, just return.
	newName := plan.Name.ValueString()
	newDescription := plan.Description.ValueString()
	newVectorDatabaseID := plan.VectorDatabaseID.ValueString()
	newLLMID := plan.LLMID.ValueString()
	if state.Name.ValueString() == newName &&
		state.Description.ValueString() == newDescription &&
		state.VectorDatabaseID.ValueString() == newVectorDatabaseID &&
		state.LLMID.ValueString() == newLLMID {
		return
	}

	id := state.ID.ValueString()

	traceAPICall("UpdateLLMBlueprint")
	llmBlueprint, err := r.provider.service.UpdateLLMBlueprint(ctx,
		id,
		&client.UpdateLLMBlueprintRequest{
			Name:             plan.Name.ValueString(),
			Description:      plan.Description.ValueString(),
			VectorDatabaseID: plan.VectorDatabaseID.ValueString(),
			LLMID:            plan.LLMID.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"LLM Blueprint not found",
				fmt.Sprintf("LLM Blueprint with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error updating LLM Blueprint",
				fmt.Sprintf("Unable to update LLM Blueprint, got error: %s", err),
			)
		}
		return
	}

	loadLLMBlueprintToTerraformState(
		id,
		llmBlueprint.Name,
		llmBlueprint.Description,
		llmBlueprint.PlaygroundID,
		llmBlueprint.VectorDatabaseID,
		llmBlueprint.LLMID,
		&state,
	)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *LLMBlueprintResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state LLMBlueprintResourceModel
	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() {
		return
	}

	id := state.ID.ValueString()

	traceAPICall("DeleteLLMBlueprint")
	err := r.provider.service.DeleteLLMBlueprint(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			// LLM Blueprint is already gone, ignore the error and remove from state
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting LLM Blueprint info",
				fmt.Sprintf("Unable to get  example, got error: %s", err),
			)
		}
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r *LLMBlueprintResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func loadLLMBlueprintToTerraformState(
	id,
	name,
	description,
	playgroundID,
	vectorDatabaseID,
	llmID string,
	state *LLMBlueprintResourceModel,
) {
	state.ID = types.StringValue(id)
	state.Name = types.StringValue(name)
	state.Description = types.StringValue(description)
	state.PlaygroundID = types.StringValue(playgroundID)
	state.VectorDatabaseID = types.StringValue(vectorDatabaseID)
	if llmID == "" {
		state.LLMID = types.StringNull()
	} else {
		state.LLMID = types.StringValue(llmID)
	}
}
