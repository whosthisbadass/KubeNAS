package controllers

// SharePermission is the permission vocabulary for share access policy.
type SharePermission string

const (
	SharePermissionRead  SharePermission = "read"
	SharePermissionWrite SharePermission = "write"
	SharePermissionAdmin SharePermission = "admin"
)

// ShareSubject binds a Kubernetes user/group to share permissions.
type ShareSubject struct {
	Kind        string            `json:"kind"` // User or Group
	Name        string            `json:"name"`
	Permissions []SharePermission `json:"permissions"`
}
