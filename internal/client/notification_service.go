package client

type CreateNotificationPolicyRequest struct {
	Name              string  `json:"name"`
	ChannelID         string  `json:"channelId"`
	ChannelScope      string  `json:"channelScope"`
	RelatedEntityID   string  `json:"relatedEntityId"`
	RelatedEntityType string  `json:"relatedEntityType"`
	Active            bool    `json:"active"`
	EventGroup        *string `json:"eventGroup,omitempty"`
	EventType         *string `json:"eventType,omitempty"`
	MaximalFrequency  *string `json:"maximalFrequency,omitempty"`
}

type UpdateNotificationPolicyRequest struct {
	Name             *string `json:"name,omitempty"`
	ChannelID        *string `json:"channelId,omitempty"`
	ChannelScope     *string `json:"channelScope,omitempty"`
	EventGroup       *string `json:"eventGroup,omitempty"`
	EventType        *string `json:"eventType,omitempty"`
	Active           *bool   `json:"active,omitempty"`
	MaximalFrequency *string `json:"maximalFrequency,omitempty"`
}

type NotificationPolicy struct {
	ID                string               `json:"id"`
	Name              string               `json:"name"`
	ChannelID         string               `json:"channelId"`
	ChannelScope      string               `json:"channelScope"`
	RelatedEntityID   string               `json:"relatedEntityId"`
	RelatedEntityType string               `json:"relatedEntityType"`
	Active            bool                 `json:"active"`
	EventGroup        *EventType           `json:"eventGroup,omitempty"`
	EventType         *EventType           `json:"eventType,omitempty"`
	Channel           *NotificationChannel `json:"channel,omitempty"`
	MaximalFrequency  *string              `json:"maximalFrequency,omitempty"`
}

type EventType struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type CreateNotificationChannelRequest struct {
	Name              string          `json:"name"`
	ChannelType       string          `json:"channelType"`
	ContentType       *string         `json:"contentType,omitempty"`
	RelatedEntityID   string          `json:"relatedEntityId"`
	RelatedEntityType string          `json:"relatedEntityType"`
	CustomHeaders     *[]CustomHeader `json:"customHeaders,omitempty"`
	DREntities        *[]DREntity     `json:"drEntities,omitempty"`
	EmailAddress      *string         `json:"emailAddress,omitempty"`
	LanguageCode      *string         `json:"languageCode,omitempty"`
	PayloadUrl        *string         `json:"payloadUrl,omitempty"`
	SecretToken       *string         `json:"secretToken,omitempty"`
	ValidateSsl       *bool           `json:"validateSsl,omitempty"`
	VerificationCode  *string         `json:"verificationCode,omitempty"`
}

type CustomHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DREntity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UpdateNotificationChannelRequest struct {
	Name             *string         `json:"name,omitempty"`
	ChannelType      *string         `json:"channelType,omitempty"`
	ContentType      *string         `json:"contentType,omitempty"`
	CustomHeaders    *[]CustomHeader `json:"customHeaders,omitempty"`
	DREntities       *[]DREntity     `json:"drEntities,omitempty"`
	EmailAddress     *string         `json:"emailAddress,omitempty"`
	LanguageCode     *string         `json:"languageCode,omitempty"`
	PayloadUrl       *string         `json:"payloadUrl,omitempty"`
	SecretToken      *string         `json:"secretToken,omitempty"`
	ValidateSsl      *bool           `json:"validateSsl,omitempty"`
	VerificationCode *string         `json:"verificationCode,omitempty"`
}

type NotificationChannel struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	ChannelType       string          `json:"channelType"`
	ContentType       *string         `json:"contentType,omitempty"`
	RelatedEntityID   string          `json:"relatedEntityId"`
	RelatedEntityType string          `json:"relatedEntityType"`
	CustomHeaders     *[]CustomHeader `json:"customHeaders,omitempty"`
	DREntities        *[]DREntity     `json:"drEntities,omitempty"`
	EmailAddress      *string         `json:"emailAddress,omitempty"`
	LanguageCode      *string         `json:"languageCode,omitempty"`
	PayloadUrl        *string         `json:"payloadUrl,omitempty"`
	SecretToken       *string         `json:"secretToken,omitempty"`
	ValidateSsl       *bool           `json:"validateSsl,omitempty"`
}
