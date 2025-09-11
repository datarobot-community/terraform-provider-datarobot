package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ExecutionEnvironmentResourceModel describes the execution environment resource.
type ExecutionEnvironmentResourceModel struct {
	ID                  types.String   `tfsdk:"id"`
	Name                types.String   `tfsdk:"name"`
	ProgrammingLanguage types.String   `tfsdk:"programming_language"`
	Description         types.String   `tfsdk:"description"`
	UseCases            []types.String `tfsdk:"use_cases"`
	VersionID           types.String   `tfsdk:"version_id"`
	VersionDescription  types.String   `tfsdk:"version_description"`
	DockerContextPath   types.String   `tfsdk:"docker_context_path"`
	DockerContextHash   types.String   `tfsdk:"docker_context_hash"`
	DockerImage         types.String   `tfsdk:"docker_image"`
	DockerImageHash     types.String   `tfsdk:"docker_image_hash"`
	DockerImageUri      types.String   `tfsdk:"docker_image_uri"`
	BuildStatus         types.String   `tfsdk:"build_status"`
}


// ExecutionEnvironmentDataSourceModel describes the execution environment data source resource.
type ExecutionEnvironmentDataSourceModel struct {
	Name                types.String `tfsdk:"name"`
	ID                  types.String `tfsdk:"id"`
	Description         types.String `tfsdk:"description"`
	ProgrammingLanguage types.String `tfsdk:"programming_language"`
	VersionID           types.String `tfsdk:"version_id"`
}
