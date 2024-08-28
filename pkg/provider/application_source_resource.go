package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const baseEnvironmentName = "[Experimental] Python 3.9 Streamlit"

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ApplicationSourceResource{}
var _ resource.ResourceWithImportState = &ApplicationSourceResource{}

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
				PlanModifiers: []planmodifier.String{
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
			"local_files": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "The list of local file paths used to build the Application Source.",
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

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	baseEnvironmentID, err := r.getBaseEnvironmentID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error getting Base Environment", err.Error())
		return
	}
	if baseEnvironmentID == "" {
		resp.Diagnostics.AddError(
			"Base Environment not found",
			fmt.Sprintf("Base Environment with name %s is not found.", baseEnvironmentName),
		)
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

	traceAPICall("CreateApplicationSourceVersion")
	createApplicationSourceVersionResp, err := r.provider.service.CreateApplicationSourceVersion(ctx,
		createApplicationSourceResp.ID,
		&client.CreateApplicationSourceVersionRequest{
			Label:             "v1",
			BaseEnvironmentID: baseEnvironmentID,
		})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Application source version", err.Error())
		return
	}
	data.VersionID = types.StringValue(createApplicationSourceVersionResp.ID)

	errSummary, errDetail := r.addLocalFilesToApplicationSource(
		ctx,
		createApplicationSourceResp.ID,
		createApplicationSourceVersionResp.ID,
		data.LocalFiles)
	if errSummary != "" {
		resp.Diagnostics.AddError(errSummary, errDetail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ApplicationSourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ApplicationSourceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetApplicationSource")
	applicationSource, err := r.provider.service.GetApplicationSource(ctx, data.ID.ValueString())
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Application Source not found",
				fmt.Sprintf("Application Source with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error getting Application Source", err.Error())
		}
		return
	}
	data.Name = types.StringValue(applicationSource.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationSourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationSourceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	if !reflect.DeepEqual(plan.LocalFiles, state.LocalFiles) {
		applicationSourceVersion, errSummary, errDetail := r.updateLocalFiles(ctx, state, plan)
		if errSummary != "" {
			resp.Diagnostics.AddError(errSummary, errDetail)
			return
		}
		plan.VersionID = types.StringValue(applicationSourceVersion)
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

func (r *ApplicationSourceResource) getBaseEnvironmentID(ctx context.Context) (baseEnvironmentID string, err error) {
	traceAPICall("ListExecutionEnvironments")
	listResp, err := r.provider.service.ListExecutionEnvironments(ctx)
	if err != nil {
		return
	}

	for _, executionEnvironment := range listResp.Data {
		if executionEnvironment.Name == baseEnvironmentName {
			baseEnvironmentID = executionEnvironment.ID
			break
		}
	}

	return
}

func (r *ApplicationSourceResource) addLocalFilesToApplicationSource(
	ctx context.Context,
	id string,
	versionId string,
	files []basetypes.StringValue,
) (
	errSummary string,
	errDetail string,
) {
	if files == nil {
		return
	}

	localFiles := make([]client.FileInfo, 0)
	for _, file := range files {
		filePath := file.ValueString()
		fileReader, err := os.Open(filePath)
		if err != nil {
			errSummary = "Error opening local file"
			errDetail = err.Error()
			return
		}
		defer fileReader.Close()
		fileContent, err := io.ReadAll(fileReader)
		if err != nil {
			errSummary = "Error reading local file"
			errDetail = err.Error()
			return
		}

		localFiles = append(localFiles, client.FileInfo{
			Name:    filePath,
			Path:    filePath,
			Content: fileContent,
		})
	}

	traceAPICall("UpdateApplicationSourceVersion")
	_, err := r.provider.service.UpdateApplicationSourceVersionFiles(ctx, id, versionId, localFiles)
	if err != nil {
		errSummary = "Error creating Application Source version from local files"
		errDetail = err.Error()
		return
	}

	return
}

func (r *ApplicationSourceResource) updateLocalFiles(
	ctx context.Context,
	state ApplicationSourceResourceModel,
	plan ApplicationSourceResourceModel,
) (
	applicationSourceVersionID string,
	errSummary string,
	errDetail string,
) {
	traceAPICall("GetApplicationSourceVersion")
	applicationSourceVersion, err := r.provider.service.GetApplicationSourceVersion(
		ctx,
		state.ID.ValueString(),
		state.VersionID.ValueString(),
	)
	if err != nil {
		errSummary = "Error getting Application Source version"
		errDetail = err.Error()
		return
	}
	applicationSourceVersionID = applicationSourceVersion.ID

	if applicationSourceVersion.IsFrozen {
		currentVersionNum := int([]rune(applicationSourceVersion.Label)[1] - '0') // v1 -> 1
		traceAPICall("CreateApplicationSourceVersion")
		newApplicationSourceVersion, err := r.provider.service.CreateApplicationSourceVersion(
			ctx,
			state.ID.ValueString(),
			&client.CreateApplicationSourceVersionRequest{
				Label:             fmt.Sprintf("v%d", currentVersionNum+1),
				BaseEnvironmentID: applicationSourceVersion.BaseEnvironmentID,
			})
		if err != nil {
			errSummary = "Error creating new Application Source version"
			errDetail = err.Error()
			return
		}
		applicationSourceVersionID = newApplicationSourceVersion.ID
	}

	filesToDelete := make([]string, 0)
	for _, file := range state.LocalFiles {
		if !contains(plan.LocalFiles, file) {
			for _, item := range applicationSourceVersion.Items {
				if item.FilePath == file.ValueString() && item.FileSource == "local" {
					filesToDelete = append(filesToDelete, item.ID)
				}
			}
		}
	}

	if len(filesToDelete) > 0 {
		traceAPICall("UpdateApplicationSourceVersion")
		_, err := r.provider.service.UpdateApplicationSourceVersion(
			ctx,
			state.ID.ValueString(),
			applicationSourceVersionID,
			&client.UpdateApplicationSourceVersionRequest{
				FilesToDelete: filesToDelete,
			})
		if err != nil {
			errSummary = "Error updating Application Source version"
			errDetail = err.Error()
			return
		}
	}

	filesToAdd := make([]string, 0)
	for _, file := range plan.LocalFiles {
		if !contains(state.LocalFiles, file) {
			filesToAdd = append(filesToAdd, file.ValueString())
		}
	}

	if len(filesToAdd) > 0 {
		errSummary, errDetail = r.addLocalFilesToApplicationSource(
			ctx,
			state.ID.ValueString(),
			applicationSourceVersionID,
			plan.LocalFiles,
		)
	}

	return
}
