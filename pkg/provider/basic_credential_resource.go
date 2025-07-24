package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BasicCredentialResource{}
var _ resource.ResourceWithImportState = &BasicCredentialResource{}

func NewBasicCredentialResource() resource.Resource {
	return &BasicCredentialResource{}
}

// BasicCredentialResource defines the resource implementation.
type BasicCredentialResource struct {
	provider *Provider
}

func (r *BasicCredentialResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_basic_credential"
}

func (r *BasicCredentialResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Basic Credential",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Basic Credential.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Basic Credential.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the Basic Credential.",
				Optional:            true,
			},
			"user": schema.StringAttribute{
				MarkdownDescription: "The user of the Basic Credential.",
				Sensitive:           true,
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password of the Basic Credential.",
				Sensitive:           true,
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				// NOTE: We are leaving the validator in to enforce the existence of a password
				// (as a password is required by DataRobot)
				// but we are not enforcing any particular length.
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *BasicCredentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BasicCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BasicCredentialResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateBasicCredential")
	createResp, err := r.provider.service.CreateCredential(ctx, &client.CredentialRequest{
		Name:           data.Name.ValueString(),
		Description:    data.Description.ValueString(),
		CredentialType: client.CredentialTypeBasic,
		User:           data.User.ValueString(),
		Password:       data.Password.ValueString(),
	})
	if err != nil {
		errMessage := checkCredentialNameAlreadyExists(err, data.Name.ValueString())
		resp.Diagnostics.AddError("Error creating Basic Credential", errMessage)
		return
	}
	data.ID = types.StringValue(createResp.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *BasicCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BasicCredentialResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetBasicCredential")
	credential, err := r.provider.service.GetCredential(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Basic Credential not found",
				fmt.Sprintf("Basic Credential with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Basic Credential with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}

	data.Name = types.StringValue(credential.Name)
	if credential.Description != "" {
		data.Description = types.StringValue(credential.Description)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BasicCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BasicCredentialResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("UpdateBasicCredential")
	_, err := r.provider.service.UpdateCredential(ctx,
		data.ID.ValueString(),
		&client.CredentialRequest{
			Name:        data.Name.ValueString(),
			Description: data.Description.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Basic Credential not found",
				fmt.Sprintf("Basic Credential with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			errMessage := checkCredentialNameAlreadyExists(err, data.Name.ValueString())
			resp.Diagnostics.AddError("Error updating Basic Credential", errMessage)
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *BasicCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BasicCredentialResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteBasicCredential")
	err := r.provider.service.DeleteCredential(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Basic Credential", err.Error())
			return
		}
	}
}

func (r *BasicCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
