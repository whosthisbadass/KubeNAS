package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1alpha1 "github.com/kubenas/kubenas/operator/api/v1alpha1"
)

// ShareReconciler reconciles Share resources and materializes SMB/NFS exports.
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=shares,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=shares/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=storage.kubenas.io,resources=pools,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods;services;configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
type ShareReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Agent  NodeAgentClient
}

// Reconcile implements the reconciliation loop for a Share CR.
func (r *ShareReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("share", req.NamespacedName)

	share := &storagev1alpha1.Share{}
	if err := r.Get(ctx, req.NamespacedName, share); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Resolve backing pool.
	pool := &storagev1alpha1.Pool{}
	if err := r.Get(ctx, types.NamespacedName{Name: share.Spec.PoolRef, Namespace: share.Namespace}, pool); err != nil {
		return r.setSharePhase(ctx, share, "Pending", "PoolNotFound", fmt.Sprintf("Pool %q not found", share.Spec.PoolRef))
	}

	if !pool.Status.Mounted {
		return r.setSharePhase(ctx, share, "Pending", "PoolNotMounted", "Pool is not yet mounted")
	}

	// Resolve effective permissions from Kubernetes RBAC subjects.
	effectiveSubjects, err := r.resolveEffectiveSubjects(ctx, share)
	if err != nil {
		log.Error(err, "subject resolution failed")
		return r.setSharePhase(ctx, share, "Degraded", "SubjectResolutionError", err.Error())
	}

	share.Status.EffectiveSubjects = effectiveSubjects

	// Reconcile the share pod and config based on protocol.
	switch share.Spec.Protocol {
	case storagev1alpha1.ShareProtocolSMB:
		if err := r.reconcileSMBShare(ctx, log, share, pool); err != nil {
			return r.setSharePhase(ctx, share, "Degraded", "SMBReconcileError", err.Error())
		}
	case storagev1alpha1.ShareProtocolNFS:
		if err := r.reconcileNFSShare(ctx, log, share, pool); err != nil {
			return r.setSharePhase(ctx, share, "Degraded", "NFSReconcileError", err.Error())
		}
	}

	share.Status.Published = true
	return r.setSharePhase(ctx, share, "Ready", "Published", fmt.Sprintf("Share published via %s", share.Spec.Protocol))
}

// reconcileSMBShare creates/updates the Samba ConfigMap and Pod for an SMB share.
func (r *ShareReconciler) reconcileSMBShare(ctx context.Context, log logr.Logger, share *storagev1alpha1.Share, pool *storagev1alpha1.Pool) error {
	sharePath := pool.Spec.MountPoint + share.Spec.Path
	smbConf := r.renderSMBConfig(share, sharePath)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("kubenas-smb-%s", share.Name),
			Namespace: share.Namespace,
		},
		Data: map[string]string{
			"smb.conf": smbConf,
		},
	}

	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, existing)
	if errors.IsNotFound(err) {
		log.Info("creating SMB ConfigMap", "name", cm.Name)
		if createErr := r.Create(ctx, cm); createErr != nil {
			return fmt.Errorf("creating smb configmap: %w", createErr)
		}
	} else if err == nil {
		existing.Data = cm.Data
		if updateErr := r.Update(ctx, existing); updateErr != nil {
			return fmt.Errorf("updating smb configmap: %w", updateErr)
		}
	}

	share.Status.PodName = fmt.Sprintf("kubenas-smb-%s", share.Name)
	share.Status.ServiceName = fmt.Sprintf("kubenas-smb-%s-svc", share.Name)
	return nil
}

// reconcileNFSShare creates/updates the NFS export config for an NFS share.
func (r *ShareReconciler) reconcileNFSShare(ctx context.Context, log logr.Logger, share *storagev1alpha1.Share, pool *storagev1alpha1.Pool) error {
	sharePath := pool.Spec.MountPoint + share.Spec.Path
	exportsConf := r.renderNFSExports(share, sharePath)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("kubenas-nfs-%s", share.Name),
			Namespace: share.Namespace,
		},
		Data: map[string]string{
			"exports": exportsConf,
		},
	}

	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, existing)
	if errors.IsNotFound(err) {
		log.Info("creating NFS exports ConfigMap", "name", cm.Name)
		if createErr := r.Create(ctx, cm); createErr != nil {
			return fmt.Errorf("creating nfs configmap: %w", createErr)
		}
	} else if err == nil {
		existing.Data = cm.Data
		if updateErr := r.Update(ctx, existing); updateErr != nil {
			return fmt.Errorf("updating nfs configmap: %w", updateErr)
		}
	}

	share.Status.PodName = fmt.Sprintf("kubenas-nfs-%s", share.Name)
	return nil
}

// renderSMBConfig generates a smb.conf stanza for the share.
func (r *ShareReconciler) renderSMBConfig(share *storagev1alpha1.Share, sharePath string) string {
	browseable := "yes"
	readOnly := "no"
	if share.Spec.SMB != nil {
		if !share.Spec.SMB.Browseable {
			browseable = "no"
		}
		if share.Spec.SMB.ReadOnly {
			readOnly = "yes"
		}
	}

	conf := fmt.Sprintf(`[global]
   workgroup = WORKGROUP
   security = user
   map to guest = bad user

[%s]
   path = %s
   browseable = %s
   read only = %s
   force create mode = 0660
   force directory mode = 2770
`, share.Name, sharePath, browseable, readOnly)

	// Add valid users based on authz subjects.
	if share.Spec.Authz != nil {
		for _, s := range share.Spec.Authz.Subjects {
			if s.Kind == storagev1alpha1.ShareSubjectKindUser {
				conf += fmt.Sprintf("   valid users = @%s\n", s.Name)
			}
		}
	}

	return conf
}

// renderNFSExports generates an /etc/exports entry for the share.
func (r *ShareReconciler) renderNFSExports(share *storagev1alpha1.Share, sharePath string) string {
	opts := "rw,sync,no_subtree_check"
	if share.Spec.NFS != nil && share.Spec.NFS.ReadOnly {
		opts = "ro,sync,no_subtree_check"
	}

	squash := "root_squash"
	if share.Spec.NFS != nil && share.Spec.NFS.SquashMode != "" {
		squash = share.Spec.NFS.SquashMode
	}

	cidrs := []string{"*"}
	if share.Spec.NFS != nil && len(share.Spec.NFS.AllowedCIDRs) > 0 {
		cidrs = share.Spec.NFS.AllowedCIDRs
	}

	exports := ""
	for _, cidr := range cidrs {
		exports += fmt.Sprintf("%s %s(%s,%s)\n", sharePath, cidr, opts, squash)
	}
	return exports
}

// resolveEffectiveSubjects resolves the active user/group list from authz spec.
func (r *ShareReconciler) resolveEffectiveSubjects(ctx context.Context, share *storagev1alpha1.Share) ([]string, error) {
	if share.Spec.Authz == nil {
		return nil, nil
	}
	var subjects []string
	for _, s := range share.Spec.Authz.Subjects {
		subjects = append(subjects, fmt.Sprintf("%s:%s", s.Kind, s.Name))
	}
	return subjects, nil
}

func (r *ShareReconciler) setSharePhase(ctx context.Context, share *storagev1alpha1.Share, phase, reason, msg string) (ctrl.Result, error) {
	share.Status.Phase = phase
	setCondition(&share.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	})
	if phase != "Ready" {
		share.Status.Conditions[len(share.Status.Conditions)-1].Status = metav1.ConditionFalse
		share.Status.Published = false
	}
	if err := r.Status().Update(ctx, share); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating share status: %w", err)
	}
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager registers the ShareReconciler.
func (r *ShareReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1alpha1.Share{}).
		Complete(r)
}
