package client

type CreateAppOAuthProviderRequest struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type UpdateAppOAuthProviderRequest struct {
	Name         string `json:"name,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
}

type AppOAuthProviderResponse struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	OrgID          string                 `json:"orgId"`
	Type           string                 `json:"type"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	ClientID       string                 `json:"clientId"`
	SecureConfigID string                 `json:"secureConfigId"`
	Status         string                 `json:"status"`
	CreatedAt      string                 `json:"createdAt"`
	UpdatedAt      string                 `json:"updatedAt"`
}
