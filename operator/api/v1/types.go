package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type DiskLifecycleState string

const (
	DiskStateDetected   DiskLifecycleState = "Detected"
	DiskStateUnmanaged  DiskLifecycleState = "Unmanaged"
	DiskStateUnassigned DiskLifecycleState = "Unassigned"
	DiskStateClaimed    DiskLifecycleState = "Claimed"
	DiskStateActive     DiskLifecycleState = "Active"
	DiskStateDegraded   DiskLifecycleState = "Degraded"
	DiskStateFailed     DiskLifecycleState = "Failed"
	DiskStateRebuilding DiskLifecycleState = "Rebuilding"
	DiskStateRecovered  DiskLifecycleState = "Recovered"
)

type DiskSpec struct {
	Device     string             `json:"device"`
	Size       string             `json:"size,omitempty"`
	Rotational bool               `json:"rotational,omitempty"`
	Filesystem string             `json:"filesystem,omitempty"`
	Labels     map[string]string  `json:"labels,omitempty"`
	Health     string             `json:"health,omitempty"`
	State      DiskLifecycleState `json:"state,omitempty"`
}

type DiskStatus struct {
	State  DiskLifecycleState `json:"state,omitempty"`
	Health string             `json:"health,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Device",type=string,JSONPath=`.spec.device`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`

type Disk struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DiskSpec   `json:"spec,omitempty"`
	Status DiskStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type DiskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Disk `json:"items"`
}

type DiskClaimSpec struct {
	Disk string `json:"disk"`
	Pool string `json:"pool"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
type DiskClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DiskClaimSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type DiskClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DiskClaim `json:"items"`
}

type UnassignedDiskSpec struct {
	Disk       string `json:"disk"`
	MountPoint string `json:"mountPoint"`
	Filesystem string `json:"filesystem,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type UnassignedDisk struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              UnassignedDiskSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type UnassignedDiskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UnassignedDisk `json:"items"`
}

type StoragePoolSpec struct {
	DiskSelector map[string]string `json:"diskSelector,omitempty"`
	ParityDisks  []string          `json:"parityDisks,omitempty"`
	SplitLevel   int               `json:"splitLevel,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
type StoragePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              StoragePoolSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type StoragePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StoragePool `json:"items"`
}

type FilesystemSpec struct {
	Pool       string `json:"pool"`
	Mountpoint string `json:"mountpoint"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type Filesystem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              FilesystemSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type FilesystemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Filesystem `json:"items"`
}

type ShareSpec struct {
	Filesystem string `json:"filesystem"`
	Path       string `json:"path"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type Share struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ShareSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type ShareList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Share `json:"items"`
}

type VolumeSpec struct {
	Share string `json:"share"`
	Path  string `json:"path"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type Volume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VolumeSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type VolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Volume `json:"items"`
}

type TierPolicySpec struct {
	SourcePool string `json:"sourcePool"`
	TargetPool string `json:"targetPool"`
	MoveAfter  string `json:"moveAfter"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type TierPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TierPolicySpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type TierPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TierPolicy `json:"items"`
}

// Safeguards

type DiskLockSpec struct {
	Disk   string `json:"disk"`
	Reason string `json:"reason,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type DiskLock struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DiskLockSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type DiskLockList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DiskLock `json:"items"`
}

type PoolHealthSpec struct {
	Pool string `json:"pool"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type PoolHealth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PoolHealthSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type PoolHealthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PoolHealth `json:"items"`
}

type FilesystemIntegritySpec struct {
	Filesystem string `json:"filesystem"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type FilesystemIntegrity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              FilesystemIntegritySpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type FilesystemIntegrityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FilesystemIntegrity `json:"items"`
}

func init() {
	SchemeBuilder.Register(
		&Disk{}, &DiskList{},
		&DiskClaim{}, &DiskClaimList{},
		&UnassignedDisk{}, &UnassignedDiskList{},
		&StoragePool{}, &StoragePoolList{},
		&Filesystem{}, &FilesystemList{},
		&Share{}, &ShareList{},
		&Volume{}, &VolumeList{},
		&TierPolicy{}, &TierPolicyList{},
		&DiskLock{}, &DiskLockList{},
		&PoolHealth{}, &PoolHealthList{},
		&FilesystemIntegrity{}, &FilesystemIntegrityList{},
	)
}
