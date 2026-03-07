package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	agentNamespace    = "kubenas-system"
	agentStatusPrefix = "kubenas-diskstatus-"
	agentOpPrefix     = "kubenas-agentop-"
)

// KubernetesAgentClient implements NodeAgentClient using Kubernetes ConfigMaps
// as a communication channel between the operator and Node Agent DaemonSet pods.
//
// The Node Agent watches for ConfigMaps with label kubenas.io/agent-op=true,
// executes the requested operation, and writes results to a corresponding
// status ConfigMap.
//
// This approach avoids requiring a direct network path to node agent pods and
// works seamlessly in restricted OpenShift/OKD network policies.
type KubernetesAgentClient struct {
	client client.Client
}

// NewKubernetesAgentClient creates a new Kubernetes-backed NodeAgentClient.
func NewKubernetesAgentClient(c client.Client) NodeAgentClient {
	return &KubernetesAgentClient{client: c}
}

// GetDiskStatus reads disk status from the Node Agent's status ConfigMap.
func (k *KubernetesAgentClient) GetDiskStatus(ctx context.Context, nodeName, devicePath string) (*DiskAgentStatus, error) {
	cmName := fmt.Sprintf("%s%s", agentStatusPrefix, nodeName)
	cm := &corev1.ConfigMap{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: cmName, Namespace: agentNamespace}, cm); err != nil {
		if errors.IsNotFound(err) {
			// Agent has not yet reported; return a pending status.
			return &DiskAgentStatus{
				DevicePath:  devicePath,
				HealthScore: 1.0,
				Mounted:     false,
			}, nil
		}
		return nil, fmt.Errorf("fetching disk status configmap %q: %w", cmName, err)
	}

	key := sanitizeDevicePath(devicePath)
	data, ok := cm.Data[key]
	if !ok {
		return &DiskAgentStatus{DevicePath: devicePath, HealthScore: 1.0}, nil
	}

	status := &DiskAgentStatus{}
	if err := json.Unmarshal([]byte(data), status); err != nil {
		return nil, fmt.Errorf("parsing disk status for %s: %w", devicePath, err)
	}
	return status, nil
}

// MountDisk creates a mount operation request ConfigMap for the Node Agent.
func (k *KubernetesAgentClient) MountDisk(ctx context.Context, nodeName string, req MountDiskRequest) error {
	return k.postAgentOp(ctx, nodeName, "mount", req)
}

// UnmountDisk creates an unmount operation request for the Node Agent.
func (k *KubernetesAgentClient) UnmountDisk(ctx context.Context, nodeName, mountPoint string) error {
	return k.postAgentOp(ctx, nodeName, "unmount", map[string]string{"mountPoint": mountPoint})
}

// ApplySnapraidConfig writes the snapraid config to a ConfigMap the Node Agent reads.
func (k *KubernetesAgentClient) ApplySnapraidConfig(ctx context.Context, nodeName string, cfg SnapraidConfig) error {
	rendered := renderSnapraidConf(cfg)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("kubenas-snapraid-%s", nodeName),
			Namespace: agentNamespace,
			Labels: map[string]string{
				"kubenas.io/snapraid-config": "true",
				"kubenas.io/node":            nodeName,
			},
		},
		Data: map[string]string{
			"snapraid.conf": rendered,
		},
	}

	existing := &corev1.ConfigMap{}
	err := k.client.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, existing)
	if errors.IsNotFound(err) {
		return k.client.Create(ctx, cm)
	} else if err != nil {
		return err
	}

	existing.Data = cm.Data
	return k.client.Update(ctx, existing)
}

// EnsureMergerFSMount creates a mergerfs mount operation for the Node Agent.
func (k *KubernetesAgentClient) EnsureMergerFSMount(ctx context.Context, nodeName string, req MergerFSMountRequest) (bool, error) {
	if err := k.postAgentOp(ctx, nodeName, "mergerfs-mount", req); err != nil {
		return false, err
	}
	// Optimistically return true; agent status reconciliation will correct if needed.
	return true, nil
}

// RunParityOperation posts a parity operation request.
func (k *KubernetesAgentClient) RunParityOperation(ctx context.Context, nodeName, operation string) error {
	return k.postAgentOp(ctx, nodeName, "parity-"+operation, nil)
}

// GetDiskIOStats reads I/O statistics from the agent status ConfigMap.
func (k *KubernetesAgentClient) GetDiskIOStats(ctx context.Context, nodeName, devicePath string) (*DiskIOStats, error) {
	cmName := fmt.Sprintf("%s%s-io", agentStatusPrefix, nodeName)
	cm := &corev1.ConfigMap{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: cmName, Namespace: agentNamespace}, cm); err != nil {
		return &DiskIOStats{}, nil
	}

	key := sanitizeDevicePath(devicePath)
	data, ok := cm.Data[key]
	if !ok {
		return &DiskIOStats{}, nil
	}

	stats := &DiskIOStats{}
	if err := json.Unmarshal([]byte(data), stats); err != nil {
		return &DiskIOStats{}, nil
	}
	return stats, nil
}

// postAgentOp serializes an operation and writes it to a request ConfigMap.
func (k *KubernetesAgentClient) postAgentOp(ctx context.Context, nodeName, opType string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling agent op payload: %w", err)
	}

	opName := fmt.Sprintf("%s%s-%d", agentOpPrefix, nodeName, time.Now().UnixMilli())
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opName,
			Namespace: agentNamespace,
			Labels: map[string]string{
				"kubenas.io/agent-op": "true",
				"kubenas.io/node":     nodeName,
				"kubenas.io/op-type":  opType,
			},
		},
		Data: map[string]string{
			"operation": opType,
			"node":      nodeName,
			"payload":   string(data),
		},
	}

	return k.client.Create(ctx, cm)
}

// renderSnapraidConf generates a snapraid.conf from a SnapraidConfig struct.
func renderSnapraidConf(cfg SnapraidConfig) string {
	conf := "# Generated by KubeNAS Operator\n\n"

	for _, p := range cfg.ParityEntries {
		if p.Index == 1 {
			conf += fmt.Sprintf("parity %s/%s.parity\n", p.MountPoint, "snapraid")
		} else {
			conf += fmt.Sprintf("%d-parity %s/%s.parity\n", p.Index, p.MountPoint, "snapraid")
		}
	}
	conf += "\n"

	for _, cf := range cfg.ContentFiles {
		conf += fmt.Sprintf("content %s\n", cf)
	}
	conf += "\n"

	for _, d := range cfg.DataEntries {
		conf += fmt.Sprintf("data %s %s\n", d.Label, d.MountPoint)
	}
	conf += "\n"

	for _, ex := range cfg.ExcludePatterns {
		conf += fmt.Sprintf("exclude %s\n", ex)
	}

	return conf
}

// sanitizeDevicePath converts /dev/sdb → dev_sdb for use as a ConfigMap key.
func sanitizeDevicePath(path string) string {
	result := ""
	for _, c := range path {
		if c == '/' {
			result += "_"
		} else {
			result += string(c)
		}
	}
	return result
}
