package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// GlobalModelDataSourceModel describes the global model data source resource.
type GlobalModelDataSourceModel struct {
	Name      types.String `tfsdk:"name"`
	ID        types.String `tfsdk:"id"`
	VersionID types.String `tfsdk:"version_id"`
}



type NotificationChannelResourceModel struct {
	ID                types.String   `tfsdk:"id"`
	Name              types.String   `tfsdk:"name"`
	ChannelType       types.String   `tfsdk:"channel_type"`
	ContentType       types.String   `tfsdk:"content_type"`
	CustomHeaders     []CustomHeader `tfsdk:"custom_headers"`
	DREntities        []DREntity     `tfsdk:"dr_entities"`
	EmailAddress      types.String   `tfsdk:"email_address"`
	LanguageCode      types.String   `tfsdk:"language_code"`
	PayloadUrl        types.String   `tfsdk:"payload_url"`
	RelatedEntityID   types.String   `tfsdk:"related_entity_id"`
	RelatedEntityType types.String   `tfsdk:"related_entity_type"`
	SecretToken       types.String   `tfsdk:"secret_token"`
	ValidateSsl       types.Bool     `tfsdk:"validate_ssl"`
	VerificationCode  types.String   `tfsdk:"verification_code"`
}


type CustomHeader struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type DREntity struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}


type NotificationPolicyResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ChannelID         types.String `tfsdk:"channel_id"`
	ChannelScope      types.String `tfsdk:"channel_scope"`
	RelatedEntityID   types.String `tfsdk:"related_entity_id"`
	RelatedEntityType types.String `tfsdk:"related_entity_type"`
	Active            types.Bool   `tfsdk:"active"`
	EventGroup        types.String `tfsdk:"event_group"`
	EventType         types.String `tfsdk:"event_type"`
	MaximalFrequency  types.String `tfsdk:"maximal_frequency"`
}


type RegisteredModelFromLeaderboardResourceModel struct {
	ID                            types.String   `tfsdk:"id"`
	VersionID                     types.String   `tfsdk:"version_id"`
	VersionName                   types.String   `tfsdk:"version_name"`
	Name                          types.String   `tfsdk:"name"`
	Description                   types.String   `tfsdk:"description"`
	ModelID                       types.String   `tfsdk:"model_id"`
	PredictionThreshold           types.Float64  `tfsdk:"prediction_threshold"`
	ComputeAllTsIntervals         types.Bool     `tfsdk:"compute_all_ts_intervals"`
	DistributionPredictionModelID types.String   `tfsdk:"distribution_prediction_model_id"`
	UseCaseIDs                    []types.String `tfsdk:"use_case_ids"`
}


// RegisteredModelResourceModel describes the registered model resource.
type RegisteredModelResourceModel struct {
	ID                   types.String   `tfsdk:"id"`
	VersionID            types.String   `tfsdk:"version_id"`
	VersionName          types.String   `tfsdk:"version_name"`
	Name                 types.String   `tfsdk:"name"`
	Description          types.String   `tfsdk:"description"`
	CustomModelVersionId types.String   `tfsdk:"custom_model_version_id"`
	UseCaseIDs           []types.String `tfsdk:"use_case_ids"`
}
