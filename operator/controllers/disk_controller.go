package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1alpha1 "github.com/kubenas/kubenas/operator/api/v1alpha1"
)

// DiskReconciler reconciles Disk CRs with node-agent disk discovery state.
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=disks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=disks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=disks/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
type DiskReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Agent  NodeAgentClient
}

const diskFinalizer = "storage.kubenas.io/disk-protection"

// Reconcile implements the main reconciliation loop for a Disk CR.
func (r *DiskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("disk", req.NamespacedName)

	// Fetch the Disk CR.
	disk := &storagev1alpha1.Disk{}
	if err := r.Get(ctx, req.NamespacedName, disk); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion lifecycle with finalizer.
	if !disk.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, log, disk)
	}

	// Ensure finalizer is present.
	if !containsString(disk.Finalizers, diskFinalizer) {
		disk.Finalizers = append(disk.Finalizers, diskFinalizer)
		if err := r.Update(ctx, disk); err != nil {
			return ctrl.Result{}, fmt.Errorf("adding finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return r.reconcileNormal(ctx, log, disk)
}

func (r *DiskReconciler) reconcileNormal(ctx context.Context, log logr.Logger, disk *storagev1alpha1.Disk) (ctrl.Result, error) {
	// Query the node agent for live disk state.
	agentStatus, err := r.Agent.GetDiskStatus(ctx, disk.Spec.NodeName, disk.Spec.DevicePath)
	if err != nil {
		log.Error(err, "failed to query node agent for disk status")
		return r.setPhase(ctx, disk, storagev1alpha1.DiskPhaseDegraded, "AgentUnavailable", err.Error())
	}

	// Build updated status from agent response.
	disk.Status.CapacityBytes = agentStatus.CapacityBytes
	disk.Status.AvailableBytes = agentStatus.AvailableBytes
	disk.Status.HealthScore = agentStatus.HealthScore
	disk.Status.SmartSummary = agentStatus.SmartSummary
	disk.Status.SerialNumber = agentStatus.SerialNumber
	disk.Status.Model = agentStatus.Model
	disk.Status.Rotational = agentStatus.Rotational
	disk.Status.Mounted = agentStatus.Mounted
	now := metav1.NewTime(time.Now())
	disk.Status.LastSmartCheckTime = &now

	// Determine disk phase from agent health data.
	targetPhase := r.computePhase(agentStatus)

	// Request mount if disk is not yet mounted and should be.
	if !agentStatus.Mounted && targetPhase == storagev1alpha1.DiskPhaseReady {
		if err := r.Agent.MountDisk(ctx, disk.Spec.NodeName, MountDiskRequest{
			DevicePath: disk.Spec.DevicePath,
			MountPoint: disk.Spec.MountPoint,
			Filesystem: disk.Spec.Filesystem,
		}); err != nil {
			log.Error(err, "failed to mount disk via node agent")
			return r.setPhase(ctx, disk, storagev1alpha1.DiskPhaseDegraded, "MountFailed", err.Error())
		}
		disk.Status.Mounted = true
	}

	return r.setPhase(ctx, disk, targetPhase, "Reconciled", "Disk reconciliation complete")
}

func (r *DiskReconciler) reconcileDelete(ctx context.Context, log logr.Logger, disk *storagev1alpha1.Disk) (ctrl.Result, error) {
	log.Info("disk deletion requested, unmounting")

	if disk.Status.Mounted {
		if err := r.Agent.UnmountDisk(ctx, disk.Spec.NodeName, disk.Spec.MountPoint); err != nil {
			log.Error(err, "failed to unmount disk on delete")
			// Don't block deletion — log and continue.
		}
	}

	disk.Finalizers = removeString(disk.Finalizers, diskFinalizer)
	if err := r.Update(ctx, disk); err != nil {
		return ctrl.Result{}, fmt.Errorf("removing finalizer: %w", err)
	}
	return ctrl.Result{}, nil
}

// computePhase derives the target DiskPhase from agent-reported state.
func (r *DiskReconciler) computePhase(s *DiskAgentStatus) storagev1alpha1.DiskPhase {
	switch {
	case s.SmartFailed:
		return storagev1alpha1.DiskPhaseFailed
	case s.HealthScore < 0.3:
		return storagev1alpha1.DiskPhaseDegraded
	case s.IOErrors > 0:
		return storagev1alpha1.DiskPhaseDegraded
	default:
		return storagev1alpha1.DiskPhaseReady
	}
}

// setPhase patches the Disk status with the given phase and condition.
func (r *DiskReconciler) setPhase(ctx context.Context, disk *storagev1alpha1.Disk, phase storagev1alpha1.DiskPhase, reason, msg string) (ctrl.Result, error) {
	disk.Status.Phase = phase
	setCondition(&disk.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             conditionStatusForPhase(phase),
		Reason:             reason,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	})
	if err := r.Status().Update(ctx, disk); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating disk status: %w", err)
	}
	// Re-check every 30 seconds for SMART drift.
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager registers the DiskReconciler with the controller manager.
func (r *DiskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1alpha1.Disk{}).
		Complete(r)
}

func conditionStatusForPhase(phase storagev1alpha1.DiskPhase) metav1.ConditionStatus {
	if phase == storagev1alpha1.DiskPhaseReady {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}
