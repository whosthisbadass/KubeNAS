package controllers

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1alpha1 "github.com/kubenas/kubenas/operator/api/v1alpha1"
)

// RebalanceReconciler orchestrates background file movement across pool disks.
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=rebalancejobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=rebalancejobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=pools,verbs=get;list;watch
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=disks,verbs=get;list;watch
type RebalanceReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Agent     NodeAgentClient
	Scheduler *PlacementScheduler
}

// Reconcile drives the rebalance job state machine.
func (r *RebalanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("rebalancejob", req.NamespacedName)

	job := &storagev1alpha1.RebalanceJob{}
	if err := r.Get(ctx, req.NamespacedName, job); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Terminal states: don't reconcile further.
	if job.Status.Phase == storagev1alpha1.RebalanceJobPhaseCompleted ||
		job.Status.Phase == storagev1alpha1.RebalanceJobPhaseFailed {
		return ctrl.Result{}, nil
	}

	// Resolve pool.
	pool := &storagev1alpha1.Pool{}
	if err := r.Get(ctx, types.NamespacedName{Name: job.Spec.PoolRef, Namespace: job.Namespace}, pool); err != nil {
		return r.setJobPhase(ctx, job, storagev1alpha1.RebalanceJobPhaseFailed, "PoolNotFound")
	}

	// Resolve placement policy.
	policy := &storagev1alpha1.PlacementPolicy{}
	if err := r.Get(ctx, types.NamespacedName{Name: job.Spec.PlacementPolicyRef, Namespace: job.Namespace}, policy); err != nil {
		return r.setJobPhase(ctx, job, storagev1alpha1.RebalanceJobPhaseFailed, "PolicyNotFound")
	}

	// Gather member disks.
	var disks []*storagev1alpha1.Disk
	for _, diskName := range pool.Status.MemberDisks {
		disk := &storagev1alpha1.Disk{}
		if err := r.Get(ctx, types.NamespacedName{Name: diskName}, disk); err == nil {
			disks = append(disks, disk)
		}
	}

	if len(disks) < 2 {
		log.Info("fewer than 2 ready disks, nothing to rebalance")
		return r.setJobPhase(ctx, job, storagev1alpha1.RebalanceJobPhaseCompleted, "NothingToRebalance")
	}

	// Check if rebalancing is needed.
	if !r.isImbalanced(disks, job.Spec.ImbalanceThresholdPercent) {
		log.Info("disks are balanced, no rebalance needed")
		return r.setJobPhase(ctx, job, storagev1alpha1.RebalanceJobPhaseCompleted, "AlreadyBalanced")
	}

	// Transition to Planning if we're in Pending.
	if job.Status.Phase == storagev1alpha1.RebalanceJobPhasePending {
		return r.setJobPhase(ctx, job, storagev1alpha1.RebalanceJobPhasePlanning, "PlanningStarted")
	}

	// In dry-run mode, just report the plan.
	if job.Spec.DryRun {
		log.Info("dry-run mode: rebalance plan computed, no moves executed",
			"diskCount", len(disks),
			"threshold", job.Spec.ImbalanceThresholdPercent)
		return r.setJobPhase(ctx, job, storagev1alpha1.RebalanceJobPhaseCompleted, "DryRunComplete")
	}

	// Begin running rebalance moves.
	return r.setJobPhase(ctx, job, storagev1alpha1.RebalanceJobPhaseRunning, "MovesStarted")
}

// isImbalanced returns true when disk utilization variance exceeds the threshold.
func (r *RebalanceReconciler) isImbalanced(disks []*storagev1alpha1.Disk, thresholdPercent int) bool {
	if len(disks) < 2 || thresholdPercent <= 0 {
		return false
	}

	var utilizations []float64
	for _, d := range disks {
		if d.Status.CapacityBytes > 0 {
			used := d.Status.CapacityBytes - d.Status.AvailableBytes
			utilizations = append(utilizations, float64(used)/float64(d.Status.CapacityBytes)*100)
		}
	}

	if len(utilizations) < 2 {
		return false
	}

	min, max := utilizations[0], utilizations[0]
	for _, u := range utilizations[1:] {
		if u < min {
			min = u
		}
		if u > max {
			max = u
		}
	}

	return math.Abs(max-min) > float64(thresholdPercent)
}

func (r *RebalanceReconciler) setJobPhase(ctx context.Context, job *storagev1alpha1.RebalanceJob, phase storagev1alpha1.RebalanceJobPhase, reason string) (ctrl.Result, error) {
	job.Status.Phase = phase
	if phase == storagev1alpha1.RebalanceJobPhaseRunning && job.Status.StartTime == nil {
		now := metav1.Now()
		job.Status.StartTime = &now
	}
	if phase == storagev1alpha1.RebalanceJobPhaseCompleted || phase == storagev1alpha1.RebalanceJobPhaseFailed {
		now := metav1.Now()
		job.Status.CompletionTime = &now
	}
	if err := r.Status().Update(ctx, job); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating rebalance job status: %w", err)
	}
	if phase == storagev1alpha1.RebalanceJobPhaseRunning {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager registers the RebalanceReconciler.
func (r *RebalanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1alpha1.RebalanceJob{}).
		Complete(r)
}
