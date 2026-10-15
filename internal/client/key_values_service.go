package client

import "context"

const (
	KeyValueCategoryTag = "tag"
	// KeyValueTypeString is the value type tags are written with. Only string tags are added to
	// an entity's searchable tags, which `GET /workloads/?tagKeys=` filters on.
	KeyValueTypeString = "string"
)

// KeyValue is an entry in the key-value store, which holds tags and other metadata of an
// entity outside the entity itself.
type KeyValue struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	Category string `json:"category"`
	// IsEditable is false for entries the platform writes, which it refuses to change or delete.
	IsEditable *bool `json:"isEditable,omitempty"`
}

// KeyValueRequest is the body of both create and update. An update takes every field as well:
// the platform requires them, and keeps the stored value of any field an update sends empty.
type KeyValueRequest struct {
	EntityID   string `json:"entityId"`
	EntityType string `json:"entityType"`
	Category   string `json:"category"`
	Name       string `json:"name"`
	Value      string `json:"value"`
	ValueType  string `json:"valueType"`
}

type CreateKeyValueResponse struct {
	ID string `json:"id"`
}

type ListKeyValuesRequest struct {
	EntityType string `url:"entityType"`
	EntityID   string `url:"entityId"`
	Category   string `url:"category,omitempty"`
}

// ListKeyValues returns every entry of an entity; the listing's `next` link keeps the filters.
func (s *ServiceImpl) ListKeyValues(ctx context.Context, req *ListKeyValuesRequest) ([]KeyValue, error) {
	return GetAllPages[KeyValue](s.client, ctx, "/keyValues/", req)
}

func (s *ServiceImpl) CreateKeyValue(ctx context.Context, req *KeyValueRequest) (*CreateKeyValueResponse, error) {
	return Post[CreateKeyValueResponse](s.client, ctx, "/keyValues/", req)
}

func (s *ServiceImpl) UpdateKeyValue(ctx context.Context, id string, req *KeyValueRequest) (*KeyValue, error) {
	return Patch[KeyValue](s.client, ctx, "/keyValues/"+id+"/", req)
}

func (s *ServiceImpl) DeleteKeyValue(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/keyValues/"+id+"/")
}
