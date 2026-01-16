package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
			"version_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The name of the Registered Model Version.",
			},
			"custom_model_version_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the custom model version for this Registered Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"use_case_ids": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "The list of Use Case IDs to add the Registered Model version to.",
				ElementType:         types.StringType,
			},
			"tags": schema.SetNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The list of tags to assign to the Registered Model version.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The name of the tag.",
						},
						"value": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The value of the tag.",
						},
					},
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

	// Normalize Tags to ensure it has the correct type
	var diags diag.Diagnostics
	data.Tags, diags = normalizeTagsSet(ctx, data.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRegisteredModelRequest := &client.CreateRegisteredModelFromCustomModelRequest{
		CustomModelVersionID: data.CustomModelVersionId.ValueString(),
		Name:                 getVersionName(data, 1),
		RegisteredModelName:  data.Name.ValueString(),
		Tags:                 convertSetTagsToClientTags(data.Tags),
	}

	if err := r.populatePromptFromCustomModel(ctx, createRegisteredModelRequest, data.CustomModelVersionId.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error populating prompt from Custom Model", err.Error())
		return
	}

	traceAPICall("CreateRegisteredModel")
	registeredModelVersion, err := r.provider.service.CreateRegisteredModelFromCustomModelVersion(ctx, createRegisteredModelRequest)
	if err != nil {
		errMessage := checkNameAlreadyExists(err, data.Name.ValueString(), "Registered Model")
		resp.Diagnostics.AddError("Error creating Registered Model", errMessage)
		return
	}
	data.ID = types.StringValue(registeredModelVersion.RegisteredModelID)
	data.VersionID = types.StringValue(registeredModelVersion.ID)
	data.VersionName = types.StringValue(registeredModelVersion.Name)

	// Set tags from API response to ensure consistency with Read method
	if len(registeredModelVersion.Tags) > 0 {
		data.Tags = initializeTagsFromModel(registeredModelVersion.Tags, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		data.Tags = types.SetNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":  types.StringType,
				"value": types.StringType,
			},
		})
	}

	if IsKnown(data.Description) {
		traceAPICall("UpdateRegisteredModel")
		_, err := r.provider.service.UpdateRegisteredModel(ctx,
			registeredModelVersion.RegisteredModelID,
			&client.UpdateRegisteredModelRequest{
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

	err = waitForRegisteredModelVersionToBeReady(ctx, r.provider.service, registeredModelVersion.RegisteredModelID, registeredModelVersion.ID)
	if err != nil {
		resp.Diagnostics.AddError("Registered model version is not ready", err.Error())
		return
	}

	for _, useCaseID := range data.UseCaseIDs {
		traceAPICall("AddRegisteredModelVersionToUseCase")
		if err = addEntityToUseCase(
			ctx,
			r.provider.service,
			useCaseID.ValueString(),
			"registeredModelVersion",
			registeredModelVersion.ID,
		); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error adding Registered Model version to Use Case %s", useCaseID), err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *RegisteredModelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RegisteredModelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Normalize Tags to ensure it has the correct type
	var diags diag.Diagnostics
	data.Tags, diags = normalizeTagsSet(ctx, data.Tags)
	resp.Diagnostics.Append(diags...)
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

	traceAPICall("GetLatestRegisteredModelVersion")
	latestRegisteredModelVersion, err := r.provider.service.GetLatestRegisteredModelVersion(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting Registered Model Version", err.Error())
		return
	}
	data.VersionID = types.StringValue(latestRegisteredModelVersion.ID)
	data.VersionName = types.StringValue(latestRegisteredModelVersion.Name)

	if len(latestRegisteredModelVersion.Tags) > 0 {
		data.Tags = initializeTagsFromModel(latestRegisteredModelVersion.Tags, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		data.Tags = types.SetNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":  types.StringType,
				"value": types.StringType,
			},
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RegisteredModelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RegisteredModelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Normalize Tags from plan to ensure it has the correct type
	var diags diag.Diagnostics
	plan.Tags, diags = normalizeTagsSet(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state RegisteredModelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Normalize Tags from state to ensure it has the correct type
	state.Tags, diags = normalizeTagsSet(ctx, state.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.VersionID = state.VersionID

	traceAPICall("UpdateRegisteredModel")
	registeredModel, err := r.provider.service.UpdateRegisteredModel(ctx,
		plan.ID.ValueString(),
		&client.UpdateRegisteredModelRequest{
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
			errMessage := checkNameAlreadyExists(err, plan.Name.ValueString(), "Registered Model")
			resp.Diagnostics.AddError("Error updating Registered Model", errMessage)
		}
		return
	}

	versionName := getVersionName(plan, registeredModel.LastVersionNum)
	plan.VersionName = types.StringValue(versionName)

	if state.VersionName.ValueString() != versionName {
		traceAPICall("UpdateRegisteredModelVersion")
		_, err := r.provider.service.UpdateRegisteredModelVersion(ctx, plan.ID.ValueString(), plan.VersionID.ValueString(), &client.UpdateRegisteredModelVersionRequest{
			Name: versionName,
		})
		if err != nil {
			resp.Diagnostics.AddError("Error updating Registered Model Version", err.Error())
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

		createRegisteredModelRequest := &client.CreateRegisteredModelFromCustomModelRequest{
			RegisteredModelID:    registeredModel.ID,
			CustomModelVersionID: plan.CustomModelVersionId.ValueString(),
			Name:                 versionName,
			Tags:                 convertSetTagsToClientTags(plan.Tags),
		}

		if err := r.populatePromptFromCustomModel(ctx, createRegisteredModelRequest, plan.CustomModelVersionId.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error populating prompt from Custom Model", err.Error())
			return
		}

		traceAPICall("CreateRegisteredModelVersion")
		registeredModelVersion, err := r.provider.service.CreateRegisteredModelFromCustomModelVersion(ctx, createRegisteredModelRequest)
		if err != nil {
			resp.Diagnostics.AddError("Error creating Registered Model Version", err.Error())
			return
		}

		err = waitForRegisteredModelVersionToBeReady(ctx, r.provider.service, registeredModelVersion.RegisteredModelID, registeredModelVersion.ID)
		if err != nil {
			resp.Diagnostics.AddError("Registered model version not ready", err.Error())
			return
		}
		plan.VersionID = types.StringValue(registeredModelVersion.ID)
	}

	// check if we created a new version
	existingUseCaseIDs := state.UseCaseIDs
	if state.VersionID.ValueString() != plan.VersionID.ValueString() {
		existingUseCaseIDs = []types.String{}
	}

	if err = updateUseCasesForEntity(
		ctx,
		r.provider.service,
		"registeredModelVersion",
		plan.VersionID.ValueString(),
		existingUseCaseIDs,
		plan.UseCaseIDs,
	); err != nil {
		resp.Diagnostics.AddError("Error updating Use Cases for Registered Model version", err.Error())
		return
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

func getVersionName(plan RegisteredModelResourceModel, versionNum int) string {
	if IsKnown(plan.VersionName) {
		return plan.VersionName.ValueString()
	}

	return fmt.Sprintf("%s (v%d)", plan.Name.ValueString(), versionNum)
}

func (r *RegisteredModelResource) findCustomModel(ctx context.Context, customModelVersionID string) (customModel client.CustomModel, err error) {
	traceAPICall("ListCustomModels")
	customModels, err := r.provider.service.ListCustomModels(ctx)
	if err != nil {
		return
	}

	for index := range customModels {
		customModel = customModels[index]
		if customModel.LatestVersion.ID == customModelVersionID {
			return
		}

		var customModelVersions []client.CustomModelVersion
		if customModelVersions, err = r.provider.service.ListCustomModelVersions(ctx, customModel.ID); err != nil {
			return
		}

		for _, customModelVersion := range customModelVersions {
			if customModelVersion.ID == customModelVersionID {
				return
			}
		}
	}

	err = fmt.Errorf("custom model with version ID %s not found", customModelVersionID)
	return
}

func (r *RegisteredModelResource) populatePromptFromCustomModel(ctx context.Context, request *client.CreateRegisteredModelFromCustomModelRequest, customModelVersionID string) error {
	customModel, err := r.findCustomModel(ctx, customModelVersionID)
	if err != nil {
		return fmt.Errorf("error finding Custom Model: %w", err)
	}

	switch customModel.TargetType {
	case "TextGeneration", "AgenticWorkflow":
		for _, runtimeParameter := range customModel.LatestVersion.RuntimeParameters {
			if runtimeParameter.FieldName == PromptRuntimeParameterName && runtimeParameter.CurrentValue != nil {
				prompt, ok := runtimeParameter.CurrentValue.(string)
				if !ok {
					return fmt.Errorf("%s value is not a string", PromptRuntimeParameterName)
				}
				request.Prompt = prompt
				return nil
			}
		}
	}
	return nil
}
