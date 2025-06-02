package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.ResourceWithImportState = &ApplicationSourceFromTemplateResource{}
var _ resource.ResourceWithModifyPlan = &ApplicationSourceFromTemplateResource{}

func NewApplicationSourceFromTemplateResource() resource.Resource {
	return &ApplicationSourceFromTemplateResource{}
}

type ApplicationSourceFromTemplateResource struct {
	provider *Provider
}

func (r *ApplicationSourceFromTemplateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_source_from_template"
}

func (r *ApplicationSourceFromTemplateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"template_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the template used to create the Application Source.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
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
				Optional:            true,
				MarkdownDescription: "The resources for the Application Source.",
				Attributes: map[string]schema.Attribute{
					"replicas": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "The replicas for the Application Source.",
					},
					"resource_label": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The resource label for the Application Source.",
					},
					"session_affinity": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "The session affinity for the Application Source.",
					},
					"service_web_requests_on_root_path": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "Whether to service web requests on the root path for the Application Source.",
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

func (r *ApplicationSourceFromTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ApplicationSourceFromTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ApplicationSourceFromTemplateResourceModel

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateApplicationSourceFromTemplate")
	createApplicationSourceFromTemplateResp, err := r.provider.service.CreateApplicationSourceFromTemplate(ctx, &client.CreateApplicationSourceFromTemplateRequest{
		CustomTemplateID: data.TemplateID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Application source", err.Error())
		return
	}
	data.ID = types.StringValue(createApplicationSourceFromTemplateResp.ID)

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
		data.Name = types.StringValue(createApplicationSourceFromTemplateResp.Name)
	}

	updateApplicationSourceVersionRequest := &client.UpdateApplicationSourceVersionRequest{}
	if data.Resources != nil {
		updateApplicationSourceVersionRequest.Resources = &client.ApplicationResources{
			Replicas:                     Int64ValuePointerOptional(data.Resources.Replicas),
			SessionAffinity:              BoolValuePointerOptional(data.Resources.SessionAffinity),
			ResourceLabel:                StringValuePointerOptional(data.Resources.ResourceLabel),
			ServiceWebRequestsOnRootPath: BoolValuePointerOptional(data.Resources.ServiceWebRequestsOnRootPath),
		}
	}

	if IsKnown(data.BaseEnvironmentVersionID) {
		updateApplicationSourceVersionRequest.BaseEnvironmentVersionID = data.BaseEnvironmentVersionID.ValueString()
	} else {
		updateApplicationSourceVersionRequest.BaseEnvironmentID = data.BaseEnvironmentID.ValueString()
	}

	traceAPICall("UpdateApplicationSourceVersion")
	createApplicationSourceFromTemplateVersionResp, err := r.provider.service.UpdateApplicationSourceVersion(
		ctx,
		createApplicationSourceFromTemplateResp.ID,
		createApplicationSourceFromTemplateResp.LatestVersion.ID,
		updateApplicationSourceVersionRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Application source version", err.Error())
		return
	}
	data.VersionID = types.StringValue(createApplicationSourceFromTemplateVersionResp.ID)
	data.BaseEnvironmentID = types.StringValue(createApplicationSourceFromTemplateVersionResp.BaseEnvironmentID)
	data.BaseEnvironmentVersionID = types.StringValue(createApplicationSourceFromTemplateVersionResp.BaseEnvironmentVersionID)

	err = r.addLocalFilesToApplicationSource(
		ctx,
		createApplicationSourceFromTemplateResp.ID,
		createApplicationSourceFromTemplateVersionResp.ID,
		data.TemplateID.ValueString(),
		data.FolderPath,
		data.Files,
		make([]string, 0))
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
			createApplicationSourceFromTemplateResp.ID,
			createApplicationSourceFromTemplateVersionResp.ID,
			&client.UpdateApplicationSourceVersionRequest{
				RuntimeParameterValues: string(jsonParams),
			})
		if err != nil {
			resp.Diagnostics.AddError("Error adding runtime parameter values to Application Source version", err.Error())
			return
		}
	}

	applicationSource, err := r.provider.service.GetApplicationSource(ctx, createApplicationSourceFromTemplateResp.ID)
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

func (r *ApplicationSourceFromTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ApplicationSourceFromTemplateResourceModel

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

	// Populate resources from API response
	if applicationSource.LatestVersion.Resources.Replicas != nil ||
		applicationSource.LatestVersion.Resources.ResourceLabel != nil ||
		applicationSource.LatestVersion.Resources.SessionAffinity != nil ||
		applicationSource.LatestVersion.Resources.ServiceWebRequestsOnRootPath != nil {
		data.Resources = &ApplicationSourceResources{
			Replicas:                     Int64PointerValue(applicationSource.LatestVersion.Resources.Replicas),
			SessionAffinity:              BoolPointerValue(applicationSource.LatestVersion.Resources.SessionAffinity),
			ResourceLabel:                StringPointerValue(applicationSource.LatestVersion.Resources.ResourceLabel),
			ServiceWebRequestsOnRootPath: BoolPointerValue(applicationSource.LatestVersion.Resources.ServiceWebRequestsOnRootPath),
		}
	} else {
		data.Resources = nil
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

func (r *ApplicationSourceFromTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationSourceFromTemplateResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ApplicationSourceFromTemplateResourceModel

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
	createApplicationSourceFromTemplateVersionResp, err := r.provider.service.CreateApplicationSourceVersion(ctx, plan.ID.ValueString(),
		&client.CreateApplicationSourceVersionRequest{
			BaseVersion: applicationSource.LatestVersion.ID,
			Label:       fmt.Sprintf("v%d", currentVersionNum+1),
		})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Application Source version", err.Error())
		return
	}
	applicationSourceFromTemplateVersion := *createApplicationSourceFromTemplateVersionResp

	if !reflect.DeepEqual(plan.Files, state.Files) ||
		!reflect.DeepEqual(plan.FilesHashes, state.FilesHashes) ||
		plan.FolderPath != state.FolderPath ||
		plan.FolderPathHash != state.FolderPathHash {
		err = r.updateLocalFiles(ctx, state, plan, applicationSourceFromTemplateVersion)
		if err != nil {
			resp.Diagnostics.AddError("Error updating Application Source files", err.Error())
			return
		}
	}

	updateVersionRequest := &client.UpdateApplicationSourceVersionRequest{}
	if plan.Resources != nil {
		updateVersionRequest.Resources = &client.ApplicationResources{
			Replicas:                     Int64ValuePointerOptional(plan.Resources.Replicas),
			SessionAffinity:              BoolValuePointerOptional(plan.Resources.SessionAffinity),
			ResourceLabel:                StringValuePointerOptional(plan.Resources.ResourceLabel),
			ServiceWebRequestsOnRootPath: BoolValuePointerOptional(plan.Resources.ServiceWebRequestsOnRootPath),
		}
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
		applicationSourceFromTemplateVersion.ID,
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

func (r *ApplicationSourceFromTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ApplicationSourceFromTemplateResourceModel

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

func (r *ApplicationSourceFromTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r ApplicationSourceFromTemplateResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Resource is being destroyed
		return
	}

	var plan ApplicationSourceFromTemplateResourceModel

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

	var state ApplicationSourceFromTemplateResourceModel

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

func (r *ApplicationSourceFromTemplateResource) addLocalFilesToApplicationSource(
	ctx context.Context,
	id string,
	versionId string,
	templateID string,
	folderPath types.String,
	files types.Dynamic,
	templateFilesToReset []string,
) (
	err error,
) {
	localFiles, err := prepareLocalFiles(folderPath, files)
	if err != nil {
		return
	}

	for _, templateFileID := range templateFilesToReset {
		templateFile, err := r.provider.service.GetCustomTemplateFile(ctx, templateID, templateFileID)
		if err != nil {
			if strings.Contains(err.Error(), "File is not utf-8 encoded") {
				// cannot reset file content if it is not utf-8 encoded
				continue
			}
			return err
		}
		if !localFilesContain(localFiles, templateFile.Filename) {
			localFiles = append(localFiles, client.FileInfo{
				Name:    templateFile.Filename,
				Path:    templateFile.Filename,
				Content: []byte(templateFile.Content),
			})
		}
	}

	traceAPICall("UpdateApplicationSourceFromTemplateVersion")
	_, err = r.provider.service.UpdateApplicationSourceVersionFiles(ctx, id, versionId, localFiles)
	if err != nil {
		return
	}

	return
}

func (r *ApplicationSourceFromTemplateResource) updateLocalFiles(
	ctx context.Context,
	state ApplicationSourceFromTemplateResourceModel,
	plan ApplicationSourceFromTemplateResourceModel,
	applicationSourceFromTemplateVersion client.ApplicationSourceVersion,
) (
	err error,
) {
	customTemplate, err := r.provider.service.GetCustomTemplate(ctx, plan.TemplateID.ValueString())
	if err != nil {
		return
	}

	filesToDelete := make([]string, 0)
	filesToReset := make([]string, 0)
	for _, item := range applicationSourceFromTemplateVersion.Items {
		if item.FileSource == "local" {
			// template files must be reset to their default contents instead of deleted
			if fileID, ok := isTemplateFile(item.FilePath, *customTemplate); ok {
				filesToReset = append(filesToReset, fileID)
			} else {
				filesToDelete = append(filesToDelete, item.ID)
			}
		}
	}

	if len(filesToDelete) > 0 {
		traceAPICall("UpdateApplicationSourceFromTemplateVersion")
		_, err = r.provider.service.UpdateApplicationSourceVersion(
			ctx,
			state.ID.ValueString(),
			applicationSourceFromTemplateVersion.ID,
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
		applicationSourceFromTemplateVersion.ID,
		state.TemplateID.ValueString(),
		plan.FolderPath,
		plan.Files,
		filesToReset,
	)

	return
}

func isTemplateFile(path string, customTemplate client.CustomTemplate) (string, bool) {
	for _, item := range customTemplate.Items {
		if item.Name == path {
			return item.ID, true
		}
	}
	return "", false
}

func localFilesContain(files []client.FileInfo, path string) bool {
	for _, file := range files {
		if file.Path == path {
			return true
		}
	}
	return false
}
