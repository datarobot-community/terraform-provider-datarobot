package client

type UserInfo struct {
	UID         string          `json:"uid"`
	Email       string          `json:"email"`
	FirstName   string          `json:"firstName"`
	LastName    string          `json:"lastName"`
	OrgID       string          `json:"orgId"`
	TenantID    string          `json:"tenantId"`
	Permissions map[string]bool `json:"permissions"`
}

type Entitlement struct {
	Name string `json:"name"`
}

type EntitlementEvaluation struct {
	Name  string `json:"name"`
	Value bool   `json:"value"`
}

type EvaluateEntitlementsRequest struct {
	Entitlements []Entitlement `json:"entitlements"`
}

type EvaluateEntitlementsResponse struct {
	Entitlements []EntitlementEvaluation `json:"entitlements"`
}
