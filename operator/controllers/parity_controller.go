package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1alpha1 "github.com/kubenas/kubenas/operator/api/v1alpha1"
)

// ParityReconciler reconciles parity schedules and sync/check/scrub jobs.
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=parityschedules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=parityschedules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
type ParityReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile implements the reconciliation loop for a ParitySchedule CR.
func (r *ParityReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("parityschedule", req.NamespacedName)

	ps := &storagev1alpha1.ParitySchedule{}
	if err := r.Get(ctx, req.NamespacedName, ps); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	snapraidConfigName, err := r.resolveSnapraidConfigName(ctx, ps)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("resolving snapraid config for array %q: %w", ps.Spec.ArrayRef, err)
	}

	// Ensure a CronJob exists for each parity operation type.
	for _, op := range []struct {
		Name     string
		Schedule string
		Command  string
	}{
		{Name: "sync", Schedule: ps.Spec.SyncCron, Command: "snapraid sync"},
		{Name: "check", Schedule: ps.Spec.CheckCron, Command: "snapraid check"},
		{Name: "scrub", Schedule: ps.Spec.ScrubCron, Command: fmt.Sprintf("snapraid scrub -p %d", r.scrubPct(ps))},
	} {
		if err := r.ensureCronJob(ctx, log, ps, snapraidConfigName, op.Name, op.Schedule, op.Command); err != nil {
			return ctrl.Result{}, fmt.Errorf("ensuring %s cronjob: %w", op.Name, err)
		}
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *ParityReconciler) resolveSnapraidConfigName(ctx context.Context, ps *storagev1alpha1.ParitySchedule) (string, error) {
	array := &storagev1alpha1.Array{}
	if err := r.Get(ctx, types.NamespacedName{Name: ps.Spec.ArrayRef, Namespace: ps.Namespace}, array); err != nil {
		return "", err
	}
	if len(array.Spec.DataDisks) == 0 {
		return "", fmt.Errorf("array has no data disks")
	}

	primaryDisk := &storagev1alpha1.Disk{}
	if err := r.Get(ctx, types.NamespacedName{Name: array.Spec.DataDisks[0]}, primaryDisk); err != nil {
		return "", err
	}
	if primaryDisk.Spec.NodeName == "" {
		return "", fmt.Errorf("primary data disk %q has empty nodeName", primaryDisk.Name)
	}

	return fmt.Sprintf("kubenas-snapraid-%s", primaryDisk.Spec.NodeName), nil
}

// ensureCronJob creates or updates a SnapRAID CronJob for a given operation.
func (r *ParityReconciler) ensureCronJob(ctx context.Context, log logr.Logger, ps *storagev1alpha1.ParitySchedule, snapraidConfigName, opName, schedule, command string) error {
	cronJobName := fmt.Sprintf("kubenas-parity-%s-%s", ps.Name, opName)

	desired := r.buildCronJob(ps, cronJobName, schedule, command, snapraidConfigName)

	existing := &batchv1.CronJob{}
	err := r.Get(ctx, types.NamespacedName{Name: cronJobName, Namespace: ps.Namespace}, existing)
	if errors.IsNotFound(err) {
		log.Info("creating parity CronJob", "name", cronJobName, "schedule", schedule)
		return r.Create(ctx, desired)
	} else if err != nil {
		return err
	}

	// Update schedule if it changed.
	existing.Spec.Schedule = schedule
	existing.Spec.JobTemplate = desired.Spec.JobTemplate
	return r.Update(ctx, existing)
}

// buildCronJob constructs a CronJob manifest for a SnapRAID operation.
func (r *ParityReconciler) buildCronJob(ps *storagev1alpha1.ParitySchedule, name, schedule, command, snapraidConfigName string) *batchv1.CronJob {
	backoffLimit := int32(2)
	successLimit := int32(3)
	failedLimit := int32(3)

	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ps.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "kubenas-operator",
				"kubenas.io/array":             ps.Spec.ArrayRef,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         storagev1alpha1.GroupVersion.String(),
					Kind:               "ParitySchedule",
					Name:               ps.Name,
					UID:                ps.UID,
					Controller:         boolPtr(true),
					BlockOwnerDeletion: boolPtr(true),
				},
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule:                   schedule,
			ConcurrencyPolicy:          batchv1.ForbidConcurrent,
			SuccessfulJobsHistoryLimit: &successLimit,
			FailedJobsHistoryLimit:     &failedLimit,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					BackoffLimit: &backoffLimit,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:    "snapraid",
									Image:   "ghcr.io/kubenas/snapraid-runner:latest",
									Command: []string{"/bin/sh", "-c", command},
									SecurityContext: &corev1.SecurityContext{
										Privileged: boolPtr(true),
									},
									VolumeMounts: []corev1.VolumeMount{
										{Name: "host-mnt", MountPath: "/mnt"},
										{Name: "snapraid-conf", MountPath: "/etc/snapraid.conf", SubPath: "snapraid.conf"},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "host-mnt",
									VolumeSource: corev1.VolumeSource{
										HostPath: &corev1.HostPathVolumeSource{Path: "/mnt"},
									},
								},
								{
									Name: "snapraid-conf",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: snapraidConfigName,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *ParityReconciler) scrubPct(ps *storagev1alpha1.ParitySchedule) int {
	if ps.Spec.ScrubPercentage > 0 {
		return ps.Spec.ScrubPercentage
	}
	return 100
}

// SetupWithManager registers the ParityReconciler.
func (r *ParityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1alpha1.ParitySchedule{}).
		Owns(&batchv1.CronJob{}).
		Complete(r)
}

func boolPtr(b bool) *bool { return &b }
