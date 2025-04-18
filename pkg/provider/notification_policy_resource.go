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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &NotificationPolicyResource{}
var _ resource.ResourceWithImportState = &NotificationPolicyResource{}
var _ resource.ResourceWithConfigValidators = &NotificationPolicyResource{}

func NewNotificationPolicyResource() resource.Resource {
	return &NotificationPolicyResource{}
}

// NotificationPolicyResource defines the resource implementation.
type NotificationPolicyResource struct {
	provider *Provider
}

func (r *NotificationPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_policy"
}

func (r *NotificationPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Notification Policy",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Notification Policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Notification Policy.",
			},
			"channel_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Channel ID of the Notification Policy.",
			},
			"channel_scope": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Channel scope of the Notification Policy.",
				Validators:          NotificationPolicyChannelScopeValidators(),
			},
			"related_entity_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the related entity for the Notification Policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"related_entity_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Type of the related entity for the Notification Policy.",
				Validators:          NotificationRelatedEntityTypeValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether or not the Notification Policy is active.",
			},
			"event_group": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The group of the events that trigger the Notification.",
				Validators:          NotificationPolicyEventGroupValidators(),
			},
			"event_type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The group of the event that triggers the Notification.",
				Validators:          NotificationPolicyEventTypeValidators(),
			},
			"maximal_frequency": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The maximal frequency between policy runs in ISO 8601 duration string.",
			},
		},
	}
}

func (r *NotificationPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NotificationPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NotificationPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateNotificationPolicy")
	createResp, err := r.provider.service.CreateNotificationPolicy(ctx, &client.CreateNotificationPolicyRequest{
		Name:              data.Name.ValueString(),
		ChannelID:         data.ChannelID.ValueString(),
		ChannelScope:      data.ChannelScope.ValueString(),
		RelatedEntityID:   data.RelatedEntityID.ValueString(),
		RelatedEntityType: data.RelatedEntityType.ValueString(),
		Active:            data.Active.ValueBool(),
		EventGroup:        StringValuePointerOptional(data.EventGroup),
		EventType:         StringValuePointerOptional(data.EventType),
		MaximalFrequency:  StringValuePointerOptional(data.MaximalFrequency),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Notification Policy", err.Error())
		return
	}
	data.ID = types.StringValue(createResp.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NotificationPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NotificationPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetNotificationPolicy")
	notificationPolicy, err := r.provider.service.GetNotificationPolicy(
		ctx,
		data.RelatedEntityType.ValueString(),
		data.RelatedEntityID.ValueString(),
		data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Notification Policy not found",
				fmt.Sprintf("Notification Policy with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Notification Policy with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.Name = types.StringValue(notificationPolicy.Name)
	data.ChannelID = types.StringValue(notificationPolicy.ChannelID)
	data.ChannelScope = types.StringValue(notificationPolicy.ChannelScope)
	data.RelatedEntityID = types.StringValue(notificationPolicy.RelatedEntityID)
	data.RelatedEntityType = types.StringValue(notificationPolicy.RelatedEntityType)
	data.Active = types.BoolValue(notificationPolicy.Active)
	if notificationPolicy.EventGroup != nil {
		data.EventGroup = types.StringValue(notificationPolicy.EventGroup.ID)
	}
	if notificationPolicy.EventType != nil {
		data.EventType = types.StringValue(notificationPolicy.EventType.ID)
	}
	data.MaximalFrequency = types.StringPointerValue(notificationPolicy.MaximalFrequency)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NotificationPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("UpdateNotificationPolicy")
	_, err := r.provider.service.UpdateNotificationPolicy(
		ctx,
		data.RelatedEntityType.ValueString(),
		data.RelatedEntityID.ValueString(),
		data.ID.ValueString(),
		&client.UpdateNotificationPolicyRequest{
			Name:             StringValuePointerOptional(data.Name),
			ChannelID:        StringValuePointerOptional(data.ChannelID),
			ChannelScope:     StringValuePointerOptional(data.ChannelScope),
			EventGroup:       StringValuePointerOptional(data.EventGroup),
			EventType:        StringValuePointerOptional(data.EventType),
			Active:           BoolValuePointerOptional(data.Active),
			MaximalFrequency: StringValuePointerOptional(data.MaximalFrequency),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Notification Policy not found",
				fmt.Sprintf("Notification Policy with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error updating Notification Policy", err.Error())
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NotificationPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NotificationPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteNotificationPolicy")
	err := r.provider.service.DeleteNotificationPolicy(
		ctx,
		data.RelatedEntityType.ValueString(),
		data.RelatedEntityID.ValueString(),
		data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error getting Notification Policy info", err.Error())
			return
		}
	}
}

func (r *NotificationPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r NotificationPolicyResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("event_type"),
			path.MatchRoot("event_group"),
		),
	}
}
