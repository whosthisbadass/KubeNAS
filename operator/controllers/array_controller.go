package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1alpha1 "github.com/kubenas/kubenas/operator/api/v1alpha1"
)

// ArrayReconciler reconciles Array resources and generates snapraid intent.
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=arrays,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=arrays/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=disks,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
type ArrayReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Agent  NodeAgentClient
}

// Reconcile implements the reconciliation loop for an Array CR.
func (r *ArrayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("array", req.NamespacedName)

	array := &storagev1alpha1.Array{}
	if err := r.Get(ctx, req.NamespacedName, array); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Validate all disk references exist and are Ready.
	dataDisks, err := r.resolveDiskRefs(ctx, array.Spec.DataDisks)
	if err != nil {
		log.Error(err, "data disk reference resolution failed")
		return r.setArrayPhase(ctx, array, storagev1alpha1.ArrayPhasePending, "DiskRefError", err.Error())
	}

	parityDisks, err := r.resolveDiskRefs(ctx, array.Spec.ParityDisks)
	if err != nil {
		log.Error(err, "parity disk reference resolution failed")
		return r.setArrayPhase(ctx, array, storagev1alpha1.ArrayPhasePending, "ParityDiskRefError", err.Error())
	}

	// Check for degraded disks.
	var degraded []string
	for _, d := range append(dataDisks, parityDisks...) {
		if d.Status.Phase == storagev1alpha1.DiskPhaseDegraded || d.Status.Phase == storagev1alpha1.DiskPhaseFailed {
			degraded = append(degraded, d.Name)
		}
	}

	array.Status.DegradedDisks = degraded

	if len(degraded) > 0 {
		log.Info("array has degraded disks", "disks", degraded)
	}

	// Build SnapRAID configuration and push to node agent on the primary disk's node.
	if len(dataDisks) > 0 {
		nodeName := dataDisks[0].Spec.NodeName
		snapConfig := r.buildSnapraidConfig(array, dataDisks, parityDisks)
		if err := r.Agent.ApplySnapraidConfig(ctx, nodeName, snapConfig); err != nil {
			log.Error(err, "failed to push snapraid config to node agent")
			return r.setArrayPhase(ctx, array, storagev1alpha1.ArrayPhaseDegraded, "SnapraidConfigError", err.Error())
		}
	}

	targetPhase := storagev1alpha1.ArrayPhaseReady
	if len(degraded) > 0 {
		targetPhase = storagev1alpha1.ArrayPhaseDegraded
	}

	return r.setArrayPhase(ctx, array, targetPhase, "Reconciled", fmt.Sprintf("Array reconciled with %d data disks, %d parity disks", len(dataDisks), len(parityDisks)))
}

// resolveDiskRefs fetches Disk CRs by name. Returns an error if any is missing.
func (r *ArrayReconciler) resolveDiskRefs(ctx context.Context, names []string) ([]*storagev1alpha1.Disk, error) {
	var disks []*storagev1alpha1.Disk
	for _, name := range names {
		disk := &storagev1alpha1.Disk{}
		// Disk is cluster-scoped, no namespace needed.
		if err := r.Get(ctx, types.NamespacedName{Name: name}, disk); err != nil {
			if errors.IsNotFound(err) {
				return nil, fmt.Errorf("disk %q not found", name)
			}
			return nil, fmt.Errorf("fetching disk %q: %w", name, err)
		}
		disks = append(disks, disk)
	}
	return disks, nil
}

// buildSnapraidConfig constructs a SnapRAID configuration struct for the agent.
func (r *ArrayReconciler) buildSnapraidConfig(array *storagev1alpha1.Array, data, parity []*storagev1alpha1.Disk) SnapraidConfig {
	cfg := SnapraidConfig{}

	for i, d := range parity {
		cfg.ParityEntries = append(cfg.ParityEntries, SnapraidParityEntry{
			Index:      i + 1,
			DevicePath: d.Spec.DevicePath,
			MountPoint: d.Spec.MountPoint,
		})
	}

	for i, d := range data {
		cfg.DataEntries = append(cfg.DataEntries, SnapraidDataEntry{
			Label:      fmt.Sprintf("d%d", i+1),
			DevicePath: d.Spec.DevicePath,
			MountPoint: d.Spec.MountPoint,
		})
	}

	if array.Spec.SnapraidConfig != nil {
		cfg.ContentFiles = array.Spec.SnapraidConfig.ContentFiles
		cfg.ExcludePatterns = array.Spec.SnapraidConfig.ExcludePatterns
	}

	// Default content files if not specified.
	if len(cfg.ContentFiles) == 0 && len(data) > 0 {
		for _, d := range data {
			cfg.ContentFiles = append(cfg.ContentFiles, fmt.Sprintf("%s/.snapraid.content", d.Spec.MountPoint))
		}
	}

	return cfg
}

func (r *ArrayReconciler) setArrayPhase(ctx context.Context, array *storagev1alpha1.Array, phase storagev1alpha1.ArrayPhase, reason, msg string) (ctrl.Result, error) {
	array.Status.Phase = phase
	setCondition(&array.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	})
	if phase != storagev1alpha1.ArrayPhaseReady {
		array.Status.Conditions[len(array.Status.Conditions)-1].Status = metav1.ConditionFalse
	}
	if err := r.Status().Update(ctx, array); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating array status: %w", err)
	}
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

// SetupWithManager registers the ArrayReconciler with the controller manager.
func (r *ArrayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1alpha1.Array{}).
		Complete(r)
}
