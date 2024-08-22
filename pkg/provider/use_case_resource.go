package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/omnistrate/terraform-provider-datarobot/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &UseCaseResource{}
var _ resource.ResourceWithImportState = &UseCaseResource{}

func NewUseCaseResource() resource.Resource {
	return &UseCaseResource{}
}

// UseCaseResource defines the resource implementation.
type UseCaseResource struct {
	provider *Provider
}

func (r *UseCaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_use_case"
}

func (r *UseCaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Use case",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Use Case.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Use Case.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the Use Case.",
				Required:            true,
			},
		},
	}
}

func (r *UseCaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UseCaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan UseCaseResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateUseCase")
	name := plan.Name.ValueString()
	description := plan.Description.ValueString()
	createResp, err := r.provider.service.CreateUseCase(ctx, &client.UseCaseRequest{
		Name:        name,
		Description: description,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Use Case",
			fmt.Sprintf("Unable to create Use Case, got error: %s", err),
		)
		return
	}

	var state UseCaseResourceModel
	loadUseCaseToTerraformState(createResp.ID, name, description, &state)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

func (r *UseCaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state UseCaseResourceModel
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

	traceAPICall("GetUseCase")
	useCase, err := r.provider.service.GetUseCase(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Use Case not found",
				fmt.Sprintf("Use Case with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting Use Case info",
				fmt.Sprintf("Unable to get Use Case, got error: %s", err),
			)
		}
		return
	}

	loadUseCaseToTerraformState(useCase.ID, useCase.Name, useCase.Description, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *UseCaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan UseCaseResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state UseCaseResourceModel

	// Read Terraform state data into the model
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the only fields that can be updated don't change, just return.
	newName := plan.Name.ValueString()
	newDescription := plan.Description.ValueString()
	if state.Name.ValueString() == newName &&
		state.Description.ValueString() == newDescription {
		return
	}

	id := state.ID.ValueString()

	traceAPICall("UpdateUseCase")
	useCase, err := r.provider.service.UpdateUseCase(ctx,
		id,
		&client.UseCaseRequest{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Use Case not found",
				fmt.Sprintf("Use Case with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error updating Use Case",
				fmt.Sprintf("Unable to update Use Case, got error: %s", err),
			)
		}
		return
	}

	loadUseCaseToTerraformState(id, useCase.Name, useCase.Description, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *UseCaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state UseCaseResourceModel
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

	traceAPICall("DeleteUseCase")
	err := r.provider.service.DeleteUseCase(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			// use case is already gone, ignore the error and remove from state
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting Use Case info",
				fmt.Sprintf("Unable to get  example, got error: %s", err),
			)
		}
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r *UseCaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func loadUseCaseToTerraformState(id, name, description string, state *UseCaseResourceModel) {
	state.ID = types.StringValue(id)
	state.Name = types.StringValue(name)
	state.Description = types.StringValue(description)
}
