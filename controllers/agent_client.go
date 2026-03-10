package controllers

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeAgentClient defines the interface used by controllers to communicate
// with the KubeNAS Node Agent DaemonSet running on storage nodes.
type NodeAgentClient interface {
	GetDiskStatus(ctx context.Context, nodeName, devicePath string) (*DiskAgentStatus, error)
	MountDisk(ctx context.Context, nodeName string, req MountDiskRequest) error
	UnmountDisk(ctx context.Context, nodeName, mountPoint string) error
	ApplySnapraidConfig(ctx context.Context, nodeName string, cfg SnapraidConfig) error
	EnsureMergerFSMount(ctx context.Context, nodeName string, req MergerFSMountRequest) (bool, error)
	RunParityOperation(ctx context.Context, nodeName, operation string) error
	GetDiskIOStats(ctx context.Context, nodeName, devicePath string) (*DiskIOStats, error)
	WaitForOperation(ctx context.Context, operationID string) (*AgentOperationStatus, error)
}

type DiskAgentStatus struct {
	DevicePath     string
	CapacityBytes  int64
	AvailableBytes int64
	HealthScore    float64
	SmartSummary   string
	SerialNumber   string
	Model          string
	Rotational     bool
	Mounted        bool
	SmartFailed    bool
	IOErrors       int64
	LastChecked    metav1.Time
}

type DiskIOStats struct {
	ReadsCompleted  int64
	WritesCompleted int64
	ReadBytes       int64
	WriteBytes      int64
	IOInProgress    int64
	IOTimeMs        int64
}

type MountDiskRequest struct {
	DevicePath    string
	MountPoint    string
	Filesystem    string
	FormatIfEmpty bool
}

type MergerFSMountRequest struct {
	Branches       []string
	MountPoint     string
	CategoryCreate string
	MinFreeSpace   string
	ExtraOptions   string
}

type SnapraidConfig struct {
	ParityEntries   []SnapraidParityEntry
	DataEntries     []SnapraidDataEntry
	ContentFiles    []string
	ExcludePatterns []string
}

type SnapraidParityEntry struct {
	Index      int
	DevicePath string
	MountPoint string
}

type SnapraidDataEntry struct {
	Label      string
	DevicePath string
	MountPoint string
}

type AgentOperationStatus struct {
	OperationID string `json:"operationID"`
	State       string `json:"state"`
	Message     string `json:"message,omitempty"`
}

func setCondition(conditions *[]metav1.Condition, newCond metav1.Condition) {
	for i, c := range *conditions {
		if c.Type == newCond.Type {
			(*conditions)[i] = newCond
			return
		}
	}
	*conditions = append(*conditions, newCond)
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}
