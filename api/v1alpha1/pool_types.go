package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MergerfsConfig holds mergerfs-specific pool options.
type MergerfsConfig struct {
	// CategoryCreate sets the creation policy for mergerfs (e.g., epmfs, mfs, lfs).
	// +kubebuilder:default=epmfs
	CategoryCreate string `json:"categoryCreate,omitempty"`

	// MinFreeSpace is the minimum free space required on a branch before mergerfs avoids writing to it.
	// +kubebuilder:default="20Gi"
	MinFreeSpace string `json:"minFreeSpace,omitempty"`

	// ExtraOptions provides additional raw mergerfs mount options.
	// +optional
	ExtraOptions string `json:"extraOptions,omitempty"`
}

// PoolSpec defines the desired state of a Pool.
type PoolSpec struct {
	// ArrayRef is the name of the Array CR to pool.
	// +kubebuilder:validation:Required
	ArrayRef string `json:"arrayRef"`

	// MountPoint is the host path where the merged pool will be mounted.
	// +kubebuilder:validation:Required
	// +kubebuilder:default="/mnt/pool"
	MountPoint string `json:"mountPoint"`

	// Mergerfs configures mergerfs pool behavior.
	// +optional
	Mergerfs *MergerfsConfig `json:"mergerfs,omitempty"`
}

// PoolStatus defines the observed state of a Pool.
type PoolStatus struct {
	// Phase is the current lifecycle phase.
	Phase string `json:"phase,omitempty"`

	// Mounted indicates whether the mergerfs pool is actively mounted.
	Mounted bool `json:"mounted,omitempty"`

	// TotalBytes is the combined capacity of all member disks.
	TotalBytes int64 `json:"totalBytes,omitempty"`

	// AvailableBytes is the total free space across all member disks.
	AvailableBytes int64 `json:"availableBytes,omitempty"`

	// MemberDisks lists the Disk CRs currently in the pool.
	MemberDisks []string `json:"memberDisks,omitempty"`

	// Conditions provides detailed status conditions.
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="ArrayRef",type=string,JSONPath=`.spec.arrayRef`
// +kubebuilder:printcolumn:name="MountPoint",type=string,JSONPath=`.spec.mountPoint`
// +kubebuilder:printcolumn:name="Mounted",type=boolean,JSONPath=`.status.mounted`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// Pool creates a mergerfs union filesystem pool from an Array's data disks.
type Pool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PoolSpec   `json:"spec,omitempty"`
	Status PoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PoolList contains a list of Pool.
type PoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Pool{}, &PoolList{})
}
