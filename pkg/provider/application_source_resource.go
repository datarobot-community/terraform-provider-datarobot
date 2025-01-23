package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	defaultAppSourceSessionAffinity = false
	defaultAppSourceReplicas        = 1
	defaultAppSourceResourceLabel   = "cpu.small"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.ResourceWithImportState = &ApplicationSourceResource{}
var _ resource.ResourceWithModifyPlan = &ApplicationSourceResource{}
var _ resource.ResourceWithConfigValidators = &ApplicationSourceResource{}

func NewApplicationSourceResource() resource.Resource {
	return &ApplicationSourceResource{}
}

type ApplicationSourceResource struct {
	provider *Provider
}

func (r *ApplicationSourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_source"
}

func (r *ApplicationSourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Application Source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Application Source.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The version ID of the Application Source.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The name of the Application Source.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"base_environment_id": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "The ID of the base environment for the Application Source.",
			},
			"base_environment_version_id": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "The ID of the base environment version for the Application Source.",
			},
			"folder_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The path to a folder containing files to build the Application Source. Each file in the folder is uploaded under path relative to a folder path.",
			},
			"folder_path_hash": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The hash of the folder path contents.",
			},
			"files": schema.DynamicAttribute{
				Optional:            true,
				MarkdownDescription: "The list of tuples, where values in each tuple are the local filesystem path and the path the file should be placed in the Application Source. If list is of strings, then basenames will be used for tuples.",
			},
			"files_hashes": schema.ListAttribute{
				Computed:            true,
				MarkdownDescription: "The hash of file contents for each file in files.",
				ElementType:         types.StringType,
			},
			"resources": schema.SingleNestedAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "The resources for the Application Source.",
				Default: objectdefault.StaticValue(types.ObjectValueMust(
					map[string]attr.Type{
						"replicas":         types.Int64Type,
						"session_affinity": types.BoolType,
						"resource_label":   types.StringType,
					},
					map[string]attr.Value{
						"replicas":         types.Int64Value(defaultAppSourceReplicas),
						"session_affinity": types.BoolValue(defaultAppSourceSessionAffinity),
						"resource_label":   types.StringValue(defaultAppSourceResourceLabel),
					},
				)),
				Attributes: map[string]schema.Attribute{
					"replicas": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(defaultAppSourceReplicas),
						MarkdownDescription: "The replicas for the Application Source.",
					},
					"resource_label": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(defaultAppSourceResourceLabel),
						MarkdownDescription: "The resource label for the Application Source.",
					},
					"session_affinity": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(defaultAppSourceSessionAffinity),
						MarkdownDescription: "The session affinity for the Application Source.",
					},
				},
			},
			"runtime_parameter_values": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The runtime parameter values for the Application Source.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The name of the runtime parameter.",
						},
						"type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The type of the runtime parameter.",
						},
						"value": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The value of the runtime parameter (type conversion is handled internally).",
						},
					},
				},
			},
		},
	}
}

func (r *ApplicationSourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ApplicationSourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ApplicationSourceResourceModel

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateApplicationSource")
	createApplicationSourceResp, err := r.provider.service.CreateApplicationSource(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Application source", err.Error())
		return
	}
	data.ID = types.StringValue(createApplicationSourceResp.ID)

	if IsKnown(data.Name) {
		traceAPICall("UpdateApplicationSource")
		_, err := r.provider.service.UpdateApplicationSource(ctx,
			data.ID.ValueString(),
			&client.UpdateApplicationSourceRequest{
				Name: data.Name.ValueString(),
			})
		if err != nil {
			resp.Diagnostics.AddError("Error updating Application source", err.Error())
			return
		}
	} else {
		data.Name = types.StringValue(createApplicationSourceResp.Name)
	}

	createApplicationSourceVersionRequest := &client.CreateApplicationSourceVersionRequest{
		Label: "v1",
		Resources: &client.ApplicationResources{
			Replicas:        data.Resources.Replicas.ValueInt64(),
			SessionAffinity: data.Resources.SessionAffinity.ValueBool(),
			ResourceLabel:   data.Resources.ResourceLabel.ValueString(),
		},
	}

	if IsKnown(data.BaseEnvironmentVersionID) {
		createApplicationSourceVersionRequest.BaseEnvironmentVersionID = data.BaseEnvironmentVersionID.ValueString()
	} else {
		createApplicationSourceVersionRequest.BaseEnvironmentID = data.BaseEnvironmentID.ValueString()
	}

	traceAPICall("CreateApplicationSourceVersion")
	createApplicationSourceVersionResp, err := r.provider.service.CreateApplicationSourceVersion(
		ctx,
		createApplicationSourceResp.ID,
		createApplicationSourceVersionRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Application source version", err.Error())
		return
	}
	data.VersionID = types.StringValue(createApplicationSourceVersionResp.ID)
	data.BaseEnvironmentID = types.StringValue(createApplicationSourceVersionResp.BaseEnvironmentID)
	data.BaseEnvironmentVersionID = types.StringValue(createApplicationSourceVersionResp.BaseEnvironmentVersionID)

	err = r.addLocalFilesToApplicationSource(
		ctx,
		createApplicationSourceResp.ID,
		createApplicationSourceVersionResp.ID,
		data.FolderPath,
		data.Files)
	if err != nil {
		resp.Diagnostics.AddError("Error adding files to Application Source", err.Error())
		return
	}

	// runtime parameter values must be set after local files are added,
	// because the runtime parameter definitions are created in the metadata.yaml file
	if IsKnown(data.RuntimeParameterValues) {
		runtimeParameterValues := make([]RuntimeParameterValue, 0)
		if diags := data.RuntimeParameterValues.ElementsAs(ctx, &runtimeParameterValues, false); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		params := make([]client.RuntimeParameterValueRequest, len(runtimeParameterValues))
		for i, param := range runtimeParameterValues {
			value, err := formatRuntimeParameterValue(param.Type.ValueString(), param.Value.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Error formatting runtime parameter value", err.Error())
				return
			}
			params[i] = client.RuntimeParameterValueRequest{
				FieldName: param.Key.ValueString(),
				Type:      param.Type.ValueString(),
				Value:     &value,
			}
		}
		jsonParams, err := json.Marshal(params)
		if err != nil {
			resp.Diagnostics.AddError("Error creating runtime parameter values", err.Error())
			return
		}

		traceAPICall("UpdateApplicationSourceVersion")
		_, err = r.provider.service.UpdateApplicationSourceVersion(ctx,
			createApplicationSourceResp.ID,
			createApplicationSourceVersionResp.ID,
			&client.UpdateApplicationSourceVersionRequest{
				RuntimeParameterValues: string(jsonParams),
			})
		if err != nil {
			resp.Diagnostics.AddError("Error adding runtime parameter values to Application Source version", err.Error())
			return
		}
	}

	applicationSource, err := r.provider.service.GetApplicationSource(ctx, createApplicationSourceResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error getting Application Source", err.Error())
		return
	}

	data.RuntimeParameterValues, diags = formatRuntimeParameterValues(
		ctx,
		applicationSource.LatestVersion.RuntimeParameters,
		data.RuntimeParameterValues)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ApplicationSourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ApplicationSourceResourceModel

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetApplicationSource")
	applicationSource, err := r.provider.service.GetApplicationSource(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Application Source not found",
				fmt.Sprintf("Application Source with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Application Source with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.Name = types.StringValue(applicationSource.Name)
	data.BaseEnvironmentID = types.StringValue(applicationSource.LatestVersion.BaseEnvironmentID)
	data.BaseEnvironmentVersionID = types.StringValue(applicationSource.LatestVersion.BaseEnvironmentVersionID)
	data.Resources = &ApplicationSourceResources{
		Replicas:        types.Int64Value(applicationSource.LatestVersion.Resources.Replicas),
		SessionAffinity: types.BoolValue(applicationSource.LatestVersion.Resources.SessionAffinity),
		ResourceLabel:   types.StringValue(applicationSource.LatestVersion.Resources.ResourceLabel),
	}

	data.RuntimeParameterValues, diags = formatRuntimeParameterValues(
		ctx,
		applicationSource.LatestVersion.RuntimeParameters,
		data.RuntimeParameterValues)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationSourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationSourceResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ApplicationSourceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Name.ValueString() != state.Name.ValueString() {
		traceAPICall("UpdateApplicationSource")
		_, err := r.provider.service.UpdateApplicationSource(ctx,
			plan.ID.ValueString(),
			&client.UpdateApplicationSourceRequest{
				Name: plan.Name.ValueString(),
			})
		if err != nil {
			if errors.Is(err, &client.NotFoundError{}) {
				resp.Diagnostics.AddWarning(
					"Application Source not found",
					fmt.Sprintf("Application Source with ID %s is not found. Removing from state.", plan.ID.ValueString()))
				resp.State.RemoveResource(ctx)
			} else {
				resp.Diagnostics.AddError("Error updating Application Source", err.Error())
			}
			return
		}
	}

	applicationSource, err := r.provider.service.GetApplicationSource(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting Application Source", err.Error())
		return
	}

	// always create a new version
	currentVersionNum := int([]rune(applicationSource.LatestVersion.Label)[1] - '0') // v1 -> 1
	traceAPICall("CreateApplicationSourceVersion")
	createApplicationSourceVersionResp, err := r.provider.service.CreateApplicationSourceVersion(ctx, plan.ID.ValueString(),
		&client.CreateApplicationSourceVersionRequest{
			BaseVersion: applicationSource.LatestVersion.ID,
			Label:       fmt.Sprintf("v%d", currentVersionNum+1),
		})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Application Source version", err.Error())
		return
	}
	applicationSourceVersion := *createApplicationSourceVersionResp

	if !reflect.DeepEqual(plan.Files, state.Files) ||
		!reflect.DeepEqual(plan.FilesHashes, state.FilesHashes) ||
		plan.FolderPath != state.FolderPath ||
		plan.FolderPathHash != state.FolderPathHash {
		err = r.updateLocalFiles(ctx, state, plan, applicationSourceVersion)
		if err != nil {
			resp.Diagnostics.AddError("Error updating Application Source files", err.Error())
			return
		}
	}

	updateVersionRequest := &client.UpdateApplicationSourceVersionRequest{
		Resources: &client.ApplicationResources{
			Replicas:        plan.Resources.Replicas.ValueInt64(),
			SessionAffinity: plan.Resources.SessionAffinity.ValueBool(),
			ResourceLabel:   plan.Resources.ResourceLabel.ValueString(),
		},
	}

	if IsKnown(plan.BaseEnvironmentVersionID) {
		updateVersionRequest.BaseEnvironmentVersionID = plan.BaseEnvironmentVersionID.ValueString()
	} else {
		updateVersionRequest.BaseEnvironmentID = plan.BaseEnvironmentID.ValueString()
	}

	runtimeParameterValues := make([]RuntimeParameterValue, 0)
	if IsKnown(plan.RuntimeParameterValues) {
		if diags := plan.RuntimeParameterValues.ElementsAs(ctx, &runtimeParameterValues, false); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	params := make([]client.RuntimeParameterValueRequest, 0)
	for _, param := range runtimeParameterValues {
		value, err := formatRuntimeParameterValue(param.Type.ValueString(), param.Value.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error formatting runtime parameter value", err.Error())
			return
		}
		params = append(params, client.RuntimeParameterValueRequest{
			FieldName: param.Key.ValueString(),
			Type:      param.Type.ValueString(),
			Value:     &value,
		})
	}

	// compute the runtime parameter values to reset
	runtimeParametersToReset := make([]RuntimeParameterValue, 0)
	if diags := state.RuntimeParameterValues.ElementsAs(ctx, &runtimeParametersToReset, false); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	for _, param := range runtimeParametersToReset {
		found := false
		for _, newParam := range runtimeParameterValues {
			if param.Key.ValueString() == newParam.Key.ValueString() {
				found = true
				break
			}
		}
		if !found {
			params = append(params, client.RuntimeParameterValueRequest{
				FieldName: param.Key.ValueString(),
				Type:      param.Type.ValueString(),
				Value:     nil,
			})
		}
	}

	jsonParams, err := json.Marshal(params)
	if err != nil {
		resp.Diagnostics.AddError("Error creating runtime parameters", err.Error())
		return
	}
	updateVersionRequest.RuntimeParameterValues = string(jsonParams)

	traceAPICall("UpdateApplicationSourceVersion")
	_, err = r.provider.service.UpdateApplicationSourceVersion(ctx,
		plan.ID.ValueString(),
		applicationSourceVersion.ID,
		updateVersionRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Application Source version", err.Error())
		return
	}

	applicationSource, err = r.provider.service.GetApplicationSource(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting Application Source", err.Error())
		return
	}
	plan.VersionID = types.StringValue(applicationSource.LatestVersion.ID)
	plan.BaseEnvironmentID = types.StringValue(applicationSource.LatestVersion.BaseEnvironmentID)
	plan.BaseEnvironmentVersionID = types.StringValue(applicationSource.LatestVersion.BaseEnvironmentVersionID)

	plan.RuntimeParameterValues, diags = formatRuntimeParameterValues(
		ctx,
		applicationSource.LatestVersion.RuntimeParameters,
		plan.RuntimeParameterValues)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ApplicationSourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ApplicationSourceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteApplicationSource")
	err := r.provider.service.DeleteApplicationSource(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Application Source", err.Error())
			return
		}
	}
}

func (r *ApplicationSourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r ApplicationSourceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Resource is being destroyed
		return
	}

	var plan ApplicationSourceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// compute file content hashes
	filesHashes, err := computeFilesHashes(ctx, plan.Files)
	if err != nil {
		resp.Diagnostics.AddError("Error calculating files hashes", err.Error())
		return
	}
	plan.FilesHashes = filesHashes

	folderPathHash, err := computeFolderHash(plan.FolderPath)
	if err != nil {
		resp.Diagnostics.AddError("Error calculating folder path hash", err.Error())
		return
	}
	plan.FolderPathHash = folderPathHash
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)

	if req.State.Raw.IsNull() {
		// resource is being created
		return
	}

	var state ApplicationSourceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !IsKnown(plan.BaseEnvironmentID) {
		if plan.BaseEnvironmentVersionID == state.BaseEnvironmentVersionID {
			// use state base environment id if base environment version id is not changed
			plan.BaseEnvironmentID = state.BaseEnvironmentID
		}
	}

	if !IsKnown(plan.BaseEnvironmentVersionID) {
		if plan.BaseEnvironmentID == state.BaseEnvironmentID {
			// use state base environment version id if base environment id is not changed
			plan.BaseEnvironmentVersionID = state.BaseEnvironmentVersionID
		}
	}

	// reset unknown version id if if hashess have been changed
	if !reflect.DeepEqual(plan.FilesHashes, state.FilesHashes) ||
		plan.FolderPathHash != state.FolderPathHash {
		plan.VersionID = types.StringUnknown()
	}

	if !IsKnown(plan.RuntimeParameterValues) {
		// use empty list if runtime parameter values are unknown
		plan.RuntimeParameterValues, _ = types.ListValueFrom(
			ctx, types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":   types.StringType,
					"type":  types.StringType,
					"value": types.StringType,
				},
			}, []RuntimeParameterValue{})
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r ApplicationSourceResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("base_environment_id"),
			path.MatchRoot("base_environment_version_id"),
		),
	}
}

func (r *ApplicationSourceResource) addLocalFilesToApplicationSource(
	ctx context.Context,
	id string,
	versionId string,
	folderPath types.String,
	files types.Dynamic,
) (
	err error,
) {
	localFiles, err := prepareLocalFiles(folderPath, files)
	if err != nil {
		return
	}

	traceAPICall("UpdateApplicationSourceVersion")
	_, err = r.provider.service.UpdateApplicationSourceVersionFiles(ctx, id, versionId, localFiles)
	if err != nil {
		return
	}

	return
}

func (r *ApplicationSourceResource) updateLocalFiles(
	ctx context.Context,
	state ApplicationSourceResourceModel,
	plan ApplicationSourceResourceModel,
	applicationSourceVersion client.ApplicationSourceVersion,
) (
	err error,
) {
	filesToDelete := make([]string, 0)
	for _, item := range applicationSourceVersion.Items {
		if item.FileSource == "local" {
			filesToDelete = append(filesToDelete, item.ID)
		}
	}

	if len(filesToDelete) > 0 {
		traceAPICall("UpdateApplicationSourceVersion")
		_, err = r.provider.service.UpdateApplicationSourceVersion(
			ctx,
			state.ID.ValueString(),
			applicationSourceVersion.ID,
			&client.UpdateApplicationSourceVersionRequest{
				FilesToDelete: filesToDelete,
			})
		if err != nil {
			return
		}
	}

	err = r.addLocalFilesToApplicationSource(
		ctx,
		state.ID.ValueString(),
		applicationSourceVersion.ID,
		plan.FolderPath,
		plan.Files,
	)

	return
}
