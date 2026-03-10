package client

type UserMCPPromptMetadataRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type UserMCPPromptMetadataResponse struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Type               string `json:"type"`
	CreatedAt          string `json:"createdAt"`
	UserId             string `json:"userId"`
	UserName           string `json:"userName"`
	MCPServerVersionID string `json:"mcpServerVersionId"`
}
