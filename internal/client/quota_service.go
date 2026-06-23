package client

import (
	"context"
)

// QuotaRule is a single rate-limit rule within a quota's default rules, e.g.
// {"rule": "requests", "limit": 750, "window": "day"}.
type QuotaRule struct {
	Rule   string `json:"rule"`
	Limit  int64  `json:"limit"`
	Window string `json:"window"`
}

// Quota governs the usage of a single DataRobot resource (e.g. a deployment).
// DataRobot models at most one quota per resource; defaultRules apply to every
// consumer (the docs site is unauthenticated, so there are no per-consumer
// policies).
type Quota struct {
	ID           string      `json:"id"`
	ResourceType string      `json:"resourceType"`
	ResourceID   string      `json:"resourceId"`
	DefaultRules []QuotaRule `json:"defaultRules"`
}

type CreateQuotaRequest struct {
	ResourceType string      `json:"resourceType"`
	ResourceID   string      `json:"resourceId"`
	DefaultRules []QuotaRule `json:"defaultRules"`
}

// UpdateQuotaRequest carries only the mutable field; resourceType / resourceId
// identify the quota and cannot change (they force a replacement instead).
type UpdateQuotaRequest struct {
	DefaultRules []QuotaRule `json:"defaultRules"`
}

// listQuotasResponse is the API-v2 list envelope returned by GET /quotas/.
type listQuotasResponse struct {
	Data []Quota `json:"data"`
}

func (s *ServiceImpl) CreateQuota(ctx context.Context, req *CreateQuotaRequest) (*Quota, error) {
	return Post[Quota](s.client, ctx, "/quotas/", req)
}

// GetQuotaForResource returns the single quota governing (resourceType, resourceID),
// or a NotFoundError if none exists. The lookup is by resource rather than by quota
// id so it survives the quota being recreated out of band and so `import` can key on
// the deployment id (mirrors set_quota.py's extract_existing).
func (s *ServiceImpl) GetQuotaForResource(ctx context.Context, resourceType, resourceID string) (*Quota, error) {
	resp, err := Get[listQuotasResponse](s.client, ctx, "/quotas/?resourceType="+resourceType+"&resourceId="+resourceID)
	if err != nil {
		return nil, err
	}
	for i := range resp.Data {
		if resp.Data[i].ResourceID == resourceID {
			return &resp.Data[i], nil
		}
	}
	return nil, NewNotFoundError("quota for " + resourceType + " " + resourceID)
}

func (s *ServiceImpl) UpdateQuota(ctx context.Context, id string, req *UpdateQuotaRequest) (*Quota, error) {
	return Patch[Quota](s.client, ctx, "/quotas/"+id+"/", req)
}

func (s *ServiceImpl) DeleteQuota(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/quotas/"+id+"/")
}
