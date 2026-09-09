package client

// DirectoryEntity is a single entity returned by the Directory Search API.
type DirectoryEntity struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	EntityType         string  `json:"entityType"`
	Description        *string `json:"description"`
	ProvisioningSource string  `json:"provisioningSource"`
	MemberCount        *int    `json:"memberCount"`
	ExternalIdpGroupID *string `json:"externalIdpGroupId"`
	UpdatedAt          *string `json:"updatedAt"`
}

// ListDirectoryEntitiesRequest is the query for the Directory Search API. Name
// matching is exact and case sensitive.
type ListDirectoryEntitiesRequest struct {
	EntityType string `url:"entityType,omitempty"`
	Name       string `url:"name,omitempty"`
	Offset     int    `url:"offset,omitempty"`
	Limit      int    `url:"limit,omitempty"`
}

// ListDirectoryEntitiesResponse is the paginated envelope returned by the
// Directory Search API. TotalCount is the number of matches before paging, and
// is what callers should assert on rather than reading Data[0]: the endpoint
// returns every match rather than erroring when a name is ambiguous.
type ListDirectoryEntitiesResponse struct {
	Count      int               `json:"count"`
	TotalCount int               `json:"totalCount"`
	Next       *string           `json:"next"`
	Previous   *string           `json:"previous"`
	Data       []DirectoryEntity `json:"data"`
}
