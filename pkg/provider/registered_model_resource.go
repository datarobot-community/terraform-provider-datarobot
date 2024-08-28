package provider

import (
	"context"
	"errors"
	"fmt"

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
				Optional:            true,
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
	var data RegisteredModelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateRegisteredModel")
	registeredModelVersion, err := r.provider.service.CreateRegisteredModelFromCustomModelVersion(ctx, &client.CreateRegisteredModelFromCustomModelRequest{
		CustomModelVersionID: data.CustomModelVersionId.ValueString(),
		Name:                 fmt.Sprintf("%s (v1)", data.Name.ValueString()),
		RegisteredModelName:  data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Registered Model", err.Error())
		return
	}
	data.ID = types.StringValue(registeredModelVersion.RegisteredModelID)
	data.VersionID = types.StringValue(registeredModelVersion.ID)

	if IsKnown(data.Description) {
		traceAPICall("UpdateRegisteredModel")
		_, err := r.provider.service.UpdateRegisteredModel(ctx,
			registeredModelVersion.RegisteredModelID,
			&client.RegisteredModelUpdate{
				Name:        data.Name.ValueString(),
				Description: data.Description.ValueString(),
			})
		if err != nil {
			if errors.Is(err, &client.NotFoundError{}) {
				resp.Diagnostics.AddWarning(
					"Registered Model not found",
					fmt.Sprintf("Registered Model with ID %s is not found. Removing from state.", registeredModelVersion.RegisteredModelID))
				resp.State.RemoveResource(ctx)
			} else {
				resp.Diagnostics.AddError("Error updating Registered Model", err.Error())
			}
			return
		}
	}

	err = r.waitForRegisteredModelVersionToBeReady(ctx, registeredModelVersion.RegisteredModelID, registeredModelVersion.ID)
	if err != nil {
		resp.Diagnostics.AddError("Registered model version is not ready", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *RegisteredModelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RegisteredModelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetRegisteredModel")
	registeredModel, err := r.provider.service.GetRegisteredModel(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Registered Model not found",
				fmt.Sprintf("Registered Model with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Registered Model with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.Name = types.StringValue(registeredModel.Name)
	if registeredModel.Description != "" {
		data.Description = types.StringValue(registeredModel.Description)
	}

	traceAPICall("ListRegisteredModelVersions")
	latestRegisteredModelVersion, err := r.provider.service.GetLatestRegisteredModelVersion(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting Registered Model Version", err.Error())
		return
	}
	data.VersionID = types.StringValue(latestRegisteredModelVersion.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RegisteredModelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RegisteredModelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state RegisteredModelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.VersionID = state.VersionID

	if state.Name.ValueString() != plan.Name.ValueString() ||
		state.Description.ValueString() != plan.Description.ValueString() {
		traceAPICall("UpdateRegisteredModel")
		_, err := r.provider.service.UpdateRegisteredModel(ctx,
			plan.ID.ValueString(),
			&client.RegisteredModelUpdate{
				Name:        plan.Name.ValueString(),
				Description: plan.Description.ValueString(),
			})
		if err != nil {
			if errors.Is(err, &client.NotFoundError{}) {
				resp.Diagnostics.AddWarning(
					"Registered Model not found",
					fmt.Sprintf("Registered Model with ID %s is not found. Removing from state.", plan.ID.ValueString()))
				resp.State.RemoveResource(ctx)
			} else {
				resp.Diagnostics.AddError("Error updating Registered Model", err.Error())
			}
			return
		}
	}

	if state.CustomModelVersionId.ValueString() != plan.CustomModelVersionId.ValueString() {
		traceAPICall("GetRegisteredModel")
		registeredModel, err := r.provider.service.GetRegisteredModel(ctx, plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error getting Registered Model info", err.Error())
			return
		}

		traceAPICall("CreateRegisteredModelVersion")
		registeredModelVersion, err := r.provider.service.CreateRegisteredModelFromCustomModelVersion(ctx, &client.CreateRegisteredModelFromCustomModelRequest{
			RegisteredModelID:    registeredModel.ID,
			CustomModelVersionID: plan.CustomModelVersionId.ValueString(),
			Name:                 fmt.Sprintf("%s (v%d)", plan.Name.ValueString(), registeredModel.LastVersionNum+1),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating Registered Model Version", err.Error())
			return
		}

		err = r.waitForRegisteredModelVersionToBeReady(ctx, registeredModelVersion.RegisteredModelID, registeredModelVersion.ID)
		if err != nil {
			resp.Diagnostics.AddError("Registered model version not ready", err.Error())
			return
		}
		plan.VersionID = types.StringValue(registeredModelVersion.ID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *RegisteredModelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RegisteredModelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteRegisteredModel")
	err := r.provider.service.DeleteRegisteredModel(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Registered Model", err.Error())
			return
		}
	}
}

func (r *RegisteredModelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *RegisteredModelResource) waitForRegisteredModelVersionToBeReady(ctx context.Context, registeredModelId string, versionId string) error {
	expBackoff := getExponentialBackoff()

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
