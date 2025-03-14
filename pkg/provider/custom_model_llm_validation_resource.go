package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CustomModelLLMValidationResource{}
var _ resource.ResourceWithImportState = &CustomModelLLMValidationResource{}

func NewCustomModelLLMValidationResource() resource.Resource {
	return &CustomModelLLMValidationResource{}
}

// CustomModelLLMValidationResource defines the resource implementation.
type CustomModelLLMValidationResource struct {
	provider *Provider
}

func (r *CustomModelLLMValidationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_model_llm_validation"
}

func (r *CustomModelLLMValidationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Custom Model LLM Validation",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the custom model LLM validation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployment_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the custom model deployment.",
			},
			"model_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The ID of the model used in the deployment.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("Untitled"),
				MarkdownDescription: "The name to use for the validated custom model.",
			},
			"prediction_timeout": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(300),
				MarkdownDescription: "The timeout in seconds for the prediction when validating a custom model. Defaults to 300.",
			},
			"prompt_column_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The name of the column the custom model uses for prompt text input.",
			},
			"target_column_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The name of the column the custom model uses for prediction output.",
			},
			"chat_model_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the chat model to use for the custom model LLM validation.",
			},
			"use_case_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the use case to associate with the validated custom model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *CustomModelLLMValidationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CustomModelLLMValidationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CustomModelLLMValidationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateCustomModelLLMValidation")
	customModelLlmValidation, statusID, err := r.provider.service.CreateCustomModelLLMValidation(ctx, &client.CreateCustomModelLLMValidationRequest{
		DeploymentID:      data.DeploymentID.ValueString(),
		ModelID:           StringValuePointerOptional(data.ModelID),
		Name:              StringValuePointerOptional(data.Name),
		PredictionTimeout: Int64ValuePointerOptional(data.PredictionTimeout),
		PromptColumnName:  StringValuePointerOptional(data.PromptColumnName),
		TargetColumnName:  StringValuePointerOptional(data.TargetColumnName),
		ChatModelID:       StringValuePointerOptional(data.ChatModelID),
		UseCaseID:         StringValuePointerOptional(data.UseCaseID),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating custom model LLM validation", err.Error())
		return
	}

	err = waitForGenAITaskStatusToComplete(ctx, r.provider.service, statusID)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for custom model LLM validation to complete", err.Error())
		return
	}

	customModelLlmValidation, err = r.provider.service.GetCustomModelLLMValidation(ctx, customModelLlmValidation.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error getting custom model LLM validation", err.Error())
		return
	}

	if customModelLlmValidation.ValidationStatus == "FAILED" {
		resp.Diagnostics.AddError("Custom model LLM validation failed", *customModelLlmValidation.ErrorMessage)
		return
	}
	data.ID = types.StringValue(customModelLlmValidation.ID)
	data.ModelID = types.StringValue(customModelLlmValidation.ModelID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomModelLLMValidationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CustomModelLLMValidationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetCustomModelLLMValidation")
	customModelLlmValidation, err := r.provider.service.GetCustomModelLLMValidation(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"custom model LLM validation not found",
				fmt.Sprintf("custom model LLM validation with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting custom model LLM validation with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.Name = types.StringValue(customModelLlmValidation.Name)
	data.DeploymentID = types.StringValue(customModelLlmValidation.DeploymentID)
	data.ModelID = types.StringValue(customModelLlmValidation.ModelID)
	data.PromptColumnName = types.StringValue(customModelLlmValidation.PromptColumnName)
	data.TargetColumnName = types.StringValue(customModelLlmValidation.TargetColumnName)
	data.ChatModelID = types.StringPointerValue(customModelLlmValidation.ChatModelID)
	data.PredictionTimeout = types.Int64Value(customModelLlmValidation.PredictionTimeout)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomModelLLMValidationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CustomModelLLMValidationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CustomModelLLMValidationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("UpdateCustomModelLLMValidation")
	_, err := r.provider.service.UpdateCustomModelLLMValidation(ctx,
		plan.ID.ValueString(),
		&client.UpdateCustomModelLLMValidationRequest{
			Name:              StringValuePointerOptional(plan.Name),
			DeploymentID:      StringValuePointerOptional(plan.DeploymentID),
			ModelID:           StringValuePointerOptional(plan.ModelID),
			PromptColumnName:  StringValuePointerOptional(plan.PromptColumnName),
			TargetColumnName:  StringValuePointerOptional(plan.TargetColumnName),
			ChatModelID:       StringValuePointerOptional(plan.ChatModelID),
			PredictionTimeout: Int64ValuePointerOptional(plan.PredictionTimeout),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"custom model LLM validation not found",
				fmt.Sprintf("custom model LLM validation with ID %s is not found. Removing from state.", plan.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error updating custom model LLM validation", err.Error())
		}
		return
	}

	customModelLLMValidation, err := r.provider.service.GetCustomModelLLMValidation(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting custom model LLM validation", err.Error())
		return
	}
	plan.ModelID = types.StringValue(customModelLLMValidation.ModelID)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *CustomModelLLMValidationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CustomModelLLMValidationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteCustomModelLLMValidation")
	err := r.provider.service.DeleteCustomModelLLMValidation(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting custom model LLM validation", err.Error())
			return
		}
	}
}

func (r *CustomModelLLMValidationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r CustomModelLLMValidationResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("chat_model_id"),
			path.MatchRoot("prompt_column_name"),
			path.MatchRoot("target_column_name"),
		),
	}
}
