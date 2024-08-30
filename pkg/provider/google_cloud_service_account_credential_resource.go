package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &GoogleCloudCredentialResource{}
var _ resource.ResourceWithImportState = &GoogleCloudCredentialResource{}

func NewGoogleCloudCredentialResource() resource.Resource {
	return &GoogleCloudCredentialResource{}
}

// GoogleCloudCredentialResource defines the resource implementation.
type GoogleCloudCredentialResource struct {
	provider *Provider
}

func (r *GoogleCloudCredentialResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_google_cloud_credential"
}

func (r *GoogleCloudCredentialResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Api Token Credential",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Google Cloud Credential.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Google Cloud Credential.",
				Required:            true,
			},
			"source_file": schema.StringAttribute{
				MarkdownDescription: "The source file of the Google Cloud Credential.",
				Required:            true,
			},
		},
	}
}

func (r *GoogleCloudCredentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GoogleCloudCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GoogleCloudCredentialResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	gcpKey, errSummary, errDetail := r.getGCPKeyFromFile(data.SourceFile.ValueString())
	if errSummary != "" {
		resp.Diagnostics.AddError(errSummary, errDetail)
		return
	}

	traceAPICall("CreateGoogleCloudCredential")
	createResp, err := r.provider.service.CreateCredential(ctx, &client.CredentialRequest{
		Name:           data.Name.ValueString(),
		CredentialType: client.CredentialTypeGCP,
		GCPKey:         &gcpKey,
	})
	if err != nil {
		errMessage := checkCredentialNameAlreadyExists(err, data.Name.ValueString())
		resp.Diagnostics.AddError("Error creating Google Cloud Credential", errMessage)
		return
	}

	data.ID = types.StringValue(createResp.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *GoogleCloudCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GoogleCloudCredentialResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetGoogleCloudCredential")
	credential, err := r.provider.service.GetCredential(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Google Cloud Credential not found",
				fmt.Sprintf("Google Cloud Credential with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Google Cloud Credential with ID %s", data.ID.ValueString()), 
				err.Error())
		}
		return
	}

	data.Name = types.StringValue(credential.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GoogleCloudCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data GoogleCloudCredentialResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	gcpKey, errSummary, errDetail := r.getGCPKeyFromFile(data.SourceFile.ValueString())
	if errSummary != "" {
		resp.Diagnostics.AddError(errSummary, errDetail)
		return
	}

	traceAPICall("UpdateGoogleCloudCredential")
	_, err := r.provider.service.UpdateCredential(ctx,
		data.ID.ValueString(),
		&client.CredentialRequest{
			Name:   data.Name.ValueString(),
			GCPKey: &gcpKey,
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Google Cloud Credential not found",
				fmt.Sprintf("Google Cloud Credential with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			errMessage := checkCredentialNameAlreadyExists(err, data.Name.ValueString())
			resp.Diagnostics.AddError("Error updating Google Cloud Credential", errMessage)
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *GoogleCloudCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GoogleCloudCredentialResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteGoogleCloudCredential")
	err := r.provider.service.DeleteCredential(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Google Cloud Credential", err.Error())
			return
		}
	}
}

func (r *GoogleCloudCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *GoogleCloudCredentialResource) getGCPKeyFromFile(filePath string) (
	gcpKey client.GCPKey,
	errSummary string,
	errDetail string,
) {
	fileReader, err := os.Open(filePath)
	if err != nil {
		errSummary = "Error opening source file"
		errDetail = err.Error()
		return
	}
	defer fileReader.Close()

	fileContent, err := io.ReadAll(fileReader)
	if err != nil {
		errSummary = "Error reading source file"
		errDetail = err.Error()
		return
	}

	err = json.Unmarshal(fileContent, &gcpKey)
	if err != nil {
		errSummary = "Error parsing source file"
		errDetail = err.Error()
		return
	}

	return
}
