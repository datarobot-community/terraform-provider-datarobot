package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CredentialResourceModel describes the credential resource.
type ApiTokenCredentialResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ApiToken    types.String `tfsdk:"api_token"`
}

type BasicCredentialResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	User        types.String `tfsdk:"user"`
	Password    types.String `tfsdk:"password"`
}

type GoogleCloudCredentialResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	GCPKey         types.String `tfsdk:"gcp_key"`
	GCPKeyFile     types.String `tfsdk:"gcp_key_file"`
	GCPKeyFileHash types.String `tfsdk:"gcp_key_file_hash"`
}

type AwsCredentialResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	AWSAccessKeyID     types.String `tfsdk:"aws_access_key_id"`
	AWSSecretAccessKey types.String `tfsdk:"aws_secret_access_key"`
	AWSSessionToken    types.String `tfsdk:"aws_session_token"`
	ConfigID           types.String `tfsdk:"config_id"`
}

type AzureCredentialResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	AzureConnectionString types.String `tfsdk:"azure_connection_string"`
}
