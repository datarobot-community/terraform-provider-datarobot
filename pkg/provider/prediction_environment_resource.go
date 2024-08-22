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
var _ resource.Resource = &PredictionEnvironmentResource{}
var _ resource.ResourceWithImportState = &PredictionEnvironmentResource{}

func NewPredictionEnvironmentResource() resource.Resource {
	return &PredictionEnvironmentResource{}
}

// VectorDatabaseResource defines the resource implementation.
type PredictionEnvironmentResource struct {
	provider *Provider
}

func (r *PredictionEnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prediction_environment"
}

func (r *PredictionEnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "prediction environment",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Prediction Environment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Prediction Environment.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the Prediction Environment.",
				Required:            true,
			},
			"platform": schema.StringAttribute{
				MarkdownDescription: "The platform for the Prediction Environment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *PredictionEnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PredictionEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan PredictionEnvironmentResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !IsKnown(plan.Name) {
		resp.Diagnostics.AddError(
			"Invalid name",
			"Name is required to create a Prediction Environment.",
		)
		return
	}
	name := plan.Name.ValueString()
	description := plan.Description.ValueString()

	if !IsKnown(plan.Platform) {
		resp.Diagnostics.AddError(
			"Invalid platform",
			"Platform is required to create a Prediction Environment.",
		)
		return
	}
	platform := plan.Platform.ValueString()

	traceAPICall("CreatePredictionEnvironment")
	createResp, err := r.provider.service.CreatePredictionEnvironment(ctx, &client.CreatePredictionEnvironmentRequest{
		Name:        name,
		Description: description,
		Platform:    platform,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Prediction Environment",
			fmt.Sprintf("Unable to create Prediction Environment, got error: %s", err),
		)
		return
	}

	var state PredictionEnvironmentResourceModel
	loadPredictionEnvironmentToTerraformState(
		createResp.ID,
		createResp.Name,
		createResp.Platform,
		createResp.Description,
		&state,
	)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *PredictionEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state PredictionEnvironmentResourceModel
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

	traceAPICall("GetPredictionEnvironment")
	predictionEnvironment, err := r.provider.service.GetPredictionEnvironment(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Prediction Environment not found",
				fmt.Sprintf("Prediction Environment with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting Prediction Environment info",
				fmt.Sprintf("Unable to get Prediction Environment, got error: %s", err),
			)
		}
		return
	}

	loadPredictionEnvironmentToTerraformState(
		id,
		predictionEnvironment.Name,
		predictionEnvironment.Platform,
		predictionEnvironment.Description,
		&state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *PredictionEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan PredictionEnvironmentResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state PredictionEnvironmentResourceModel

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

	traceAPICall("UpdatePredictionEnvironment")
	predictionEnvironment, err := r.provider.service.UpdatePredictionEnvironment(ctx,
		id,
		&client.UpdatePredictionEnvironmentRequest{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Prediction Environment not found",
				fmt.Sprintf("Prediction Environment with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error updating Prediction Environment",
				fmt.Sprintf("Unable to update Prediction Environment, got error: %s", err),
			)
		}
		return
	}

	loadPredictionEnvironmentToTerraformState(
		id,
		predictionEnvironment.Name,
		predictionEnvironment.Platform,
		predictionEnvironment.Description,
		&state,
	)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *PredictionEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state PredictionEnvironmentResourceModel

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

	traceAPICall("DeletePredictionEnvironment")
	err := r.provider.service.DeletePredictionEnvironment(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			// prediction environment is already gone, ignore the error and remove from state
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error deleting Prediction Environment",
				fmt.Sprintf("Unable to delete prediction environment, got error: %s", err),
			)
		}
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r *PredictionEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func loadPredictionEnvironmentToTerraformState(
	id,
	name,
	platform,
	description string,
	state *PredictionEnvironmentResourceModel,
) {
	state.ID = types.StringValue(id)
	state.Name = types.StringValue(name)
	state.Platform = types.StringValue(platform)
	state.Description = types.StringValue(description)
}
