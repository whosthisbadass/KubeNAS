package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ─────────────────────────────────────────
// ParitySchedule
// ─────────────────────────────────────────

// ParityScheduleSpec defines SnapRAID sync/check/scrub cron schedules.
type ParityScheduleSpec struct {
	// ArrayRef is the name of the Array to run parity operations against.
	// +kubebuilder:validation:Required
	ArrayRef string `json:"arrayRef"`

	// SyncCron is the cron expression for parity sync (daily recommended).
	// +kubebuilder:validation:Required
	SyncCron string `json:"syncCron"`

	// CheckCron is the cron expression for parity check (weekly recommended).
	// +kubebuilder:validation:Required
	CheckCron string `json:"checkCron"`

	// ScrubCron is the cron expression for full parity scrub (monthly recommended).
	// +kubebuilder:validation:Required
	ScrubCron string `json:"scrubCron"`

	// ScrubPercentage is the percentage of data scrubbed per scrub run.
	// +optional
	// +kubebuilder:default=100
	ScrubPercentage int `json:"scrubPercentage,omitempty"`
}

// ParityScheduleStatus reflects the last execution times for parity operations.
type ParityScheduleStatus struct {
	LastSyncTime   *metav1.Time `json:"lastSyncTime,omitempty"`
	LastCheckTime  *metav1.Time `json:"lastCheckTime,omitempty"`
	LastScrubTime  *metav1.Time `json:"lastScrubTime,omitempty"`
	LastSyncResult string       `json:"lastSyncResult,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced

// ParitySchedule schedules SnapRAID parity sync, check, and scrub operations.
type ParitySchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ParityScheduleSpec   `json:"spec,omitempty"`
	Status            ParityScheduleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ParityScheduleList contains a list of ParitySchedule.
type ParityScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ParitySchedule `json:"items"`
}

// ─────────────────────────────────────────
// PlacementPolicy
// ─────────────────────────────────────────

// PlacementStrategy defines the disk selection algorithm.
// +kubebuilder:validation:Enum=balanced;least-used;fill-first;tiered
type PlacementStrategy string

const (
	PlacementStrategyBalanced  PlacementStrategy = "balanced"
	PlacementStrategyLeastUsed PlacementStrategy = "least-used"
	PlacementStrategyFillFirst PlacementStrategy = "fill-first"
	PlacementStrategyTiered    PlacementStrategy = "tiered"
)

// PlacementWeights configures the scoring weights for the balanced strategy.
type PlacementWeights struct {
	// FreeSpace is the weight applied to free space ratio (0.0-1.0).
	// +kubebuilder:default=0.7
	FreeSpace float64 `json:"freeSpace,omitempty"`

	// Load is the weight applied to inverse disk I/O load (0.0-1.0).
	// +kubebuilder:default=0.2
	Load float64 `json:"load,omitempty"`

	// Health is the weight applied to SMART health score (0.0-1.0).
	// +kubebuilder:default=0.1
	Health float64 `json:"health,omitempty"`
}

// PlacementPolicySpec defines the desired state of a PlacementPolicy.
type PlacementPolicySpec struct {
	// Strategy selects the placement algorithm.
	Strategy PlacementStrategy `json:"strategy"`

	// Weights configures the balanced strategy scoring formula.
	// +optional
	Weights *PlacementWeights `json:"weights,omitempty"`

	// MinFreeSpace is the minimum free space a disk must have to be eligible.
	// +optional
	// +kubebuilder:default="20Gi"
	MinFreeSpace string `json:"minFreeSpace,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced

// PlacementPolicy controls how new file writes are routed across disks.
type PlacementPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PlacementPolicySpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// PlacementPolicyList contains a list of PlacementPolicy.
type PlacementPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlacementPolicy `json:"items"`
}

// ─────────────────────────────────────────
// RebalanceJob
// ─────────────────────────────────────────

// RebalanceJobPhase describes the lifecycle phase of a RebalanceJob.
// +kubebuilder:validation:Enum=Pending;Planning;Running;Completed;Failed
type RebalanceJobPhase string

const (
	RebalanceJobPhasePending   RebalanceJobPhase = "Pending"
	RebalanceJobPhasePlanning  RebalanceJobPhase = "Planning"
	RebalanceJobPhaseRunning   RebalanceJobPhase = "Running"
	RebalanceJobPhaseCompleted RebalanceJobPhase = "Completed"
	RebalanceJobPhaseFailed    RebalanceJobPhase = "Failed"
)

// RebalanceJobSpec defines a rebalance operation across pool disks.
type RebalanceJobSpec struct {
	// PoolRef is the name of the Pool to rebalance.
	// +kubebuilder:validation:Required
	PoolRef string `json:"poolRef"`

	// PlacementPolicyRef is the PlacementPolicy to enforce during rebalance.
	// +kubebuilder:validation:Required
	PlacementPolicyRef string `json:"placementPolicyRef"`

	// MaxConcurrentMoves limits how many file moves run in parallel.
	// +optional
	// +kubebuilder:default=2
	MaxConcurrentMoves int `json:"maxConcurrentMoves,omitempty"`

	// DryRun plans the rebalance without executing moves.
	// +optional
	// +kubebuilder:default=false
	DryRun bool `json:"dryRun,omitempty"`

	// ImbalanceThresholdPercent triggers rebalancing when disk utilization
	// variance exceeds this percentage.
	// +optional
	// +kubebuilder:default=20
	ImbalanceThresholdPercent int `json:"imbalanceThresholdPercent,omitempty"`
}

// RebalanceJobStatus reflects the observed state of a RebalanceJob.
type RebalanceJobStatus struct {
	Phase          RebalanceJobPhase `json:"phase,omitempty"`
	PlannedMoves   int               `json:"plannedMoves,omitempty"`
	CompletedMoves int               `json:"completedMoves,omitempty"`
	BytesMoved     int64             `json:"bytesMoved,omitempty"`
	StartTime      *metav1.Time      `json:"startTime,omitempty"`
	CompletionTime *metav1.Time      `json:"completionTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Pool",type=string,JSONPath=`.spec.poolRef`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Moves",type=integer,JSONPath=`.status.completedMoves`

// RebalanceJob executes a data rebalancing operation across pool disks.
type RebalanceJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RebalanceJobSpec   `json:"spec,omitempty"`
	Status            RebalanceJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RebalanceJobList contains a list of RebalanceJob.
type RebalanceJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RebalanceJob `json:"items"`
}

// ─────────────────────────────────────────
// DiskFailure
// ─────────────────────────────────────────

// DiskFailureSeverity classifies the urgency of a disk failure.
// +kubebuilder:validation:Enum=warning;critical
type DiskFailureSeverity string

const (
	DiskFailureSeverityWarning  DiskFailureSeverity = "warning"
	DiskFailureSeverityCritical DiskFailureSeverity = "critical"
)

// DiskFailurePhase describes the lifecycle of a DiskFailure.
// +kubebuilder:validation:Enum=Detected;Acknowledged;Rebuilding;Resolved
type DiskFailurePhase string

const (
	DiskFailurePhaseDetected     DiskFailurePhase = "Detected"
	DiskFailurePhaseAcknowledged DiskFailurePhase = "Acknowledged"
	DiskFailurePhaseRebuilding   DiskFailurePhase = "Rebuilding"
	DiskFailurePhaseResolved     DiskFailurePhase = "Resolved"
)

// DiskFailureSpec describes a detected or predicted disk failure.
type DiskFailureSpec struct {
	// DiskRef is the name of the failed Disk CR.
	// +kubebuilder:validation:Required
	DiskRef string `json:"diskRef"`

	// Severity indicates urgency level.
	// +kubebuilder:validation:Required
	Severity DiskFailureSeverity `json:"severity"`

	// Reason is a machine-readable cause code (e.g., SMART_PREFAIL, IO_ERROR).
	// +kubebuilder:validation:Required
	Reason string `json:"reason"`

	// RecommendedAction describes the advised operator action.
	// +kubebuilder:validation:Required
	RecommendedAction string `json:"recommendedAction"`

	// ReplacementDiskRef is set once a replacement disk is provisioned.
	// +optional
	ReplacementDiskRef string `json:"replacementDiskRef,omitempty"`
}

// DiskFailureStatus reflects the current handling state.
type DiskFailureStatus struct {
	Phase      DiskFailurePhase `json:"phase,omitempty"`
	ResolvedAt *metav1.Time     `json:"resolvedAt,omitempty"`
	Message    string           `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Disk",type=string,JSONPath=`.spec.diskRef`
// +kubebuilder:printcolumn:name="Severity",type=string,JSONPath=`.spec.severity`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// DiskFailure tracks the detection and remediation of a disk failure event.
type DiskFailure struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DiskFailureSpec   `json:"spec,omitempty"`
	Status            DiskFailureStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DiskFailureList contains a list of DiskFailure.
type DiskFailureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DiskFailure `json:"items"`
}

// ─────────────────────────────────────────
// CachePool
// ─────────────────────────────────────────

// CacheMode defines how writes flow through the SSD cache.
// +kubebuilder:validation:Enum=write-back;write-through
type CacheMode string

const (
	CacheModeWriteBack    CacheMode = "write-back"
	CacheModeWriteThrough CacheMode = "write-through"
)

// FlushPolicy defines when dirty cache data is flushed to HDD.
type FlushPolicy struct {
	// MaxDirtyAge is the maximum age of dirty data before forced flush (e.g., "30m").
	// +optional
	MaxDirtyAge string `json:"maxDirtyAge,omitempty"`

	// HighWatermarkPercent triggers a flush when cache utilization exceeds this percent.
	// +optional
	// +kubebuilder:default=80
	HighWatermarkPercent int `json:"highWatermarkPercent,omitempty"`
}

// CachePoolSpec defines the desired state of a CachePool.
type CachePoolSpec struct {
	// PoolRef is the Pool that this cache accelerates.
	// +kubebuilder:validation:Required
	PoolRef string `json:"poolRef"`

	// CacheDisks is the list of Disk CRs serving as cache.
	// +kubebuilder:validation:MinItems=1
	CacheDisks []string `json:"cacheDisks"`

	// Mode controls write caching behavior.
	// +kubebuilder:validation:Required
	Mode CacheMode `json:"mode"`

	// FlushPolicy configures dirty data eviction.
	// +optional
	FlushPolicy *FlushPolicy `json:"flushPolicy,omitempty"`
}

// CachePoolStatus reflects the observed state of a CachePool.
type CachePoolStatus struct {
	Phase              string  `json:"phase,omitempty"`
	CacheUsedBytes     int64   `json:"cacheUsedBytes,omitempty"`
	CacheCapacityBytes int64   `json:"cacheCapacityBytes,omitempty"`
	DirtyPercent       float64 `json:"dirtyPercent,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced

// CachePool attaches an SSD/NVMe cache tier to an existing Pool.
type CachePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CachePoolSpec   `json:"spec,omitempty"`
	Status            CachePoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CachePoolList contains a list of CachePool.
type CachePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CachePool `json:"items"`
}

func init() {
	SchemeBuilder.Register(
		&ParitySchedule{}, &ParityScheduleList{},
		&PlacementPolicy{}, &PlacementPolicyList{},
		&RebalanceJob{}, &RebalanceJobList{},
		&DiskFailure{}, &DiskFailureList{},
		&CachePool{}, &CachePoolList{},
	)
}
