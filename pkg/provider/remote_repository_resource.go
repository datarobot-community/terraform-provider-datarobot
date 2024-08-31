package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RemoteRepositoryResource{}
var _ resource.ResourceWithImportState = &RemoteRepositoryResource{}

func NewRemoteRepositoryResource() resource.Resource {
	return &RemoteRepositoryResource{}
}

// VectorDatabaseResource defines the resource implementation.
type RemoteRepositoryResource struct {
	provider *Provider
}

func (r *RemoteRepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_remote_repository"
}

func (r *RemoteRepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "remote repository",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Remote Repository.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Remote Repository.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the Remote Repository.",
				Optional:            true,
			},
			"location": schema.StringAttribute{
				MarkdownDescription: "The location of the Remote Repository. (Bucket name for S3)",
				Required:            true,
			},
			"source_type": schema.StringAttribute{
				MarkdownDescription: "The source type of the Remote Repository.",
				Required:            true,
			},
			"personal_access_token": schema.StringAttribute{
				MarkdownDescription: "The personal access token for the Remote Repository.",
				Optional:            true,
			},

			// S3 remote repository specific attributes
			"aws_access_key_id": schema.StringAttribute{
				MarkdownDescription: "The AWS access key ID for the Remote Repository.",
				Optional:            true,
			},
			"aws_secret_access_key": schema.StringAttribute{
				MarkdownDescription: "The AWS secret access key for the Remote Repository.",
				Optional:            true,
			},
			"aws_session_token": schema.StringAttribute{
				MarkdownDescription: "The AWS session token for the Remote Repository.",
				Optional:            true,
			},
		},
	}
}

func (r *RemoteRepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RemoteRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RemoteRepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var credentialID string
	if IsKnown(data.PersonalAccessToken) {
		traceAPICall("CreateCredential")
		credential, err := r.provider.service.CreateCredential(ctx, &client.CredentialRequest{
			Name:           fmt.Sprintf("%s_%d", data.Name.ValueString(), time.Now().UnixNano()),
			Token:          data.PersonalAccessToken.ValueString(),
			RefreshToken:   "dummy",
			CredentialType: "oauth",
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating personal access token Credential", err.Error())
			return
		}
		credentialID = credential.ID
	} else if data.SourceType.ValueString() == "s3" && IsKnown(data.AWSAccessKeyID) && IsKnown(data.AWSSecretAccessKey) {
		traceAPICall("CreateCredential")
		credential, err := r.provider.service.CreateCredential(ctx, &client.CredentialRequest{
			Name:               fmt.Sprintf("%s_%s_%d", data.Name.ValueString(), data.Location.ValueString(), time.Now().UnixNano()),
			CredentialType:     "s3",
			AWSAccessKeyID:     data.AWSAccessKeyID.ValueString(),
			AWSSecretAccessKey: data.AWSSecretAccessKey.ValueString(),
			AWSSessionToken:    data.AWSSessionToken.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating S3 Credential", err.Error())
			return
		}
		credentialID = credential.ID
	}

	traceAPICall("CreateRemoteRepository")
	createResp, err := r.provider.service.CreateRemoteRepository(ctx, &client.CreateRemoteRepositoryRequest{
		Name:         data.Name.ValueString(),
		Description:  data.Description.ValueString(),
		Location:     data.Location.ValueString(),
		SourceType:   data.SourceType.ValueString(),
		CredentialID: credentialID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Remote Repository", err.Error())
		return
	}
	data.ID = types.StringValue(createResp.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *RemoteRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RemoteRepositoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetRemoteRepository")
	remoteRepository, err := r.provider.service.GetRemoteRepository(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Remote Repository not found",
				fmt.Sprintf("Remote Repository with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Remote Repository with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.Name = types.StringValue(remoteRepository.Name)
	data.Location = types.StringValue(remoteRepository.Location)
	data.SourceType = types.StringValue(remoteRepository.SourceType)
	if remoteRepository.Description != "" {
		data.Description = types.StringValue(remoteRepository.Description)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RemoteRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RemoteRepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state RemoteRepositoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("UpdateRemoteRepository")
	remoteRepository, err := r.provider.service.UpdateRemoteRepository(ctx,
		plan.ID.ValueString(),
		&client.UpdateRemoteRepositoryRequest{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			Location:    plan.Location.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Remote Repository not found",
				fmt.Sprintf("Remote Repository with ID %s is not found. Removing from state.", plan.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error updating Remote Repository", err.Error())
		}
		return
	}

	if state.PersonalAccessToken.ValueString() != plan.PersonalAccessToken.ValueString() {
		traceAPICall("UpdateCredential")
		_, err = r.provider.service.UpdateCredential(ctx, remoteRepository.CredentialID, &client.CredentialRequest{
			Name:         fmt.Sprintf("%s_%d", remoteRepository.Name, time.Now().UnixNano()),
			Token:        plan.PersonalAccessToken.ValueString(),
			RefreshToken: "dummy",
		})
		if err != nil {
			resp.Diagnostics.AddError("Error updating personal access token Credential", err.Error())
			return
		}
	}

	if state.AWSAccessKeyID.ValueString() != plan.AWSAccessKeyID.ValueString() ||
		state.AWSSecretAccessKey.ValueString() != plan.AWSSecretAccessKey.ValueString() ||
		state.AWSSessionToken.ValueString() != plan.AWSSessionToken.ValueString() {
		traceAPICall("UpdateCredential")
		_, err = r.provider.service.UpdateCredential(ctx, remoteRepository.CredentialID, &client.CredentialRequest{
			Name:               fmt.Sprintf("%s_%s_%d", remoteRepository.Name, remoteRepository.Location, time.Now().UnixNano()),
			AWSAccessKeyID:     plan.AWSAccessKeyID.ValueString(),
			AWSSecretAccessKey: plan.AWSSecretAccessKey.ValueString(),
			AWSSessionToken:    plan.AWSSessionToken.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error updating S3 Credential", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *RemoteRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RemoteRepositoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("GetRemoteRepository")
	remoteRepository, err := r.provider.service.GetRemoteRepository(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error getting Remote Repository info", err.Error())
			return
		}
	}

	traceAPICall("DeleteRemoteRepository")
	err = r.provider.service.DeleteRemoteRepository(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Remote Repository", err.Error())
			return
		}
	}

	if remoteRepository.CredentialID != "" {
		traceAPICall("DeleteCredential")
		err = r.provider.service.DeleteCredential(ctx, remoteRepository.CredentialID)
		if err != nil {
			resp.Diagnostics.AddError("Error deleting Credential", err.Error())
			return
		}
	}
}

func (r *RemoteRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
