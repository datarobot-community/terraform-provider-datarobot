package client

type UserMCPResourceMetadataRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Uri  string `json:"uri"`
}

type UserMCPResourceMetadataResponse struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Type               string `json:"type"`
	Uri                string `json:"uri"`
	CreatedAt          string `json:"createdAt"`
	UserId             string `json:"userId"`
	UserName           string `json:"userName"`
	MCPServerVersionID string `json:"mcpServerVersionId"`
}
