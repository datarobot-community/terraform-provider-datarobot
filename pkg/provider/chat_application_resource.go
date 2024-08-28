package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ChatApplicationResource{}
var _ resource.ResourceWithImportState = &ChatApplicationResource{}

func NewChatApplicationResource() resource.Resource {
	return &ChatApplicationResource{}
}

// VectorDatabaseResource defines the resource implementation.
type ChatApplicationResource struct {
	provider *Provider
}

func (r *ChatApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_chat_application"
}

func (r *ChatApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Application",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The version ID of the Application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The URL of the Application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Application.",
			},
			"deployment_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The deployment ID of the Application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ChatApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ChatApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ChatApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateChatApplication")
	createResp, err := r.provider.service.CreateChatApplication(ctx, &client.CreateChatApplicationRequest{
		DeploymentID: data.DeploymentID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Application", err.Error())
		return
	}

	traceAPICall("UpdateChatApplication")
	_, err = r.provider.service.UpdateChatApplication(ctx,
		createResp.ID,
		&client.UpdateChatApplicationRequest{
			Name: data.Name.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Application not found",
				fmt.Sprintf("Application with ID %s is not found. Removing from state.", createResp.ID))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error updating Application", err.Error())
		}
		return
	}

	application, err := r.waitForChatApplicationToBeReady(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Application not ready", err.Error())
		return
	}
	data.ID = types.StringValue(application.ID)
	data.ApplicationUrl = types.StringValue(application.ApplicationUrl)

	traceAPICall("GetChatApplicationSource")
	applicationSource, err := r.provider.service.GetChatApplicationSource(ctx, application.CustomApplicationSourceID)
	if err != nil {
		resp.Diagnostics.AddError("Error getting Application Source info", err.Error())
		return
	}
	data.VersionID = types.StringValue(applicationSource.LatestVersion.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ChatApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ChatApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetChatApplication")
	application, err := r.provider.service.GetChatApplication(ctx, data.ID.ValueString())
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Application not found",
				fmt.Sprintf("Application with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error getting Application info", err.Error())
		}
		return
	}
	data.Name = types.StringValue(application.Name)
	data.ApplicationUrl = types.StringValue(application.ApplicationUrl)

	traceAPICall("GetChatApplicationSource")
	applicationSource, err := r.provider.service.GetChatApplicationSource(ctx, application.CustomApplicationSourceID)
	if err != nil {
		resp.Diagnostics.AddError("Error getting Application Source info", err.Error())
		return
	}
	data.VersionID = types.StringValue(applicationSource.LatestVersion.ID)

	// TODO: deployment ID

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ChatApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ChatApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ChatApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	newName := plan.Name.ValueString()
	if state.Name.ValueString() == newName {
		return
	}

	traceAPICall("UpdateApplication")
	_, err := r.provider.service.UpdateChatApplication(ctx,
		plan.ID.ValueString(),
		&client.UpdateChatApplicationRequest{
			Name: newName,
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Application not found",
				fmt.Sprintf("Application with ID %s is not found. Removing from state.", plan.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error updating Application", err.Error())
		}
		return
	}

	application, err := r.waitForChatApplicationToBeReady(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Application not ready", err.Error())
		return
	}

	traceAPICall("GetChatApplicationSource")
	applicationSource, err := r.provider.service.GetChatApplicationSource(ctx, application.CustomApplicationSourceID)
	if err != nil {
		resp.Diagnostics.AddError("Error getting Application Source info", err.Error())
		return
	}
	plan.VersionID = types.StringValue(applicationSource.LatestVersion.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ChatApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ChatApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteChatApplication")
	err := r.provider.service.DeleteChatApplication(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Application", err.Error())
			return
		}
	}
}

func (r *ChatApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ChatApplicationResource) waitForChatApplicationToBeReady(ctx context.Context, id string) (*client.ChatApplicationResponse, error) {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 30 * time.Second
	expBackoff.MaxElapsedTime = 30 * time.Minute

	operation := func() error {
		ready, err := r.provider.service.IsChatApplicationReady(ctx, id)
		if err != nil {
			return backoff.Permanent(err)
		}
		if !ready {
			return errors.New("application is not ready")
		}
		return nil
	}

	// Retry the operation using the backoff strategy
	err := backoff.Retry(operation, expBackoff)
	if err != nil {
		return nil, err
	}

	traceAPICall("GetChatApplication")
	return r.provider.service.GetChatApplication(ctx, id)
}
