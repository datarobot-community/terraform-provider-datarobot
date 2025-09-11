package collaboration

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/common"
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &UseCaseResource{}
var _ resource.ResourceWithImportState = &UseCaseResource{}

func NewUseCaseResource() resource.Resource {
	return &UseCaseResource{}
}

// UseCaseResource defines the resource implementation.
type UseCaseResource struct {
	service client.Service
}

func (r *UseCaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_use_case"
}

func (r *UseCaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Use case",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Use Case.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Use Case.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the Use Case.",
				Optional:            true,
			},
		},
	}
}

func (r *UseCaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil { return }
	accessor, ok := req.ProviderData.(common.ServiceAccessor)
	if !ok {
			resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
			return
	}
	r.service = accessor.GetService()
}

func (r *UseCaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.UseCaseResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("CreateUseCase")
	createResp, err := r.service.CreateUseCase(ctx, &client.UseCaseRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Use Case", err.Error())
		return
	}
	data.ID = types.StringValue(createResp.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *UseCaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.UseCaseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	common.TraceAPICall("GetUseCase")
	useCase, err := r.service.GetUseCase(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Use Case not found",
				fmt.Sprintf("Use Case with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Use Case with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.Name = types.StringValue(useCase.Name)
	if useCase.Description != "" {
		data.Description = types.StringValue(useCase.Description)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UseCaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data models.UseCaseResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("UpdateUseCase")
	_, err := r.service.UpdateUseCase(ctx,
		data.ID.ValueString(),
		&client.UseCaseRequest{
			Name:        data.Name.ValueString(),
			Description: data.Description.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Use Case not found",
				fmt.Sprintf("Use Case with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error updating Use Case",
				fmt.Sprintf("Unable to update Use Case, got error: %s", err),
			)
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *UseCaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.UseCaseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("DeleteUseCase")
	err := r.service.DeleteUseCase(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error getting Use Case info", err.Error())
			return
		}
	}
}

func (r *UseCaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
