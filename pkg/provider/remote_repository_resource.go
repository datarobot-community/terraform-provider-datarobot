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
				Required:            true,
			},
			"location": schema.StringAttribute{
				MarkdownDescription: "The location of the Remote Repository.",
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
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan RemoteRepositoryResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !IsKnown(plan.Name) {
		resp.Diagnostics.AddError(
			"Invalid name",
			"Name is required to create a Remote Repository.",
		)
		return
	}

	if !IsKnown(plan.Location) {
		resp.Diagnostics.AddError(
			"Invalid location",
			"Location is required to create a Remote Repository.",
		)
		return
	}

	if !IsKnown(plan.SourceType) {
		resp.Diagnostics.AddError(
			"Invalid source type",
			"Source Type is required to create a Remote Repository.",
		)
		return
	}

	var credentialID string
	if IsKnown(plan.PersonalAccessToken) {
		traceAPICall("CreateCredential")
		credential, err := r.provider.service.CreateCredential(ctx, &client.CredentialRequest{
			Name:           fmt.Sprintf("%s_%d", plan.Name.ValueString(), time.Now().UnixNano()),
			Token:          plan.PersonalAccessToken.ValueString(),
			RefreshToken:   "dummy",
			CredentialType: "oauth",
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating personal access token Credential",
				fmt.Sprintf("Unable to create personal access token Credential, got error: %s", err),
			)
			return
		}
		credentialID = credential.ID
	}

	traceAPICall("CreateRemoteRepository")
	createResp, err := r.provider.service.CreateRemoteRepository(ctx, &client.CreateRemoteRepositoryRequest{
		Name:         plan.Name.ValueString(),
		Description:  plan.Description.ValueString(),
		Location:     plan.Location.ValueString(),
		SourceType:   plan.SourceType.ValueString(),
		CredentialID: credentialID,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Remote Repository",
			fmt.Sprintf("Unable to create Remote Repository, got error: %s", err),
		)
		return
	}

	var state RemoteRepositoryResourceModel
	loadRemoteRepositoryToTerraformState(
		createResp.ID,
		createResp.Name,
		createResp.Description,
		createResp.Location,
		createResp.SourceType,
		plan.PersonalAccessToken.ValueString(),
		&state,
	)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *RemoteRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state RemoteRepositoryResourceModel
	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() {
		return
	}

	id := state.ID.ValueString()

	traceAPICall("GetRemoteRepository")
	remoteRepository, err := r.provider.service.GetRemoteRepository(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Remote Repository not found",
				fmt.Sprintf("Remote Repository with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting Remote Repository info",
				fmt.Sprintf("Unable to get Remote Repository, got error: %s", err),
			)
		}
		return
	}

	loadRemoteRepositoryToTerraformState(
		id,
		remoteRepository.Name,
		remoteRepository.Description,
		remoteRepository.Location,
		remoteRepository.SourceType,
		state.PersonalAccessToken.ValueString(),
		&state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *RemoteRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan RemoteRepositoryResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state RemoteRepositoryResourceModel

	// Read Terraform state data into the model
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the only fields that can be updated don't change, just return.
	newName := plan.Name.ValueString()
	newDescription := plan.Description.ValueString()
	newLocation := plan.Location.ValueString()
	newPersonalAccessToken := plan.PersonalAccessToken.ValueString()

	if state.Name.ValueString() == newName &&
		state.Description.ValueString() == newDescription &&
		state.Location.ValueString() == newLocation &&
		state.PersonalAccessToken.ValueString() == newPersonalAccessToken {
		return
	}

	id := state.ID.ValueString()

	traceAPICall("UpdateRemoteRepository")
	remoteRepository, err := r.provider.service.UpdateRemoteRepository(ctx,
		id,
		&client.UpdateRemoteRepositoryRequest{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
			Location:    plan.Location.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Remote Repository not found",
				fmt.Sprintf("Remote Repository with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error updating Remote Repository",
				fmt.Sprintf("Unable to update Remote Repository, got error: %s", err),
			)
		}
		return
	}

	if state.PersonalAccessToken.ValueString() != newPersonalAccessToken {
		traceAPICall("UpdateCredential")
		_, err = r.provider.service.UpdateCredential(ctx, remoteRepository.CredentialID, &client.CredentialRequest{
			Name:         fmt.Sprintf("%s_%d", remoteRepository.Name, time.Now().UnixNano()),
			Token:        newPersonalAccessToken,
			RefreshToken: "dummy",
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating personal access token Credential",
				fmt.Sprintf("Unable to update personal access token Credential, got error: %s", err),
			)
			return
		}
	}

	loadRemoteRepositoryToTerraformState(
		id,
		remoteRepository.Name,
		remoteRepository.Description,
		remoteRepository.Location,
		remoteRepository.SourceType,
		plan.PersonalAccessToken.ValueString(),
		&state,
	)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *RemoteRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state RemoteRepositoryResourceModel

	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() {
		return
	}

	id := state.ID.ValueString()

	traceAPICall("GetRemoteRepository")
	remoteRepository, err := r.provider.service.GetRemoteRepository(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			// remote repository is already gone, ignore the error and remove from state
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting Remote Repository info",
				fmt.Sprintf("Unable to get Remote Repository, got error: %s", err),
			)
		}
		return
	}

	traceAPICall("DeleteRemoteRepository")
	err = r.provider.service.DeleteRemoteRepository(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			// remote repository is already gone, ignore the error and remove from state
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error deleting Remote Repository",
				fmt.Sprintf("Unable to delete remote repository, got error: %s", err),
			)
		}
		return
	}

	if remoteRepository.CredentialID != "" {
		traceAPICall("DeleteCredential")
		err = r.provider.service.DeleteCredential(ctx, remoteRepository.CredentialID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting personal access token Credential",
				fmt.Sprintf("Unable to delete personal access token Credential, got error: %s", err),
			)
			return
		}
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r *RemoteRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func loadRemoteRepositoryToTerraformState(
	id,
	name,
	description,
	location,
	sourceType string,
	personalAccessToken string,
	state *RemoteRepositoryResourceModel,
) {
	state.ID = types.StringValue(id)
	state.Name = types.StringValue(name)
	state.Description = types.StringValue(description)
	state.Location = types.StringValue(location)
	state.SourceType = types.StringValue(sourceType)
	if personalAccessToken == "" {
		state.PersonalAccessToken = types.StringNull()
	} else {
		state.PersonalAccessToken = types.StringValue(personalAccessToken)
	}
}
