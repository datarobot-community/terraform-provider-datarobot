package models

import "github.com/hashicorp/terraform-plugin-framework/types"

// RemoteRepositoryResourceModel describes the remote repository resource.
type RemoteRepositoryResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Location            types.String `tfsdk:"location"`
	SourceType          types.String `tfsdk:"source_type"`
	PersonalAccessToken types.String `tfsdk:"personal_access_token"`

	// optional fields for S3 remote repositories
	AWSAccessKeyID     types.String `tfsdk:"aws_access_key_id"`
	AWSSecretAccessKey types.String `tfsdk:"aws_secret_access_key"`
	AWSSessionToken    types.String `tfsdk:"aws_session_token"`
}
