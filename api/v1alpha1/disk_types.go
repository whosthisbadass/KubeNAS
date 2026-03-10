package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DiskRole defines the role of a disk within KubeNAS.
// +kubebuilder:validation:Enum=data;parity;cache
type DiskRole string

const (
	DiskRoleData   DiskRole = "data"
	DiskRoleParity DiskRole = "parity"
	DiskRoleCache  DiskRole = "cache"
)

// DiskPhase describes the lifecycle phase of a Disk.
// +kubebuilder:validation:Enum=Pending;Ready;Degraded;Failed
type DiskPhase string

const (
	DiskPhasePending  DiskPhase = "Pending"
	DiskPhaseReady    DiskPhase = "Ready"
	DiskPhaseDegraded DiskPhase = "Degraded"
	DiskPhaseFailed   DiskPhase = "Failed"
)

// DiskSpec defines the desired state of a Disk.
type DiskSpec struct {
	// NodeName is the Kubernetes node hosting this disk.
	// +kubebuilder:validation:Required
	NodeName string `json:"nodeName"`

	// DevicePath is the block device path, e.g. /dev/sdb.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^/dev/[a-z][a-z0-9/]*$`
	DevicePath string `json:"devicePath"`

	// Filesystem is the filesystem type to format/expect on this disk.
	// +kubebuilder:default=xfs
	Filesystem string `json:"filesystem"`

	// MountPoint is the path to mount the disk on the host.
	// +kubebuilder:validation:Required
	MountPoint string `json:"mountPoint"`

	// Role designates how this disk participates in an array.
	// +kubebuilder:validation:Required
	Role DiskRole `json:"role"`

	// SpinDown enables disk spin-down after idle period.
	// +optional
	SpinDown *SpinDownConfig `json:"spinDown,omitempty"`
}

// SpinDownConfig configures automatic disk spin-down behavior.
type SpinDownConfig struct {
	// Enabled controls whether spin-down is active.
	Enabled bool `json:"enabled"`
	// IdleMinutes is the idle time before spin-down occurs.
	// +kubebuilder:default=30
	IdleMinutes int `json:"idleMinutes,omitempty"`
}

// DiskStatus defines the observed state of a Disk.
type DiskStatus struct {
	// Phase is the lifecycle phase of the disk.
	Phase DiskPhase `json:"phase,omitempty"`

	// CapacityBytes is the raw capacity of the disk.
	CapacityBytes int64 `json:"capacityBytes,omitempty"`

	// AvailableBytes is the free space on the disk.
	AvailableBytes int64 `json:"availableBytes,omitempty"`

	// HealthScore is a normalized 0.0-1.0 health score from SMART data.
	HealthScore float64 `json:"healthScore,omitempty"`

	// SmartSummary is a brief SMART status summary.
	SmartSummary string `json:"smartSummary,omitempty"`

	// SerialNumber is the disk serial number from SMART.
	SerialNumber string `json:"serialNumber,omitempty"`

	// Model is the disk model string.
	Model string `json:"model,omitempty"`

	// Rotational indicates whether this is an HDD (true) or SSD (false).
	Rotational bool `json:"rotational,omitempty"`

	// Mounted indicates whether the disk is currently mounted.
	Mounted bool `json:"mounted,omitempty"`

	// LastSmartCheckTime is when the last SMART check ran.
	LastSmartCheckTime *metav1.Time `json:"lastSmartCheckTime,omitempty"`

	// Conditions represents detailed disk condition tracking.
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=kd
// +kubebuilder:printcolumn:name="Node",type=string,JSONPath=`.spec.nodeName`
// +kubebuilder:printcolumn:name="Device",type=string,JSONPath=`.spec.devicePath`
// +kubebuilder:printcolumn:name="Role",type=string,JSONPath=`.spec.role`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Health",type=number,JSONPath=`.status.healthScore`

// Disk represents a host block device managed by KubeNAS.
type Disk struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DiskSpec   `json:"spec,omitempty"`
	Status DiskStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DiskList contains a list of Disk.
type DiskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Disk `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Disk{}, &DiskList{})
}
