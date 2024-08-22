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
var _ resource.Resource = &PlaygroundResource{}
var _ resource.ResourceWithImportState = &PlaygroundResource{}

func NewPlaygroundResource() resource.Resource {
	return &PlaygroundResource{}
}

// PlaygroundResource defines the resource implementation.
type PlaygroundResource struct {
	provider *Provider
}

func (r *PlaygroundResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playground"
}

func (r *PlaygroundResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Playground",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Playground.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Playground.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the Playground.",
				Required:            true,
			},
			"use_case_id": schema.StringAttribute{
				MarkdownDescription: "The id of the Playground.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *PlaygroundResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PlaygroundResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan PlaygroundResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	var useCaseID string
	if IsKnown(plan.UseCaseID) {
		useCaseID = plan.UseCaseID.ValueString()
		_, err := r.provider.service.GetUseCase(ctx, useCaseID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting Playground",
				fmt.Sprintf("Unable to get Playground, got error: %s", err),
			)
			return
		}
	}

	traceAPICall("CreatePlayground")
	name := plan.Name.ValueString()
	description := plan.Description.ValueString()
	createResp, err := r.provider.service.CreatePlayground(ctx, &client.CreatePlaygroundRequest{
		Name:        name,
		Description: description,
		UseCaseID:   useCaseID,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Playground",
			fmt.Sprintf("Unable to create Playground, got error: %s", err),
		)
		return
	}

	var state PlaygroundResourceModel
	loadPlaygroundToTerraformState(createResp.ID, name, description, useCaseID, &state)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

func (r *PlaygroundResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state PlaygroundResourceModel
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

	traceAPICall("GetPlayground")
	playground, err := r.provider.service.GetPlayground(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Playground not found",
				fmt.Sprintf("Playground with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting Playground info",
				fmt.Sprintf("Unable to get Playground, got error: %s", err),
			)
		}
		return
	}

	loadPlaygroundToTerraformState(playground.ID, playground.Name, playground.Description, playground.UseCaseID, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *PlaygroundResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan PlaygroundResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state PlaygroundResourceModel

	// Read Terraform state data into the model
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// It he only fields that can be updated don't change, just return.
	newName := plan.Name.ValueString()
	newDescription := plan.Description.ValueString()
	if state.Name.ValueString() == newName &&
		state.Description.ValueString() == newDescription {
		return
	}

	id := state.ID.ValueString()

	traceAPICall("UpdatePlayground")
	playground, err := r.provider.service.UpdatePlayground(ctx,
		id,
		&client.UpdatePlaygroundRequest{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Playground not found",
				fmt.Sprintf("Playground with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error updating Playground",
				fmt.Sprintf("Unable to update Playground, got error: %s", err),
			)
		}
		return
	}

	loadPlaygroundToTerraformState(id, playground.Name, playground.Description, playground.UseCaseID, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *PlaygroundResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state PlaygroundResourceModel
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

	traceAPICall("DeletePlayground")
	err := r.provider.service.DeletePlayground(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			// Playground is already gone, ignore the error and remove from state
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting Playground info",
				fmt.Sprintf("Unable to get  example, got error: %s", err),
			)
		}
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r *PlaygroundResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func loadPlaygroundToTerraformState(id, name, description, useCaseId string, state *PlaygroundResourceModel) {
	state.ID = types.StringValue(id)
	state.Name = types.StringValue(name)
	state.Description = types.StringValue(description)
	state.UseCaseID = types.StringValue(useCaseId)
}
