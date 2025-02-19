package client

type UserInfo struct {
	UID       string `json:"uid"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	OrgID     string `json:"orgId"`
	TenantID  string `json:"tenantId"`
}
