package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ShareProtocol defines the network file sharing protocol.
// +kubebuilder:validation:Enum=SMB;NFS
type ShareProtocol string

const (
	ShareProtocolSMB ShareProtocol = "SMB"
	ShareProtocolNFS ShareProtocol = "NFS"
)

// SharePermission defines the access level for a share subject.
// +kubebuilder:validation:Enum=read;write;admin
type SharePermission string

const (
	SharePermissionRead  SharePermission = "read"
	SharePermissionWrite SharePermission = "write"
	SharePermissionAdmin SharePermission = "admin"
)

// ShareSubjectKind identifies the type of Kubernetes identity.
// +kubebuilder:validation:Enum=User;Group
type ShareSubjectKind string

const (
	ShareSubjectKindUser  ShareSubjectKind = "User"
	ShareSubjectKindGroup ShareSubjectKind = "Group"
)

// ShareSubject maps a Kubernetes user or group to share permissions.
type ShareSubject struct {
	// Kind is User or Group.
	Kind ShareSubjectKind `json:"kind"`

	// Name is the Kubernetes user/group name.
	Name string `json:"name"`

	// Permissions is the set of allowed operations.
	// +kubebuilder:validation:MinItems=1
	Permissions []SharePermission `json:"permissions"`
}

// ShareAuthz configures authorization for share access.
type ShareAuthz struct {
	// Mode defines the authorization model. Currently only kubernetes-rbac is supported.
	// +kubebuilder:default=kubernetes-rbac
	// +kubebuilder:validation:Enum=kubernetes-rbac
	Mode string `json:"mode"`

	// Subjects lists the users and groups with access grants.
	// +optional
	Subjects []ShareSubject `json:"subjects,omitempty"`
}

// SMBConfig holds Samba-specific share configuration.
type SMBConfig struct {
	// Browseable controls whether this share appears in browse lists.
	// +kubebuilder:default=true
	Browseable bool `json:"browseable,omitempty"`

	// ReadOnly forces read-only access regardless of authz grants.
	// +kubebuilder:default=false
	ReadOnly bool `json:"readOnly,omitempty"`

	// GuestOk allows unauthenticated access.
	// +kubebuilder:default=false
	GuestOk bool `json:"guestOk,omitempty"`
}

// NFSConfig holds NFS-specific share configuration.
type NFSConfig struct {
	// AllowedCIDRs restricts NFS access to specified CIDR ranges.
	// +optional
	AllowedCIDRs []string `json:"allowedCIDRs,omitempty"`

	// ReadOnly forces read-only exports.
	// +kubebuilder:default=false
	ReadOnly bool `json:"readOnly,omitempty"`

	// SquashMode controls UID/GID squashing (none, root_squash, all_squash).
	// +kubebuilder:default=root_squash
	// +kubebuilder:validation:Enum=none;root_squash;all_squash
	SquashMode string `json:"squashMode,omitempty"`

	// AnonUID is the UID to map anonymous/squashed users to.
	// +optional
	AnonUID int `json:"anonUID,omitempty"`

	// AnonGID is the GID to map anonymous/squashed groups to.
	// +optional
	AnonGID int `json:"anonGID,omitempty"`
}

// ShareSpec defines the desired state of a Share.
type ShareSpec struct {
	// PoolRef is the name of the Pool CR backing this share.
	// +kubebuilder:validation:Required
	PoolRef string `json:"poolRef"`

	// Path is the sub-path under the pool mount point to expose.
	// +kubebuilder:validation:Required
	Path string `json:"path"`

	// Protocol is the network sharing protocol.
	// +kubebuilder:validation:Required
	Protocol ShareProtocol `json:"protocol"`

	// Authz defines Kubernetes-native authorization for share access.
	// +optional
	Authz *ShareAuthz `json:"authz,omitempty"`

	// SMB holds Samba-specific config. Used when Protocol=SMB.
	// +optional
	SMB *SMBConfig `json:"smb,omitempty"`

	// NFS holds NFS-specific config. Used when Protocol=NFS.
	// +optional
	NFS *NFSConfig `json:"nfs,omitempty"`
}

// ShareStatus defines the observed state of a Share.
type ShareStatus struct {
	// Phase is the current lifecycle phase.
	Phase string `json:"phase,omitempty"`

	// Published indicates whether the share is actively serving.
	Published bool `json:"published,omitempty"`

	// EffectiveSubjects lists the resolved subjects with active grants.
	EffectiveSubjects []string `json:"effectiveSubjects,omitempty"`

	// PodName is the share pod managing this export.
	PodName string `json:"podName,omitempty"`

	// ServiceName is the Kubernetes Service fronting this share pod.
	ServiceName string `json:"serviceName,omitempty"`

	// Conditions provides detailed status conditions.
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Pool",type=string,JSONPath=`.spec.poolRef`
// +kubebuilder:printcolumn:name="Protocol",type=string,JSONPath=`.spec.protocol`
// +kubebuilder:printcolumn:name="Path",type=string,JSONPath=`.spec.path`
// +kubebuilder:printcolumn:name="Published",type=boolean,JSONPath=`.status.published`

// Share exposes a pool sub-path as an SMB or NFS network share.
type Share struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ShareSpec   `json:"spec,omitempty"`
	Status ShareStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ShareList contains a list of Share.
type ShareList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Share `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Share{}, &ShareList{})
}
