package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1alpha1 "github.com/whosthisbadass/KubeNAS/api/v1alpha1"
)

// PoolReconciler reconciles mergerfs pool configuration.
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=pools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=pools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=arrays,verbs=get;list;watch
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=disks,verbs=get;list;watch
type PoolReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Agent  NodeAgentClient
}

// Reconcile implements the reconciliation loop for a Pool CR.
func (r *PoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("pool", req.NamespacedName)

	pool := &storagev1alpha1.Pool{}
	if err := r.Get(ctx, req.NamespacedName, pool); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Resolve the backing array.
	array := &storagev1alpha1.Array{}
	if err := r.Get(ctx, types.NamespacedName{Name: pool.Spec.ArrayRef, Namespace: pool.Namespace}, array); err != nil {
		return r.setPoolPhase(ctx, pool, "Pending", "ArrayNotFound", fmt.Sprintf("Array %q not found", pool.Spec.ArrayRef))
	}

	if array.Status.Phase != storagev1alpha1.ArrayPhaseReady {
		log.Info("array not yet ready, requeueing", "arrayPhase", array.Status.Phase)
		return r.setPoolPhase(ctx, pool, "Pending", "ArrayNotReady", fmt.Sprintf("Array %q is not Ready (%s)", pool.Spec.ArrayRef, array.Status.Phase))
	}

	// Resolve data disks and collect mount points.
	var branches []string
	var diskNames []string
	var totalBytes, availableBytes int64

	for _, diskName := range array.Spec.DataDisks {
		disk := &storagev1alpha1.Disk{}
		if err := r.Get(ctx, types.NamespacedName{Name: diskName}, disk); err != nil {
			log.Error(err, "failed to fetch disk for pool branches", "disk", diskName)
			continue
		}
		if disk.Status.Mounted && disk.Status.Phase == storagev1alpha1.DiskPhaseReady {
			branches = append(branches, disk.Spec.MountPoint)
			diskNames = append(diskNames, diskName)
			totalBytes += disk.Status.CapacityBytes
			availableBytes += disk.Status.AvailableBytes
		}
	}

	if len(branches) == 0 {
		return r.setPoolPhase(ctx, pool, "Pending", "NoBranchesReady", "No data disks are ready and mounted")
	}

	// Determine the node to apply mount (single-node assumed for SNO).
	nodeName := ""
	for _, diskName := range array.Spec.DataDisks {
		disk := &storagev1alpha1.Disk{}
		if err := r.Get(ctx, types.NamespacedName{Name: diskName}, disk); err == nil {
			nodeName = disk.Spec.NodeName
			break
		}
	}

	// Build mergerfs mount request.
	mountReq := MergerFSMountRequest{
		Branches:       branches,
		MountPoint:     pool.Spec.MountPoint,
		CategoryCreate: r.categoryCreate(pool),
		MinFreeSpace:   r.minFreeSpace(pool),
		ExtraOptions:   r.extraOptions(pool),
	}

	log.Info("ensuring mergerfs pool mount", "mountPoint", pool.Spec.MountPoint, "branches", strings.Join(branches, ":"))

	mounted, err := r.Agent.EnsureMergerFSMount(ctx, nodeName, mountReq)
	if err != nil {
		log.Error(err, "failed to ensure mergerfs mount")
		return r.setPoolPhase(ctx, pool, "Degraded", "MountError", err.Error())
	}

	pool.Status.TotalBytes = totalBytes
	pool.Status.AvailableBytes = availableBytes
	pool.Status.MemberDisks = diskNames
	pool.Status.Mounted = mounted

	return r.setPoolPhase(ctx, pool, "Ready", "Reconciled", fmt.Sprintf("Pool mounted with %d disks at %s", len(branches), pool.Spec.MountPoint))
}

func (r *PoolReconciler) categoryCreate(pool *storagev1alpha1.Pool) string {
	if pool.Spec.Mergerfs != nil && pool.Spec.Mergerfs.CategoryCreate != "" {
		return pool.Spec.Mergerfs.CategoryCreate
	}
	return "epmfs"
}

func (r *PoolReconciler) minFreeSpace(pool *storagev1alpha1.Pool) string {
	if pool.Spec.Mergerfs != nil && pool.Spec.Mergerfs.MinFreeSpace != "" {
		return pool.Spec.Mergerfs.MinFreeSpace
	}
	return "20G"
}

func (r *PoolReconciler) extraOptions(pool *storagev1alpha1.Pool) string {
	if pool.Spec.Mergerfs != nil {
		return pool.Spec.Mergerfs.ExtraOptions
	}
	return ""
}

func (r *PoolReconciler) setPoolPhase(ctx context.Context, pool *storagev1alpha1.Pool, phase, reason, msg string) (ctrl.Result, error) {
	pool.Status.Phase = phase
	setCondition(&pool.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	})
	if phase != "Ready" {
		pool.Status.Conditions[len(pool.Status.Conditions)-1].Status = metav1.ConditionFalse
	}
	if err := r.Status().Update(ctx, pool); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating pool status: %w", err)
	}
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager registers the PoolReconciler.
func (r *PoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1alpha1.Pool{}).
		Complete(r)
}
