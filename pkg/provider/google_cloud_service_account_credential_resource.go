package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

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
var _ resource.Resource = &GoogleCloudCredentialResource{}
var _ resource.ResourceWithImportState = &GoogleCloudCredentialResource{}
var _ resource.ResourceWithModifyPlan = &GoogleCloudCredentialResource{}

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
			"gcp_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The GCP key in JSON format.",
			},
			"gcp_key_file": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The file that has the GCP key. Cannot be used with `gcp_key`.",
			},
			"gcp_key_file_hash": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The hash of the GCP key file contents.",
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

	gcpKey, err := r.getGCPKey(data)
	if err != nil {
		resp.Diagnostics.AddError("Error getting GCP key", err.Error())
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

	gcpKey, err := r.getGCPKey(data)
	if err != nil {
		resp.Diagnostics.AddError("Error getting GCP key", err.Error())
		return
	}

	traceAPICall("UpdateGoogleCloudCredential")
	_, err = r.provider.service.UpdateCredential(ctx,
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

func (r GoogleCloudCredentialResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("gcp_key"),
			path.MatchRoot("gcp_key_file"),
		),
	}
}

func (r GoogleCloudCredentialResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Resource is being destroyed
		return
	}

	var plan GoogleCloudCredentialResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// compute gcp key file content hash
	hash := types.StringNull()
	if IsKnown(plan.GCPKeyFile) {
		fileContentHash, err := computeFileHash(plan.GCPKeyFile.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error calculating gcp key file hash", err.Error())
			return
		}
		hash = types.StringValue(fileContentHash)
	}
	plan.GCPKeyFileHash = hash

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *GoogleCloudCredentialResource) getGCPKey(data GoogleCloudCredentialResourceModel) (gcpKey client.GCPKey, err error) {
	var gcpKeyBytes []byte
	if IsKnown(data.GCPKey) {
		gcpKeyBytes = []byte(data.GCPKey.ValueString())
	} else {
		if gcpKeyBytes, err = r.getGCPKeyFromFile(data.GCPKeyFile.ValueString()); err != nil {
			return
		}
	}

	err = json.Unmarshal(gcpKeyBytes, &gcpKey)
	return
}

func (r *GoogleCloudCredentialResource) getGCPKeyFromFile(filePath string) (
	fileContent []byte,
	err error,
) {
	fileReader, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer fileReader.Close()

	fileContent, err = io.ReadAll(fileReader)
	if err != nil {
		return
	}

	return
}
