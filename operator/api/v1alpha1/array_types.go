package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ArrayPhase describes the lifecycle phase of an Array.
// +kubebuilder:validation:Enum=Pending;Ready;Degraded;Rebuilding;Failed
type ArrayPhase string

const (
	ArrayPhasePending   ArrayPhase = "Pending"
	ArrayPhaseReady     ArrayPhase = "Ready"
	ArrayPhaseDegraded  ArrayPhase = "Degraded"
	ArrayPhaseRebuilding ArrayPhase = "Rebuilding"
	ArrayPhaseFailed    ArrayPhase = "Failed"
)

// SnapraidConfig holds SnapRAID-specific configuration.
type SnapraidConfig struct {
	// ContentFiles is the list of paths where SnapRAID stores content files.
	// +listType=atomic
	ContentFiles []string `json:"contentFiles,omitempty"`

	// ExcludePatterns lists glob patterns to exclude from parity.
	// +optional
	ExcludePatterns []string `json:"excludePatterns,omitempty"`

	// AutoSyncOnChange triggers a sync after detected file changes.
	// +optional
	// +kubebuilder:default=false
	AutoSyncOnChange bool `json:"autoSyncOnChange,omitempty"`
}

// ArraySpec defines the desired state of an Array.
type ArraySpec struct {
	// DataDisks is the ordered list of Disk CR names for data storage.
	// +kubebuilder:validation:MinItems=1
	DataDisks []string `json:"dataDisks"`

	// ParityDisks is the ordered list of Disk CR names for parity.
	// +kubebuilder:validation:MinItems=1
	ParityDisks []string `json:"parityDisks"`

	// SnapraidConfig holds SnapRAID tuning parameters.
	// +optional
	SnapraidConfig *SnapraidConfig `json:"snapraidConfig,omitempty"`
}

// ArrayStatus defines the observed state of an Array.
type ArrayStatus struct {
	// Phase is the current lifecycle phase of the array.
	Phase ArrayPhase `json:"phase,omitempty"`

	// ParityHealthy indicates whether the parity is in sync and valid.
	ParityHealthy bool `json:"parityHealthy,omitempty"`

	// ParityDriftPercent tracks how far parity is from current state (0 = fully synced).
	ParityDriftPercent float64 `json:"parityDriftPercent,omitempty"`

	// DegradedDisks lists disk names currently in Degraded or Failed state.
	DegradedDisks []string `json:"degradedDisks,omitempty"`

	// LastSyncTime is when parity was last successfully synced.
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// LastCheckTime is when parity was last successfully checked.
	LastCheckTime *metav1.Time `json:"lastCheckTime,omitempty"`

	// Conditions provides detailed status conditions.
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="ParityHealthy",type=boolean,JSONPath=`.status.parityHealthy`
// +kubebuilder:printcolumn:name="LastSync",type=string,JSONPath=`.status.lastSyncTime`

// Array defines a NAS array composed of data and parity disks.
type Array struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArraySpec   `json:"spec,omitempty"`
	Status ArrayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ArrayList contains a list of Array.
type ArrayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Array `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Array{}, &ArrayList{})
}
