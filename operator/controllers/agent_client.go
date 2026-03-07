package controllers

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ─────────────────────────────────────────
// Node Agent Client Interface
// ─────────────────────────────────────────

// NodeAgentClient defines the interface used by controllers to communicate
// with the KubeNAS Node Agent DaemonSet running on storage nodes.
// The production implementation communicates via Kubernetes API (ConfigMaps/Secrets
// as operation channels) or gRPC. The interface enables clean unit testing via mocks.
type NodeAgentClient interface {
	// GetDiskStatus returns live hardware state for a disk on a given node.
	GetDiskStatus(ctx context.Context, nodeName, devicePath string) (*DiskAgentStatus, error)

	// MountDisk instructs the node agent to mount a disk at the given path.
	MountDisk(ctx context.Context, nodeName string, req MountDiskRequest) error

	// UnmountDisk instructs the node agent to unmount a disk.
	UnmountDisk(ctx context.Context, nodeName, mountPoint string) error

	// ApplySnapraidConfig pushes the generated snapraid.conf to the node agent.
	ApplySnapraidConfig(ctx context.Context, nodeName string, cfg SnapraidConfig) error

	// EnsureMergerFSMount creates or updates the mergerfs pool mount on the node.
	EnsureMergerFSMount(ctx context.Context, nodeName string, req MergerFSMountRequest) (bool, error)

	// RunParityOperation triggers a synchronous parity operation (sync/check/scrub).
	RunParityOperation(ctx context.Context, nodeName, operation string) error

	// GetDiskIOStats returns recent I/O statistics for a device.
	GetDiskIOStats(ctx context.Context, nodeName, devicePath string) (*DiskIOStats, error)
}

// ─────────────────────────────────────────
// Agent Request / Response Types
// ─────────────────────────────────────────

// DiskAgentStatus is the live hardware status returned by the Node Agent.
type DiskAgentStatus struct {
	DevicePath     string
	CapacityBytes  int64
	AvailableBytes int64
	HealthScore    float64 // 0.0 (critical) to 1.0 (perfect)
	SmartSummary   string
	SerialNumber   string
	Model          string
	Rotational     bool
	Mounted        bool
	SmartFailed    bool
	IOErrors       int64
	LastChecked    metav1.Time
}

// DiskIOStats holds I/O counters collected from /proc/diskstats.
type DiskIOStats struct {
	ReadsCompleted  int64
	WritesCompleted int64
	ReadBytes       int64
	WriteBytes      int64
	IOInProgress    int64
	IOTimeMs        int64
}

// MountDiskRequest specifies the parameters for mounting a disk.
type MountDiskRequest struct {
	DevicePath string
	MountPoint string
	Filesystem string
	// FormatIfEmpty instructs the agent to format the disk if no filesystem is detected.
	FormatIfEmpty bool
}

// MergerFSMountRequest specifies parameters for creating a mergerfs pool mount.
type MergerFSMountRequest struct {
	Branches       []string // e.g., ["/mnt/disks/disk1", "/mnt/disks/disk2"]
	MountPoint     string   // e.g., "/mnt/pool"
	CategoryCreate string   // mergerfs policy: epmfs, mfs, lfs, etc.
	MinFreeSpace   string   // e.g., "20G"
	ExtraOptions   string   // additional raw mergerfs options
}

// SnapraidConfig describes the full snapraid.conf content for a node.
type SnapraidConfig struct {
	ParityEntries   []SnapraidParityEntry
	DataEntries     []SnapraidDataEntry
	ContentFiles    []string
	ExcludePatterns []string
}

// SnapraidParityEntry represents a parity definition in snapraid.conf.
type SnapraidParityEntry struct {
	Index      int // 1 = parity, 2 = 2-parity, etc.
	DevicePath string
	MountPoint string
}

// SnapraidDataEntry represents a data disk entry in snapraid.conf.
type SnapraidDataEntry struct {
	Label      string // e.g., "d1", "d2"
	DevicePath string
	MountPoint string
}

// ─────────────────────────────────────────
// Condition and Finalizer Helpers
// ─────────────────────────────────────────

// setCondition upserts a condition by type in the given slice.
func setCondition(conditions *[]metav1.Condition, newCond metav1.Condition) {
	for i, c := range *conditions {
		if c.Type == newCond.Type {
			(*conditions)[i] = newCond
			return
		}
	}
	*conditions = append(*conditions, newCond)
}

// containsString checks whether s is in slice.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return s == item
		}
	}
	return false
}

// removeString removes all occurrences of s from slice.
func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}
