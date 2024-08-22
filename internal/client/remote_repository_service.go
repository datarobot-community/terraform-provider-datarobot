package client

type CreateRemoteRepositoryRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Location     string `json:"location"`
	SourceType   string `json:"sourceType"`
	CredentialID string `json:"credentialId,omitempty"`
}

type UpdateRemoteRepositoryRequest struct {
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	Location     string `json:"location,omitempty"`
	CredentialID string `json:"credentialId,omitempty"`
}

type RemoteRepositoryResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Location     string `json:"location"`
	SourceType   string `json:"sourceType"`
	CredentialID string `json:"credentialId"`
}
