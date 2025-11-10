package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ExecutionEnvironmentResource{}
var _ resource.ResourceWithImportState = &ExecutionEnvironmentResource{}
var _ resource.ResourceWithConfigValidators = &ExecutionEnvironmentResource{}
var _ resource.ResourceWithModifyPlan = &ExecutionEnvironmentResource{}

func NewExecutionEnvironmentResource() resource.Resource {
	return &ExecutionEnvironmentResource{}
}

type ExecutionEnvironmentResource struct {
	provider *Provider
}

func (r *ExecutionEnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_execution_environment"
}

func (r *ExecutionEnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Execution Environment",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Execution Environment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Execution Environment.",
			},
			"programming_language": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The programming language of the Execution Environment.",
				Validators:          ProgrammingLanguageValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The description of the Execution Environment.",
			},
			"use_cases": schema.ListAttribute{
				Required:            true,
				MarkdownDescription: "The list of Use Cases that the Execution Environment supports.",
				ElementType:         types.StringType,
				Validators:          ExecutionEnvironmentUseCasesValidators(),
			},
			"version_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Execution Environment Version.",
			},
			"version_description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The description of the Execution Environment version.",
			},
			"docker_context_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The path to a docker context archive or folder",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"docker_image_uri": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The URI of a pre-built environment image (e.g., in a remote Docker registry).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"docker_context_hash": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The hash of the docker context contents.",
			},
			"docker_image": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A prebuilt environment image saved as a tarball using the Docker save command.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"docker_image_hash": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The hash of the docker image file",
			},
			"build_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the Execution Environment version build.",
			},
		},
	}
}

func (r *ExecutionEnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ExecutionEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ExecutionEnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var dockerContextPath string
	var fileContent []byte
	var dockerImageContents []byte
	var err error
	if IsKnown(data.DockerContextPath) {
		dockerContextPath, fileContent, err = getDockerContext(data.DockerContextPath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error getting Docker context", err.Error())
			return
		}
	}
	if IsKnown(data.DockerImage) {
		if dockerImageContents, err = os.ReadFile(data.DockerImage.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error getting Docker image", err.Error())
			return
		}
	}

	useCases := make([]string, 0, len(data.UseCases))
	for _, useCase := range data.UseCases {
		useCases = append(useCases, useCase.ValueString())
	}

	traceAPICall("CreateExecutionEnvironment")
	executionEnvironment, err := r.provider.service.CreateExecutionEnvironment(ctx, &client.CreateExecutionEnvironmentRequest{
		Name:                data.Name.ValueString(),
		Description:         data.Description.ValueString(),
		ProgrammingLanguage: data.ProgrammingLanguage.ValueString(),
		UseCases:            useCases,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Execution Environment", err.Error())
		return
	}

	createExecutionEnvironmentVersionRequest := &client.CreateExecutionEnvironmentVersionRequest{
		Description: data.VersionDescription.ValueString(),
		Files:       make([]client.FileInfo, 0),
	}

	if IsKnown(data.DockerContextPath) {
		createExecutionEnvironmentVersionRequest.Files = append(createExecutionEnvironmentVersionRequest.Files, client.FileInfo{
			Name:          filepath.Base(dockerContextPath),
			Content:       fileContent,
			FormFieldName: "docker_context",
		})
	}
	if IsKnown(data.DockerImage) {
		createExecutionEnvironmentVersionRequest.Files = append(createExecutionEnvironmentVersionRequest.Files, client.FileInfo{
			Name:          filepath.Base(data.DockerImage.ValueString()),
			Content:       dockerImageContents,
			FormFieldName: "docker_image",
		})
	}
	if IsKnown(data.DockerImageUri) {
		createExecutionEnvironmentVersionRequest.DockerImageUri = data.DockerImageUri.ValueString()
	}

	traceAPICall("CreateExecutionEnvironmentVersion")
	if _, err := r.provider.service.CreateExecutionEnvironmentVersion(ctx, executionEnvironment.ID, createExecutionEnvironmentVersionRequest); err != nil {
		resp.Diagnostics.AddError("Error creating Execution Environment Version", err.Error())
		return
	}

	executionEnvironment, err = waitForExecutionEnvironmentToBeReady(ctx, r.provider.service, executionEnvironment.ID)
	if err != nil {
		resp.Diagnostics.AddError("Execution Environment failed to build", err.Error())
		return
	}
	data.ID = types.StringValue(executionEnvironment.ID)
	data.VersionID = types.StringValue(executionEnvironment.LatestVersion.ID)
	data.BuildStatus = types.StringValue(executionEnvironment.LatestVersion.BuildStatus)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ExecutionEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ExecutionEnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetExecutionEnvironment")
	executionEnvironment, err := r.provider.service.GetExecutionEnvironment(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Execution Environment not found",
				fmt.Sprintf("Execution Environment with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Execution Environment with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}

	var executionEnvironmentVersion *client.ExecutionEnvironmentVersion
	if IsKnown(data.VersionID) && data.VersionID.ValueString() != "" {
		targetExecutionEnvironmentVersion, err := r.provider.service.GetExecutionEnvironmentVersion(ctx, executionEnvironment.ID, data.VersionID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error getting Execution Environment Version", err.Error())
			return
		} else {
			executionEnvironmentVersion = targetExecutionEnvironmentVersion
		}
	} else {
		executionEnvironmentVersion = &executionEnvironment.LatestVersion
	}

	// data.ID = types.StringValue(executionEnvironment.ID)
	data.Name = types.StringValue(executionEnvironment.Name)
	data.ProgrammingLanguage = types.StringValue(executionEnvironment.ProgrammingLanguage)
	useCases := make([]types.String, len(executionEnvironment.UseCases))
	for i, useCase := range executionEnvironment.UseCases {
		useCases[i] = types.StringValue(useCase)
	}
	data.UseCases = useCases
	if executionEnvironment.Description != "" {
		data.Description = types.StringValue(executionEnvironment.Description)
	}
	data.VersionID = types.StringValue(executionEnvironmentVersion.ID)
	if executionEnvironmentVersion.Description != "" {
		data.VersionDescription = types.StringValue(executionEnvironmentVersion.Description)
	}
	if executionEnvironmentVersion.DockerImageUri != "" {
		data.DockerImageUri = types.StringValue(executionEnvironmentVersion.DockerImageUri)
	}
	data.BuildStatus = types.StringValue(executionEnvironmentVersion.BuildStatus)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExecutionEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ExecutionEnvironmentResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ExecutionEnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var dockerContextPath string
	var fileContent []byte
	var dockerImageContent []byte
	var err error
	if IsKnown(plan.DockerContextPath) {
		dockerContextPath, fileContent, err = getDockerContext(plan.DockerContextPath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error getting Docker context", err.Error())
			return
		}
	}
	if IsKnown(plan.DockerImage) {
		if dockerImageContent, err = os.ReadFile(plan.DockerImage.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error getting Docker image", err.Error())
			return
		}
	}

	useCases := make([]string, 0, len(plan.UseCases))
	for _, useCase := range plan.UseCases {
		useCases = append(useCases, useCase.ValueString())
	}

	traceAPICall("UpdateExecutionEnvironment")
	executionEnvironment, err := r.provider.service.UpdateExecutionEnvironment(ctx,
		plan.ID.ValueString(),
		&client.UpdateExecutionEnvironmentRequest{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			UseCases:    useCases,
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Execution Environment not found",
				fmt.Sprintf("Execution Environment with ID %s is not found. Removing from state.", plan.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error updating Execution Environment", err.Error())
		}
		return
	}

	if plan.VersionDescription.ValueString() != state.VersionDescription.ValueString() ||
		plan.DockerContextPath.ValueString() != state.DockerContextPath.ValueString() ||
		plan.DockerContextHash.ValueString() != state.DockerContextHash.ValueString() ||
		plan.DockerImageHash.ValueString() != state.DockerImageHash.ValueString() ||
		plan.DockerImage.ValueString() != state.DockerImage.ValueString() ||
		plan.DockerImageUri.ValueString() != state.DockerImageUri.ValueString() {
		traceAPICall("CreateExecutionEnvironmentVersion")
		updateExecutionEnvironmentRequest := &client.CreateExecutionEnvironmentVersionRequest{
			Description: plan.VersionDescription.ValueString(),
			Files:       make([]client.FileInfo, 0),
		}

		if IsKnown(plan.DockerContextPath) {
			updateExecutionEnvironmentRequest.Files = append(updateExecutionEnvironmentRequest.Files, client.FileInfo{
				Name:          filepath.Base(dockerContextPath),
				Content:       fileContent,
				FormFieldName: "docker_context",
			})
		}
		if IsKnown(plan.DockerImage) {
			updateExecutionEnvironmentRequest.Files = append(updateExecutionEnvironmentRequest.Files, client.FileInfo{
				Name:          filepath.Base(plan.DockerImage.ValueString()),
				Content:       dockerImageContent,
				FormFieldName: "docker_image",
			})
		}

		if IsKnown(plan.DockerImageUri) {
			updateExecutionEnvironmentRequest.DockerImageUri = plan.DockerImageUri.ValueString()
		}

		if _, err := r.provider.service.CreateExecutionEnvironmentVersion(ctx, executionEnvironment.ID, updateExecutionEnvironmentRequest); err != nil {
			resp.Diagnostics.AddError("Error creating new Execution Environment Version", err.Error())
			return
		}
	}

	executionEnvironment, err = waitForExecutionEnvironmentToBeReady(ctx, r.provider.service, executionEnvironment.ID)
	if err != nil {
		resp.Diagnostics.AddError("Execution Environment failed to build", err.Error())
		return
	}
	plan.VersionID = types.StringValue(executionEnvironment.LatestVersion.ID)
	plan.BuildStatus = types.StringValue(executionEnvironment.LatestVersion.BuildStatus)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ExecutionEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ExecutionEnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteExecutionEnvironment")
	err := r.provider.service.DeleteExecutionEnvironment(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Execution Environment", err.Error())
			return
		}
	}
}

func (r *ExecutionEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r ExecutionEnvironmentResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("docker_context_path"),
			path.MatchRoot("docker_image"),
			path.MatchRoot("docker_image_uri"),
		),

		resourcevalidator.Conflicting(
			path.MatchRoot("docker_image"),
			path.MatchRoot("docker_image_uri"),
		),
	}
}

func (r ExecutionEnvironmentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Resource is being destroyed
		return
	}

	var plan ExecutionEnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var fileContent []byte = nil
	// compute docker context hash
	if IsKnown(plan.DockerContextPath) {
		var err error
		_, fileContent, err = getDockerContext(plan.DockerContextPath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error getting Docker context", err.Error())
			return
		}
	}

	var dockerImageContent []byte = nil
	if IsKnown(plan.DockerImage) {
		var err error
		if dockerImageContent, err = os.ReadFile(plan.DockerImage.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error getting Docker image", err.Error())
			return
		}
	}

	plan.DockerImageHash = types.StringValue(computeHash(dockerImageContent))
	plan.DockerContextHash = types.StringValue(computeHash(fileContent))
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)

	if req.State.Raw.IsNull() {
		// resource is being created
		return
	}

	var state ExecutionEnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.DockerContextHash != state.DockerContextHash || plan.DockerImageHash != state.DockerImageHash {
		plan.VersionID = types.StringUnknown()
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func getDockerContext(dockerContextPath string) (path string, fileContent []byte, err error) {
	path = dockerContextPath
	var fileInfo os.FileInfo
	if fileInfo, err = os.Stat(dockerContextPath); err != nil {
		return
	} else if fileInfo.IsDir() {
		zipPath := path + ".zip"
		if fileContent, err = zipDirectory(path, zipPath); err != nil {
			return
		}
		path = zipPath
	} else {
		if fileContent, err = os.ReadFile(path); err != nil {
			return
		}
	}

	return
}

func waitForExecutionEnvironmentToBeReady(ctx context.Context, service client.Service, id string) (*client.ExecutionEnvironment, error) {
	var executionEnvironment *client.ExecutionEnvironment
	var err error
	if executionEnvironment, err = service.GetExecutionEnvironment(ctx, id); err != nil {
		return nil, err
	}

	expBackoff := getExponentialBackoff()

	operation := func() error {
		traceAPICall("GetExecutionEnvironmentVersion")
		executionEnvironmentVersion, err := service.GetExecutionEnvironmentVersion(ctx, id, executionEnvironment.LatestVersion.ID)
		if err != nil {
			return backoff.Permanent(err)
		}
		if executionEnvironmentVersion.BuildStatus == "failed" {
			return backoff.Permanent(errors.New("execution environment failed to create, review the logs for more details"))
		}

		if executionEnvironmentVersion.BuildStatus != "success" {
			return errors.New("execution environment is not ready")
		}

		return nil
	}

	// Retry the operation using the backoff strategy
	if err = backoff.Retry(operation, expBackoff); err != nil {
		return nil, err
	}

	traceAPICall("GetExecutionEnvironment")
	return service.GetExecutionEnvironment(ctx, id)
}
