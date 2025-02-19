package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &NotificationChannelResource{}
var _ resource.ResourceWithImportState = &NotificationChannelResource{}

func NewNotificationChannelResource() resource.Resource {
	return &NotificationChannelResource{}
}

// NotificationChannelResource defines the resource implementation.
type NotificationChannelResource struct {
	provider *Provider
}

func (r *NotificationChannelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_channel"
}

func (r *NotificationChannelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Notification Channel",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Notification Channel.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Notification Channel.",
			},
			"channel_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Type of Notification Channel.",
				Validators:          ChannelTypeValidators(),
			},
			"content_type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The content type of the messages of the Notification Channel.",
				Validators:          ContentTypeValidators(),
			},
			"custom_headers": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Custom headers and their values to be sent in the Notification Channel.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The name of the header.",
						},
						"value": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The value of the header.",
						},
					},
				},
			},
			"dr_entities": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The IDs of the DataRobot Users, Group or Custom Job associated with the DataRobotUser, DataRobotGroup or DataRobotCustomJob channel types.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The ID of the DataRobot entity.",
						},
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The name of the entity.",
						},
					},
				},
			},
			"email_address": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The email address to be used in the Notification Channel.",
			},
			"language_code": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("en"),
				MarkdownDescription: "The preferred language code.",
				Validators:          LanguageCodeValidators(),
			},
			"payload_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The payload URL of the Notification Channel.",
			},
			"related_entity_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of related entity.",
			},
			"related_entity_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The type of related entity.",
				Validators:          NotificationRelatedEntityTypeValidators(),
			},
			"secret_token": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The secret token to be used for the Notification Channel.",
			},
			"validate_ssl": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Defines if validate ssl or not in the Notification Channel.",
			},
			"verification_code": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Required if the channel type is email.",
			},
		},
	}
}

func (r *NotificationChannelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NotificationChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NotificationChannelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := &client.CreateNotificationChannelRequest{
		Name:              data.Name.ValueString(),
		ChannelType:       data.ChannelType.ValueString(),
		ContentType:       StringValuePointerOptional(data.ContentType),
		RelatedEntityID:   data.RelatedEntityID.ValueString(),
		RelatedEntityType: data.RelatedEntityType.ValueString(),
		PayloadUrl:        StringValuePointerOptional(data.PayloadUrl),
		SecretToken:       StringValuePointerOptional(data.SecretToken),
		ValidateSsl:       BoolValuePointerOptional(data.ValidateSsl),
		VerificationCode:  StringValuePointerOptional(data.VerificationCode),
	}

	if len(data.CustomHeaders) > 0 {
		customHeaders := make([]client.CustomHeader, len(data.CustomHeaders))
		for i, header := range data.CustomHeaders {
			customHeaders[i] = client.CustomHeader{
				Name:  header.Name.ValueString(),
				Value: header.Value.ValueString(),
			}
		}
		request.CustomHeaders = &customHeaders
	}

	if len(data.DREntities) > 0 {
		drEntities := make([]client.DREntity, len(data.DREntities))
		for i, entity := range data.DREntities {
			drEntities[i] = client.DREntity{
				ID:   entity.ID.ValueString(),
				Name: entity.Name.ValueString(),
			}
		}
		request.DREntities = &drEntities
	}

	traceAPICall("CreateNotificationChannel")
	createResp, err := r.provider.service.CreateNotificationChannel(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Notification Channel", err.Error())
		return
	}
	data.ID = types.StringValue(createResp.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NotificationChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NotificationChannelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetNotificationChannel")
	notificationChannel, err := r.provider.service.GetNotificationChannel(
		ctx,
		data.RelatedEntityType.ValueString(),
		data.RelatedEntityID.ValueString(),
		data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Notification Channel not found",
				fmt.Sprintf("Notification Channel with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Notification Channel with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.Name = types.StringValue(notificationChannel.Name)
	data.ChannelType = types.StringValue(notificationChannel.ChannelType)
	data.RelatedEntityID = types.StringValue(notificationChannel.RelatedEntityID)
	data.RelatedEntityType = types.StringValue(notificationChannel.RelatedEntityType)
	data.ContentType = types.StringPointerValue(notificationChannel.ContentType)
	data.LanguageCode = types.StringPointerValue(notificationChannel.LanguageCode)
	data.EmailAddress = types.StringPointerValue(notificationChannel.EmailAddress)
	data.PayloadUrl = types.StringPointerValue(notificationChannel.PayloadUrl)
	data.SecretToken = types.StringPointerValue(notificationChannel.SecretToken)
	data.ValidateSsl = types.BoolPointerValue(notificationChannel.ValidateSsl)
	if notificationChannel.CustomHeaders != nil {
		data.CustomHeaders = make([]CustomHeader, len(*notificationChannel.CustomHeaders))
		for i, header := range *notificationChannel.CustomHeaders {
			data.CustomHeaders[i] = CustomHeader{
				Name:  types.StringValue(header.Name),
				Value: types.StringValue(header.Value),
			}
		}
	}
	if notificationChannel.DREntities != nil {
		data.DREntities = make([]DREntity, len(*notificationChannel.DREntities))
		for i, entity := range *notificationChannel.DREntities {
			data.DREntities[i] = DREntity{
				ID:   types.StringValue(entity.ID),
				Name: types.StringValue(entity.Name),
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NotificationChannelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := &client.UpdateNotificationChannelRequest{
		Name:             StringValuePointerOptional(data.Name),
		ChannelType:      StringValuePointerOptional(data.ChannelType),
		ContentType:      StringValuePointerOptional(data.ContentType),
		PayloadUrl:       StringValuePointerOptional(data.PayloadUrl),
		SecretToken:      StringValuePointerOptional(data.SecretToken),
		ValidateSsl:      BoolValuePointerOptional(data.ValidateSsl),
		VerificationCode: StringValuePointerOptional(data.VerificationCode),
	}

	if len(data.CustomHeaders) > 0 {
		customHeaders := make([]client.CustomHeader, len(data.CustomHeaders))
		for i, header := range data.CustomHeaders {
			customHeaders[i] = client.CustomHeader{
				Name:  header.Name.ValueString(),
				Value: header.Value.ValueString(),
			}
		}
		request.CustomHeaders = &customHeaders
	}

	if len(data.DREntities) > 0 {
		drEntities := make([]client.DREntity, len(data.DREntities))
		for i, entity := range data.DREntities {
			drEntities[i] = client.DREntity{
				ID:   entity.ID.ValueString(),
				Name: entity.Name.ValueString(),
			}
		}
		request.DREntities = &drEntities
	}

	traceAPICall("UpdateNotificationChannel")
	_, err := r.provider.service.UpdateNotificationChannel(
		ctx,
		data.RelatedEntityType.ValueString(),
		data.RelatedEntityID.ValueString(),
		data.ID.ValueString(),
		request)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Notification Channel not found",
				fmt.Sprintf("Notification Channel with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error updating Notification Channel", err.Error())
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NotificationChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NotificationChannelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteNotificationChannel")
	err := r.provider.service.DeleteNotificationChannel(
		ctx,
		data.RelatedEntityType.ValueString(),
		data.RelatedEntityID.ValueString(),
		data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error getting Notification Channel info", err.Error())
			return
		}
	}
}

func (r *NotificationChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
