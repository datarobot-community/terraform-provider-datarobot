package client

// SharedRoleGrant is a single role assignment sent to a sharedRoles endpoint.
// Role accepts the values in ExternalSharingRoles, plus NO_ROLE to revoke an
// existing grant.
type SharedRoleGrant struct {
	ShareRecipientType string `json:"shareRecipientType"`
	ID                 string `json:"id"`
	Role               string `json:"role"`
}

// UpdateSharedRolesRequest is the body for a sharedRoles PATCH. Operation only
// accepts updateRoles, and it has set rather than append semantics, so
// re-sending an unchanged role is a no-op.
type UpdateSharedRolesRequest struct {
	Operation string            `json:"operation"`
	Roles     []SharedRoleGrant `json:"roles"`
}

// SharedRole is a single grant returned by a sharedRoles GET.
type SharedRole struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	ShareRecipientType string  `json:"shareRecipientType"`
	Role               string  `json:"role"`
	ScimManaged        *bool   `json:"scimManaged,omitempty"`
	ScimIdpName        *string `json:"scimIdpName,omitempty"`
}

// ListSharedRolesResponse is the paginated envelope returned by a
// sharedRoles GET.
type ListSharedRolesResponse struct {
	Count      int          `json:"count"`
	TotalCount int          `json:"totalCount"`
	Next       *string      `json:"next"`
	Previous   *string      `json:"previous"`
	Data       []SharedRole `json:"data"`
}

const (
	// ShareRecipientTypeGroup is the recipient type for a directory group.
	ShareRecipientTypeGroup = "group"

	// SharedRolesOperationUpdate is the only operation a sharedRoles PATCH accepts.
	SharedRolesOperationUpdate = "updateRoles"

	// SharedRoleNone revokes an existing grant. The API converts it to a null
	// role internally, which is the documented removal path.
	SharedRoleNone = "NO_ROLE"
)
