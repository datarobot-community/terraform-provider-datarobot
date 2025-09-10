package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/common"
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AwsCredentialResource{}
var _ resource.ResourceWithImportState = &AwsCredentialResource{}
var _ resource.ResourceWithConfigValidators = &AwsCredentialResource{}

func NewAwsCredentialResource() resource.Resource {
	return &AwsCredentialResource{}
}

// AwsCredentialResource defines the resource implementation.
type AwsCredentialResource struct {
	service client.Service
}

func (r *AwsCredentialResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_credential"
}

func (r *AwsCredentialResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "AWS Credential",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the AWS Credential.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the AWS Credential.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the AWS Credential.",
				Optional:            true,
			},
			"aws_access_key_id": schema.StringAttribute{
				MarkdownDescription: "The AWS Access Key ID.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"aws_secret_access_key": schema.StringAttribute{
				MarkdownDescription: "The AWS Secret Access Key.",
				Sensitive:           true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"aws_session_token": schema.StringAttribute{
				MarkdownDescription: "The AWS Session Token.",
				Sensitive:           true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the saved shared secure configuration. If specified, cannot include awsAccessKeyId, awsSecretAccessKey or awsSessionToken.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *AwsCredentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
		// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil { return }
	accessor, ok := req.ProviderData.(common.ServiceAccessor)
	if !ok {
			resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
			return
	}
	r.service = accessor.GetService()
}

func (r *AwsCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.AwsCredentialResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("CreateAwsCredential")
	createResp, err := r.service.CreateCredential(ctx, &client.CredentialRequest{
		Name:               data.Name.ValueString(),
		Description:        data.Description.ValueString(),
		CredentialType:     client.CredentialTypeS3,
		AWSAccessKeyID:     data.AWSAccessKeyID.ValueString(),
		AWSSecretAccessKey: data.AWSSecretAccessKey.ValueString(),
		AWSSessionToken:    data.AWSSessionToken.ValueString(),
		ConfigID:           data.ConfigID.ValueString(),
	})
	if err != nil {
		errMessage := common.CheckCredentialNameAlreadyExists(err, data.Name.ValueString())
		resp.Diagnostics.AddError("Error creating AWS Credential", errMessage)
		return
	}
	data.ID = types.StringValue(createResp.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *AwsCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.AwsCredentialResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	common.TraceAPICall("GetAwsCredential")
	credential, err := r.service.GetCredential(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"AWS Credential not found",
				fmt.Sprintf("AWS Credential with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting AWS Credential with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.Name = types.StringValue(credential.Name)
	if credential.Description != "" {
		data.Description = types.StringValue(credential.Description)
	}
	if credential.ConfigID != "" {
		data.ConfigID = types.StringValue(credential.ConfigID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AwsCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data models.AwsCredentialResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("UpdateAwsCredential")
	_, err := r.service.UpdateCredential(ctx,
		data.ID.ValueString(),
		&client.CredentialRequest{
			Name:        data.Name.ValueString(),
			Description: data.Description.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"AWS Credential not found",
				fmt.Sprintf("AWS Credential with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			errMessage := common.CheckCredentialNameAlreadyExists(err, data.Name.ValueString())
			resp.Diagnostics.AddError("Error updating AWS Credential", errMessage)
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *AwsCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.AwsCredentialResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("DeleteAwsCredential")
	err := r.service.DeleteCredential(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting AWS Credential", err.Error())
			return
		}
	}
}

func (r *AwsCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r AwsCredentialResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("config_id"),
			path.MatchRoot("aws_access_key_id"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("config_id"),
			path.MatchRoot("aws_secret_access_key"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("config_id"),
			path.MatchRoot("aws_session_token"),
		),
	}
}
