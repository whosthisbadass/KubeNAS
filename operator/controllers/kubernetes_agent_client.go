package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	agentStatusPrefix = "kubenas-diskstatus-"
	agentOpPrefix     = "kubenas-agentop-"
)

func operatorNamespace() string {
	if ns := os.Getenv("WATCH_NAMESPACE"); ns != "" {
		return ns
	}
	return "kubenas-system"
}

type KubernetesAgentClient struct {
	client client.Client
}

func NewKubernetesAgentClient(c client.Client) NodeAgentClient {
	return &KubernetesAgentClient{client: c}
}

func (k *KubernetesAgentClient) GetDiskStatus(ctx context.Context, nodeName, devicePath string) (*DiskAgentStatus, error) {
	cmName := fmt.Sprintf("%s%s", agentStatusPrefix, nodeName)
	cm := &corev1.ConfigMap{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: cmName, Namespace: operatorNamespace()}, cm); err != nil {
		if errors.IsNotFound(err) {
			return &DiskAgentStatus{DevicePath: devicePath, HealthScore: 1.0}, nil
		}
		return nil, err
	}
	data, ok := cm.Data[sanitizeDevicePath(devicePath)]
	if !ok {
		return &DiskAgentStatus{DevicePath: devicePath, HealthScore: 1.0}, nil
	}
	status := &DiskAgentStatus{}
	if err := json.Unmarshal([]byte(data), status); err != nil {
		return nil, err
	}
	return status, nil
}

func (k *KubernetesAgentClient) MountDisk(ctx context.Context, nodeName string, req MountDiskRequest) error {
	opID, err := k.postAgentOp(ctx, nodeName, "mount", req)
	if err != nil {
		return err
	}
	_, err = k.waitForOperationState(ctx, opID)
	return err
}

func (k *KubernetesAgentClient) UnmountDisk(ctx context.Context, nodeName, mountPoint string) error {
	opID, err := k.postAgentOp(ctx, nodeName, "unmount", map[string]string{"mountPoint": mountPoint})
	if err != nil {
		return err
	}
	_, err = k.waitForOperationState(ctx, opID)
	return err
}

func (k *KubernetesAgentClient) ApplySnapraidConfig(ctx context.Context, nodeName string, cfg SnapraidConfig) error {
	rendered := renderSnapraidConf(cfg)
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("kubenas-snapraid-%s", nodeName), Namespace: operatorNamespace(), Labels: map[string]string{"kubenas.io/snapraid-config": "true", "kubenas.io/node": nodeName}}, Data: map[string]string{"snapraid.conf": rendered}}
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

func (k *KubernetesAgentClient) EnsureMergerFSMount(ctx context.Context, nodeName string, req MergerFSMountRequest) (bool, error) {
	opID, err := k.postAgentOp(ctx, nodeName, "mergerfs-mount", req)
	if err != nil {
		return false, err
	}
	st, err := k.waitForOperationState(ctx, opID)
	if err != nil {
		return false, err
	}
	return st.State == "Success", nil
}

func (k *KubernetesAgentClient) RunParityOperation(ctx context.Context, nodeName, operation string) error {
	opID, err := k.postAgentOp(ctx, nodeName, "parity-"+operation, nil)
	if err != nil {
		return err
	}
	_, err = k.waitForOperationState(ctx, opID)
	return err
}

func (k *KubernetesAgentClient) GetDiskIOStats(ctx context.Context, nodeName, devicePath string) (*DiskIOStats, error) {
	cm := &corev1.ConfigMap{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s%s-io", agentStatusPrefix, nodeName), Namespace: operatorNamespace()}, cm); err != nil {
		return &DiskIOStats{}, nil
	}
	data, ok := cm.Data[sanitizeDevicePath(devicePath)]
	if !ok {
		return &DiskIOStats{}, nil
	}
	stats := &DiskIOStats{}
	if err := json.Unmarshal([]byte(data), stats); err != nil {
		return &DiskIOStats{}, nil
	}
	return stats, nil
}

func (k *KubernetesAgentClient) WaitForOperation(ctx context.Context, operationID string) (*AgentOperationStatus, error) {
	return k.waitForOperationState(ctx, operationID)
}

func (k *KubernetesAgentClient) waitForOperationState(ctx context.Context, opName string) (*AgentOperationStatus, error) {
	for i := 0; i < 24; i++ {
		cm := &corev1.ConfigMap{}
		if err := k.client.Get(ctx, types.NamespacedName{Name: opName, Namespace: operatorNamespace()}, cm); err != nil {
			if errors.IsNotFound(err) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(2 * time.Second):
				}
				continue
			}
			return nil, err
		}
		st := &AgentOperationStatus{OperationID: opName, State: cm.Data["status.state"], Message: cm.Data["status.message"]}
		switch st.State {
		case "Success":
			return st, nil
		case "Failed":
			return st, fmt.Errorf("operation %s failed: %s", opName, st.Message)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil, fmt.Errorf("operation %s timed out", opName)
}

func (k *KubernetesAgentClient) postAgentOp(ctx context.Context, nodeName, opType string, payload interface{}) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	opName := fmt.Sprintf("%s%s-%d", agentOpPrefix, nodeName, time.Now().UnixMilli())
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: opName, Namespace: operatorNamespace(), Labels: map[string]string{"kubenas.io/agent-op": "true", "kubenas.io/node": nodeName, "kubenas.io/op-type": opType}}, Data: map[string]string{"operationID": opName, "operation": opType, "node": nodeName, "payload": string(data), "status.state": "Pending", "status.message": "queued"}}
	return opName, k.client.Create(ctx, cm)
}

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

func sanitizeDevicePath(path string) string { return strings.ReplaceAll(path, "/", "_") }
