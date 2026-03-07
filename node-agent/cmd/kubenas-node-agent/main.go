// KubeNAS Node Agent
// Runs as a privileged DaemonSet on storage nodes.
// Responsibilities: disk discovery, SMART monitoring, mount management,
// mergerfs pooling, SnapRAID parity operations, and status reporting
// via Kubernetes ConfigMaps consumed by the KubeNAS Operator.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubenas/kubenas/node-agent/pkg/disk"
	"github.com/kubenas/kubenas/node-agent/pkg/smart"
)

const (
	namespace          = "kubenas-system"
	discoveryInterval  = 30 * time.Second
	smartCheckInterval = 5 * time.Minute
)

func main() {
	var kubeconfigPath string
	var nodeName string
	var hostDevPath string

	flag.StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig (leave empty for in-cluster)")
	flag.StringVar(&nodeName, "node-name", "", "Kubernetes node name this agent runs on")
	flag.StringVar(&hostDevPath, "host-dev", "/host/dev", "Path to host /dev directory")
	flag.Parse()

	if nodeName == "" {
		nodeName = os.Getenv("NODE_NAME")
	}
	if nodeName == "" {
		fmt.Fprintln(os.Stderr, "node-name is required (set via --node-name or NODE_NAME env)")
		os.Exit(1)
	}

	zapLog, _ := zap.NewDevelopment()
	log := zapr.NewLogger(zapLog)

	log.Info("KubeNAS Node Agent starting", "node", nodeName)

	// Build Kubernetes client.
	var config *rest.Config
	var err error
	if kubeconfigPath != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Error(err, "failed to build Kubernetes config")
		os.Exit(1)
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err, "failed to build Kubernetes client")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigCh
		log.Info("received shutdown signal", "signal", sig)
		cancel()
	}()

	// Start the main agent loop.
	agent := &NodeAgent{
		k8s:         k8sClient,
		dynamic:     dynamic.NewForConfigOrDie(config),
		nodeName:    nodeName,
		hostDevPath: hostDevPath,
		log:         log,
	}

	agent.Run(ctx)
	log.Info("KubeNAS Node Agent stopped")
}

// NodeAgent is the main agent struct managing all node-level operations.
type NodeAgent struct {
	k8s         kubernetes.Interface
	dynamic     dynamic.Interface
	nodeName    string
	hostDevPath string
	log         interface {
		Info(msg string, keysAndValues ...interface{})
		Error(err error, msg string, keysAndValues ...interface{})
	}
}

// Run starts the agent's main reconciliation loops.
func (a *NodeAgent) Run(ctx context.Context) {
	discoveryTicker := time.NewTicker(discoveryInterval)
	smartTicker := time.NewTicker(smartCheckInterval)
	opWatchTicker := time.NewTicker(5 * time.Second)

	defer discoveryTicker.Stop()
	defer smartTicker.Stop()
	defer opWatchTicker.Stop()

	// Run discovery immediately on start.
	a.runDiscovery(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-discoveryTicker.C:
			a.runDiscovery(ctx)
		case <-smartTicker.C:
			a.runSmartChecks(ctx)
		case <-opWatchTicker.C:
			a.processOperationRequests(ctx)
		}
	}
}

// runDiscovery discovers all block devices and reports status to the operator.
func (a *NodeAgent) runDiscovery(ctx context.Context) {
	a.log.Info("running disk discovery")

	devices, err := disk.DiscoverDevices()
	if err != nil {
		a.log.Error(err, "disk discovery failed")
		return
	}

	a.log.Info("discovered devices", "count", len(devices))

	// Build status map: device path → serialized status.
	statusData := make(map[string]string)
	for _, dev := range devices {
		// Run a quick SMART query for each disk.
		health := &smart.DiskHealth{
			DevicePath:    dev.DevicePath,
			Model:         dev.Model,
			SerialNumber:  dev.Serial,
			Rotational:    dev.Rotational,
			HealthScore:   1.0,
			OverallHealth: "UNKNOWN",
		}

		if smart.IsSmartAvailable() {
			if h, err := smart.QueryDisk(dev.DevicePath); err == nil {
				health = h
			}
		}

		status := AgentDiskStatus{
			DevicePath:    dev.DevicePath,
			CapacityBytes: dev.SizeBytes,
			Rotational:    dev.Rotational,
			Model:         dev.Model,
			SerialNumber:  dev.Serial,
			Filesystem:    dev.Filesystem,
			MountPoint:    dev.MountPoint,
			Mounted:       dev.MountPoint != "",
			HealthScore:   health.HealthScore,
			SmartSummary:  smart.FormatSMARTSummary(health),
			SmartFailed:   health.SmartFailed,
		}

		if dev.MountPoint != "" {
			_, avail, err := disk.GetDiskUsage(dev.MountPoint)
			if err == nil {
				status.AvailableBytes = avail
			}
		}

		key := sanitizeKey(dev.DevicePath)
		data, _ := json.Marshal(status)
		statusData[key] = string(data)
	}

	// Write status to a ConfigMap in kubenas-system.
	cmName := fmt.Sprintf("kubenas-diskstatus-%s", a.nodeName)
	a.upsertConfigMap(ctx, cmName, statusData)
	a.upsertDiskCRs(ctx, devices)
}

func (a *NodeAgent) upsertDiskCRs(ctx context.Context, devices []disk.Device) {
	gvr := schema.GroupVersionResource{Group: "storage.kubenas.io", Version: "v1", Resource: "disks"}
	for _, d := range devices {
		name := "disk-" + strings.TrimPrefix(strings.ReplaceAll(d.DevicePath, "/", "-"), "-dev-")
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "storage.kubenas.io/v1",
			"kind":       "Disk",
			"metadata":   map[string]interface{}{"name": name},
			"spec": map[string]interface{}{
				"device":     d.DevicePath,
				"size":       fmt.Sprintf("%d", d.SizeBytes),
				"rotational": d.Rotational,
				"filesystem": d.Filesystem,
				"state":      "Detected",
				"health":     "Healthy",
			},
		}}

		if _, err := a.dynamic.Resource(gvr).Get(ctx, name, metav1.GetOptions{}); err == nil {
			_, _ = a.dynamic.Resource(gvr).Update(ctx, obj, metav1.UpdateOptions{})
		} else {
			_, _ = a.dynamic.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
		}
	}
}

// runSmartChecks performs periodic deep SMART checks across all disks.
func (a *NodeAgent) runSmartChecks(ctx context.Context) {
	a.log.Info("running SMART health checks")

	if !smart.IsSmartAvailable() {
		a.log.Info("smartctl not available, skipping SMART checks")
		return
	}

	devices, err := disk.DiscoverDevices()
	if err != nil {
		a.log.Error(err, "device discovery for SMART check failed")
		return
	}

	for _, dev := range devices {
		health, err := smart.QueryDisk(dev.DevicePath)
		if err != nil {
			a.log.Error(err, "SMART query failed", "device", dev.DevicePath)
			continue
		}

		if health.SmartFailed || health.HealthScore < 0.5 {
			a.log.Info("disk health degraded",
				"device", dev.DevicePath,
				"healthScore", health.HealthScore,
				"summary", smart.FormatSMARTSummary(health),
			)
		}
	}
}

// processOperationRequests watches for operator-posted operation ConfigMaps and executes them.
func (a *NodeAgent) processOperationRequests(ctx context.Context) {
	cms, err := a.k8s.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kubenas.io/agent-op=true,kubenas.io/node=%s", a.nodeName),
	})
	if err != nil {
		a.log.Error(err, "listing agent operation ConfigMaps")
		return
	}

	for _, cm := range cms.Items {
		a.executeOperation(ctx, cm)
	}
}

// executeOperation runs the action described in an operation ConfigMap.
func (a *NodeAgent) executeOperation(ctx context.Context, cm corev1.ConfigMap) {
	opType := cm.Data["operation"]
	a.log.Info("executing agent operation", "op", opType, "cm", cm.Name)

	var execErr error
	switch opType {
	case "mount":
		var req MountRequest
		if err := json.Unmarshal([]byte(cm.Data["payload"]), &req); err == nil {
			execErr = disk.MountDevice(req.DevicePath, req.MountPoint, req.Filesystem)
		}
	case "unmount":
		var req UnmountRequest
		if err := json.Unmarshal([]byte(cm.Data["payload"]), &req); err == nil {
			execErr = disk.UnmountDevice(req.MountPoint)
		}
	case "mergerfs-mount":
		var req MergerFSMountRequest
		if err := json.Unmarshal([]byte(cm.Data["payload"]), &req); err == nil {
			execErr = disk.MountMergerFS(req.Branches, req.MountPoint, req.CategoryCreate, req.MinFreeSpace, req.ExtraOptions)
		}
	default:
		a.log.Info("unknown operation type, skipping", "op", opType)
	}

	if execErr != nil {
		a.log.Error(execErr, "operation failed", "op", opType)
	}

	// Delete the operation ConfigMap after processing (consumed pattern).
	_ = a.k8s.CoreV1().ConfigMaps(namespace).Delete(ctx, cm.Name, metav1.DeleteOptions{})
}

// upsertConfigMap creates or updates a ConfigMap with the given data.
func (a *NodeAgent) upsertConfigMap(ctx context.Context, name string, data map[string]string) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"kubenas.io/agent-status": "true",
				"kubenas.io/node":         a.nodeName,
			},
		},
		Data: data,
	}

	existing, err := a.k8s.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		// Create.
		if _, createErr := a.k8s.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{}); createErr != nil {
			a.log.Error(createErr, "failed to create status ConfigMap", "name", name)
		}
		return
	}

	// Update.
	existing.Data = data
	if _, updateErr := a.k8s.CoreV1().ConfigMaps(namespace).Update(ctx, existing, metav1.UpdateOptions{}); updateErr != nil {
		a.log.Error(updateErr, "failed to update status ConfigMap", "name", name)
	}
}

// sanitizeKey converts /dev/sdb → dev_sdb for ConfigMap keys.
func sanitizeKey(path string) string {
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

// ─────────────────────────────────────────
// Operation Payload Types
// ─────────────────────────────────────────

// AgentDiskStatus is written by the agent into status ConfigMaps.
type AgentDiskStatus struct {
	DevicePath     string  `json:"devicePath"`
	CapacityBytes  int64   `json:"capacityBytes"`
	AvailableBytes int64   `json:"availableBytes"`
	Rotational     bool    `json:"rotational"`
	Model          string  `json:"model"`
	SerialNumber   string  `json:"serialNumber"`
	Filesystem     string  `json:"filesystem"`
	MountPoint     string  `json:"mountPoint"`
	Mounted        bool    `json:"mounted"`
	HealthScore    float64 `json:"healthScore"`
	SmartSummary   string  `json:"smartSummary"`
	SmartFailed    bool    `json:"smartFailed"`
	IOErrors       int64   `json:"ioErrors"`
}

// MountRequest carries mount operation parameters.
type MountRequest struct {
	DevicePath    string `json:"devicePath"`
	MountPoint    string `json:"mountPoint"`
	Filesystem    string `json:"filesystem"`
	FormatIfEmpty bool   `json:"formatIfEmpty"`
}

// UnmountRequest carries unmount operation parameters.
type UnmountRequest struct {
	MountPoint string `json:"mountPoint"`
}

// MergerFSMountRequest carries mergerfs pool mount parameters.
type MergerFSMountRequest struct {
	Branches       []string `json:"branches"`
	MountPoint     string   `json:"mountPoint"`
	CategoryCreate string   `json:"categoryCreate"`
	MinFreeSpace   string   `json:"minFreeSpace"`
	ExtraOptions   string   `json:"extraOptions"`
}
