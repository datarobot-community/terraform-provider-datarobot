package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type AppOAuthResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	OrgID          types.String `tfsdk:"org_id"`
	Type           types.String `tfsdk:"type"`
	ClientID       types.String `tfsdk:"client_id"`
	ClientSecret   types.String `tfsdk:"client_secret"`
	SecureConfigID types.String `tfsdk:"secure_config_id"`
	Status         types.String `tfsdk:"status"`
}

type ApplicationSourceFromTemplateResourceModel struct {
	ID                       types.String                `tfsdk:"id"`
	VersionID                types.String                `tfsdk:"version_id"`
	TemplateID               types.String                `tfsdk:"template_id"`
	Name                     types.String                `tfsdk:"name"`
	BaseEnvironmentID        types.String                `tfsdk:"base_environment_id"`
	BaseEnvironmentVersionID types.String                `tfsdk:"base_environment_version_id"`
	FolderPath               types.String                `tfsdk:"folder_path"`
	FolderPathHash           types.String                `tfsdk:"folder_path_hash"`
	Files                    types.Dynamic                  `tfsdk:"files"`
	FilesHashes              types.List                  `tfsdk:"files_hashes"`
	Resources                *ApplicationSourceResources `tfsdk:"resources"`
	RuntimeParameterValues   types.List                  `tfsdk:"runtime_parameter_values"`
}

type ApplicationSourceResources struct {
	Replicas                     types.Int64  `tfsdk:"replicas"`
	SessionAffinity              types.Bool   `tfsdk:"session_affinity"`
	ResourceLabel                types.String `tfsdk:"resource_label"`
	ServiceWebRequestsOnRootPath types.Bool   `tfsdk:"service_web_requests_on_root_path"`
}


type ApplicationSourceResourceModel struct {
	ID                       types.String                `tfsdk:"id"`
	VersionID                types.String                `tfsdk:"version_id"`
	Name                     types.String                `tfsdk:"name"`
	BaseEnvironmentID        types.String                `tfsdk:"base_environment_id"`
	BaseEnvironmentVersionID types.String                `tfsdk:"base_environment_version_id"`
	FolderPath               types.String                `tfsdk:"folder_path"`
	FolderPathHash           types.String                `tfsdk:"folder_path_hash"`
	Files                    types.Dynamic               `tfsdk:"files"`
	FilesHashes              types.List                  `tfsdk:"files_hashes"`
	Resources                *ApplicationSourceResources `tfsdk:"resources"`
	RuntimeParameterValues   types.List                  `tfsdk:"runtime_parameter_values"`
}

type QAApplicationResourceModel struct {
	ID                       types.String   `tfsdk:"id"`
	SourceID                 types.String   `tfsdk:"source_id"`
	SourceVersionID          types.String   `tfsdk:"source_version_id"`
	Name                     types.String   `tfsdk:"name"`
	DeploymentID             types.String   `tfsdk:"deployment_id"`
	ApplicationUrl           types.String   `tfsdk:"application_url"`
	ExternalAccessEnabled    types.Bool     `tfsdk:"external_access_enabled"`
	ExternalAccessRecipients []types.String `tfsdk:"external_access_recipients"`
	AllowAutoStopping        types.Bool     `tfsdk:"allow_auto_stopping"`
}


type CustomApplicationFromEnvironmentResourceModel struct {
	ID                       types.String                `tfsdk:"id"`
	EnvironmentID            types.String                `tfsdk:"environment_id"`
	EnvironmentVersionID     types.String                `tfsdk:"environment_version_id"`
	Name                     types.String                `tfsdk:"name"`
	ApplicationUrl           types.String                `tfsdk:"application_url"`
	ExternalAccessEnabled    types.Bool                  `tfsdk:"external_access_enabled"`
	ExternalAccessRecipients []types.String              `tfsdk:"external_access_recipients"`
	AllowAutoStopping        types.Bool                  `tfsdk:"allow_auto_stopping"`
	Resources                *ApplicationSourceResources `tfsdk:"resources"`
	UseCaseIDs               []types.String              `tfsdk:"use_case_ids"`
}


type CustomApplicationResourceModel struct {
	ID                       types.String                `tfsdk:"id"`
	SourceID                 types.String                `tfsdk:"source_id"`
	SourceVersionID          types.String                `tfsdk:"source_version_id"`
	Name                     types.String                `tfsdk:"name"`
	ApplicationUrl           types.String                `tfsdk:"application_url"`
	ExternalAccessEnabled    types.Bool                  `tfsdk:"external_access_enabled"`
	ExternalAccessRecipients []types.String              `tfsdk:"external_access_recipients"`
	AllowAutoStopping        types.Bool                  `tfsdk:"allow_auto_stopping"`
	Resources                *ApplicationSourceResources `tfsdk:"resources"`
	UseCaseIDs               []types.String              `tfsdk:"use_case_ids"`
}
