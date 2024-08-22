package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RegisteredModelResource{}
var _ resource.ResourceWithImportState = &RegisteredModelResource{}

func NewRegisteredModelResource() resource.Resource {
	return &RegisteredModelResource{}
}

// VectorDatabaseResource defines the resource implementation.
type RegisteredModelResource struct {
	provider *Provider
}

func (r *RegisteredModelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_registered_model"
}

func (r *RegisteredModelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "registered model",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Registered Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Registered Model.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the Registered Model.",
				Required:            true,
			},
			"version_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Registered Model Version.",
			},
			"custom_model_version_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the custom model version for this Registered Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RegisteredModelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RegisteredModelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan RegisteredModelResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !IsKnown(plan.CustomModelVersionId) {
		resp.Diagnostics.AddError(
			"Invalid Custom Model Version ID",
			"Custom Model Version ID is required to create a Registered Model.",
		)
		return
	}
	customModelVersionID := plan.CustomModelVersionId.ValueString()

	if !IsKnown(plan.Name) {
		resp.Diagnostics.AddError(
			"Invalid name",
			"Name is required to create a Registered Model.",
		)
		return
	}
	name := plan.Name.ValueString()

	traceAPICall("CreateRegisteredModel")
	registeredModelVersion, err := r.provider.service.CreateRegisteredModelFromCustomModelVersion(ctx, &client.CreateRegisteredModelFromCustomModelRequest{
		CustomModelVersionID: customModelVersionID,
		Name:                 fmt.Sprintf("%s (v1)", name),
		RegisteredModelName:  name,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Registered Model",
			fmt.Sprintf("Unable to create Registered Model for custom model version id: %s, got error: %s", customModelVersionID, err),
		)
		return
	}

	var state RegisteredModelResourceModel
	loadRegisteredModelToTerraformState(
		registeredModelVersion.RegisteredModelID,
		registeredModelVersion.ID,
		name,
		customModelVersionID,
		nil,
		&state,
	)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("UpdateRegisteredModel")
	registeredModel, err := r.provider.service.UpdateRegisteredModel(ctx,
		registeredModelVersion.RegisteredModelID,
		&client.RegisteredModelUpdate{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Registered Model not found",
				fmt.Sprintf("Registered Model with ID %s is not found. Removing from state.", registeredModelVersion.RegisteredModelID))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error updating Registered Model",
				fmt.Sprintf("Unable to update Registered Model, got error: %s", err),
			)
		}
		return
	}

	loadRegisteredModelToTerraformState(
		registeredModel.ID,
		registeredModelVersion.ID,
		registeredModel.Name,
		customModelVersionID,
		&registeredModel.Description,
		&state,
	)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Wait for the registered model version to be ready
	err = r.waitForRegisteredModelVersionToBeReady(ctx, registeredModelVersion.RegisteredModelID, registeredModelVersion.ID)
	if err != nil {
		resp.Diagnostics.AddError("Registered model version not ready",
			"Registered model version is not ready after 5 minutes or failed to check the status.")
		return
	}
}

func (r *RegisteredModelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state RegisteredModelResourceModel
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
	customModelVersionID := state.CustomModelVersionId.ValueString()

	traceAPICall("GetRegisteredModel")
	registeredModel, err := r.provider.service.GetRegisteredModel(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Registered Model not found",
				fmt.Sprintf("Registered Model with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting Registered Model info",
				fmt.Sprintf("Unable to get Registered Model, got error: %s", err),
			)
		}
		return
	}

	traceAPICall("ListRegisteredModelVersions")
	latestRegisteredModelVersion, err := r.provider.service.GetLatestRegisteredModelVersion(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting Registered Model Version",
			fmt.Sprintf("Unable to get Registered Model Version, got error: %s", err),
		)
		return
	}

	loadRegisteredModelToTerraformState(
		id,
		latestRegisteredModelVersion.ID,
		registeredModel.Name,
		customModelVersionID,
		&registeredModel.Description,
		&state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *RegisteredModelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan RegisteredModelResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state RegisteredModelResourceModel

	// Read Terraform state data into the model
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	newName := plan.Name.ValueString()
	newDescription := plan.Description.ValueString()
	if state.Name.ValueString() != newName || state.Description.ValueString() != newDescription {
		versionId := state.VersionID.ValueString()
		customModelVersionID := state.CustomModelVersionId.ValueString()

		traceAPICall("UpdateRegisteredModel")
		registeredModel, err := r.provider.service.UpdateRegisteredModel(ctx,
			id,
			&client.RegisteredModelUpdate{
				Name:        plan.Name.ValueString(),
				Description: plan.Description.ValueString(),
			})
		if err != nil {
			if errors.Is(err, &client.NotFoundError{}) {
				resp.Diagnostics.AddWarning(
					"Registered Model not found",
					fmt.Sprintf("Registered Model with ID %s is not found. Removing from state.", id))
				resp.State.RemoveResource(ctx)
			} else {
				resp.Diagnostics.AddError(
					"Error updating Registered Model",
					fmt.Sprintf("Unable to update Registered Model, got error: %s", err),
				)
			}
			return
		}

		loadRegisteredModelToTerraformState(
			id,
			versionId,
			registeredModel.Name,
			customModelVersionID,
			&registeredModel.Description,
			&state,
		)

		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	newCustomModelVersionID := plan.CustomModelVersionId.ValueString()
	if state.CustomModelVersionId.ValueString() != newCustomModelVersionID {
		traceAPICall("GetRegisteredModel")
		registeredModel, err := r.provider.service.GetRegisteredModel(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting Registered Model info",
				fmt.Sprintf("Unable to get Registered Model, got error: %s", err),
			)
			return
		}

		traceAPICall("CreateRegisteredModelVersion")
		registeredModelVersion, err := r.provider.service.CreateRegisteredModelFromCustomModelVersion(ctx, &client.CreateRegisteredModelFromCustomModelRequest{
			RegisteredModelID:    registeredModel.ID,
			CustomModelVersionID: newCustomModelVersionID,
			Name:                 fmt.Sprintf("%s (v%d)", newName, registeredModel.LastVersionNum+1),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating Registered Model Version",
				fmt.Sprintf("Unable to create Registered Model Version, got error: %s", err),
			)
			return
		}

		err = r.waitForRegisteredModelVersionToBeReady(ctx, registeredModelVersion.RegisteredModelID, registeredModelVersion.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Registered model version not ready",
				fmt.Sprintf("Error waiting for Registered model version: %s", err),
			)
			return
		}

		loadRegisteredModelToTerraformState(
			id,
			registeredModelVersion.ID,
			newName,
			newCustomModelVersionID,
			&newDescription,
			&state,
		)

		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func (r *RegisteredModelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state RegisteredModelResourceModel

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

	traceAPICall("DeleteRegisteredModel")
	err := r.provider.service.DeleteRegisteredModel(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			// registered model is already gone, ignore the error and remove from state
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting Registered Model info",
				fmt.Sprintf("Unable to get  example, got error: %s", err),
			)
		}
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r *RegisteredModelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func loadRegisteredModelToTerraformState(
	id,
	versionId,
	name,
	customModeVersionId string,
	description *string,
	state *RegisteredModelResourceModel,
) {
	state.ID = types.StringValue(id)
	state.VersionID = types.StringValue(versionId)
	state.Name = types.StringValue(name)
	if description != nil {
		state.Description = types.StringValue(*description)
	}
	state.CustomModelVersionId = types.StringValue(customModeVersionId)
}

func (r *RegisteredModelResource) waitForRegisteredModelVersionToBeReady(ctx context.Context, registeredModelId string, versionId string) error {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 30 * time.Second
	expBackoff.MaxElapsedTime = 5 * time.Minute

	operation := func() error {
		ready, err := r.provider.service.IsRegisteredModelVersionReady(ctx, registeredModelId, versionId)
		if err != nil {
			return backoff.Permanent(err)
		}
		if !ready {
			return errors.New("registered model version is not ready")
		}
		return nil
	}

	// Retry the operation using the backoff strategy
	err := backoff.Retry(operation, expBackoff)
	if err != nil {
		return err
	}
	return nil
}
