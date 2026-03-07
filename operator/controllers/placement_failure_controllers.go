package controllers

import (
	"context"
	"fmt"
	"sort"
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

// ─────────────────────────────────────────
// Placement Scheduler
// ─────────────────────────────────────────

// DiskScore holds a candidate disk with its computed placement score.
type DiskScore struct {
	Disk  *storagev1alpha1.Disk
	Score float64
}

// PlacementScheduler selects target disks using placement policies.
type PlacementScheduler struct{}

// SelectDisk picks the optimal disk from candidates based on the given policy.
// Returns an error if no eligible disk is found.
func (ps *PlacementScheduler) SelectDisk(policy *storagev1alpha1.PlacementPolicy, candidates []*storagev1alpha1.Disk) (*storagev1alpha1.Disk, error) {
	eligible := ps.filterEligible(policy, candidates)
	if len(eligible) == 0 {
		return nil, fmt.Errorf("no eligible disks meet policy constraints")
	}

	switch policy.Spec.Strategy {
	case storagev1alpha1.PlacementStrategyBalanced:
		return ps.selectBalanced(policy, eligible)
	case storagev1alpha1.PlacementStrategyLeastUsed:
		return ps.selectLeastUsed(eligible)
	case storagev1alpha1.PlacementStrategyFillFirst:
		return ps.selectFillFirst(eligible)
	case storagev1alpha1.PlacementStrategyTiered:
		return ps.selectTiered(eligible)
	default:
		return ps.selectBalanced(policy, eligible)
	}
}

// filterEligible removes disks that are degraded, full, or below min free space.
func (ps *PlacementScheduler) filterEligible(policy *storagev1alpha1.PlacementPolicy, disks []*storagev1alpha1.Disk) []*storagev1alpha1.Disk {
	var eligible []*storagev1alpha1.Disk
	for _, d := range disks {
		if d.Status.Phase != storagev1alpha1.DiskPhaseReady {
			continue
		}
		if d.Status.CapacityBytes == 0 {
			continue
		}
		// Minimum free space check (simplified — full production version parses k8s resource quantities).
		freeRatio := float64(d.Status.AvailableBytes) / float64(d.Status.CapacityBytes)
		if freeRatio < 0.05 { // Always require at least 5% free
			continue
		}
		eligible = append(eligible, d)
	}
	return eligible
}

// selectBalanced uses the weighted scoring formula:
// score = (free_space_ratio * wFree) + (inverse_load * wLoad) + (health_score * wHealth)
func (ps *PlacementScheduler) selectBalanced(policy *storagev1alpha1.PlacementPolicy, disks []*storagev1alpha1.Disk) (*storagev1alpha1.Disk, error) {
	wFree := 0.7
	wLoad := 0.2
	wHealth := 0.1

	if policy.Spec.Weights != nil {
		wFree = policy.Spec.Weights.FreeSpace
		wLoad = policy.Spec.Weights.Load
		wHealth = policy.Spec.Weights.Health
	}

	var scored []DiskScore
	for _, d := range disks {
		freeRatio := 0.0
		if d.Status.CapacityBytes > 0 {
			freeRatio = float64(d.Status.AvailableBytes) / float64(d.Status.CapacityBytes)
		}
		// Inverse load is approximated as free ratio (full impl would use I/O counters from agent).
		inverseLoad := freeRatio
		healthScore := d.Status.HealthScore

		score := (freeRatio * wFree) + (inverseLoad * wLoad) + (healthScore * wHealth)
		scored = append(scored, DiskScore{Disk: d, Score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	return scored[0].Disk, nil
}

func (ps *PlacementScheduler) selectLeastUsed(disks []*storagev1alpha1.Disk) (*storagev1alpha1.Disk, error) {
	var best *storagev1alpha1.Disk
	for _, d := range disks {
		if best == nil || d.Status.AvailableBytes > best.Status.AvailableBytes {
			best = d
		}
	}
	return best, nil
}

func (ps *PlacementScheduler) selectFillFirst(disks []*storagev1alpha1.Disk) (*storagev1alpha1.Disk, error) {
	var best *storagev1alpha1.Disk
	for _, d := range disks {
		usedBytes := d.Status.CapacityBytes - d.Status.AvailableBytes
		if best == nil {
			best = d
			continue
		}
		bestUsed := best.Status.CapacityBytes - best.Status.AvailableBytes
		if usedBytes > bestUsed {
			best = d
		}
	}
	return best, nil
}

func (ps *PlacementScheduler) selectTiered(disks []*storagev1alpha1.Disk) (*storagev1alpha1.Disk, error) {
	// Prefer non-rotational (SSD/NVMe) disks first, then fall back to HDD.
	for _, d := range disks {
		if !d.Status.Rotational {
			return d, nil
		}
	}
	return ps.selectLeastUsed(disks)
}

// ─────────────────────────────────────────
// FailureController
// ─────────────────────────────────────────

// FailureReconciler watches Disk CRs and creates DiskFailure events on degradation.
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=diskfailures,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=diskfailures/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=disks,verbs=get;list;watch
type FailureReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile watches Disk CRs and opens/closes DiskFailure records.
func (r *FailureReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("disk", req.NamespacedName)

	disk := &storagev1alpha1.Disk{}
	if err := r.Get(ctx, req.NamespacedName, disk); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Only react to degraded/failed disks.
	if disk.Status.Phase != storagev1alpha1.DiskPhaseDegraded && disk.Status.Phase != storagev1alpha1.DiskPhaseFailed {
		// Resolve any open DiskFailure if disk is healthy.
		return r.resolveOpenFailures(ctx, log, disk)
	}

	// Determine severity and reason.
	severity := storagev1alpha1.DiskFailureSeverityWarning
	reason := "DISK_DEGRADED"
	action := "Monitor disk health and prepare for replacement"

	if disk.Status.Phase == storagev1alpha1.DiskPhaseFailed {
		severity = storagev1alpha1.DiskFailureSeverityCritical
		reason = "SMART_PREFAIL"
		if disk.Status.HealthScore < 0.1 {
			reason = "SMART_FAILURE"
		}
		action = "Replace disk immediately and initiate rebuild from parity"
	}

	// Check if a DiskFailure already exists for this disk.
	failureName := fmt.Sprintf("%s-failure", disk.Name)
	existing := &storagev1alpha1.DiskFailure{}
	err := r.Get(ctx, types.NamespacedName{Name: failureName, Namespace: "kubenas-system"}, existing)
	if errors.IsNotFound(err) {
		log.Info("opening DiskFailure", "disk", disk.Name, "severity", severity)
		failure := &storagev1alpha1.DiskFailure{
			ObjectMeta: metav1.ObjectMeta{
				Name:      failureName,
				Namespace: "kubenas-system",
			},
			Spec: storagev1alpha1.DiskFailureSpec{
				DiskRef:           disk.Name,
				Severity:          severity,
				Reason:            reason,
				RecommendedAction: action,
			},
		}
		if createErr := r.Create(ctx, failure); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("creating disk failure: %w", createErr)
		}
	} else if err == nil {
		// Update severity if it escalated.
		if existing.Spec.Severity != severity {
			existing.Spec.Severity = severity
			existing.Spec.Reason = reason
			if updateErr := r.Update(ctx, existing); updateErr != nil {
				return ctrl.Result{}, fmt.Errorf("updating disk failure: %w", updateErr)
			}
		}
	}

	return ctrl.Result{RequeueAfter: 2 * time.Minute}, nil
}

func (r *FailureReconciler) resolveOpenFailures(ctx context.Context, log logr.Logger, disk *storagev1alpha1.Disk) (ctrl.Result, error) {
	failureName := fmt.Sprintf("%s-failure", disk.Name)
	existing := &storagev1alpha1.DiskFailure{}
	if err := r.Get(ctx, types.NamespacedName{Name: failureName, Namespace: "kubenas-system"}, existing); err != nil {
		return ctrl.Result{}, nil // No open failure, nothing to do.
	}

	if existing.Status.Phase != storagev1alpha1.DiskFailurePhaseResolved {
		log.Info("resolving DiskFailure", "failure", failureName)
		now := metav1.Now()
		existing.Status.Phase = storagev1alpha1.DiskFailurePhaseResolved
		existing.Status.ResolvedAt = &now
		existing.Status.Message = fmt.Sprintf("Disk %s recovered to phase %s", disk.Name, disk.Status.Phase)
		if err := r.Status().Update(ctx, existing); err != nil {
			return ctrl.Result{}, fmt.Errorf("resolving disk failure: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager registers the FailureReconciler — watches Disk CRs.
func (r *FailureReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1alpha1.Disk{}).
		Complete(r)
}
