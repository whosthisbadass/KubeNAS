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
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	storagev1alpha1 "github.com/kubenas/kubenas/operator/api/v1alpha1"
)

type ShareReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Agent  NodeAgentClient
}

func (r *ShareReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	share := &storagev1alpha1.Share{}
	if err := r.Get(ctx, req.NamespacedName, share); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	pool := &storagev1alpha1.Pool{}
	if err := r.Get(ctx, types.NamespacedName{Name: share.Spec.PoolRef, Namespace: share.Namespace}, pool); err != nil {
		return r.setSharePhase(ctx, share, "Pending", "PoolNotFound", err.Error())
	}
	if !pool.Status.Mounted {
		return r.setSharePhase(ctx, share, "Pending", "PoolNotMounted", "Pool is not yet mounted")
	}
	effectiveSubjects, _ := r.resolveEffectiveSubjects(ctx, share)
	share.Status.EffectiveSubjects = effectiveSubjects

	var err error
	switch share.Spec.Protocol {
	case storagev1alpha1.ShareProtocolSMB:
		err = r.reconcileSMBShare(ctx, r.Log, share, pool)
	case storagev1alpha1.ShareProtocolNFS:
		err = r.reconcileNFSShare(ctx, r.Log, share, pool)
	}
	if err != nil {
		return r.setSharePhase(ctx, share, "Degraded", "DataPlaneError", err.Error())
	}
	share.Status.Published = true
	return r.setSharePhase(ctx, share, "Ready", "Published", fmt.Sprintf("Share published via %s", share.Spec.Protocol))
}

func (r *ShareReconciler) reconcileSMBShare(ctx context.Context, log logr.Logger, share *storagev1alpha1.Share, pool *storagev1alpha1.Pool) error {
	sharePath := pool.Spec.MountPoint + share.Spec.Path
	cmName := fmt.Sprintf("kubenas-smb-%s", share.Name)
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: share.Namespace}, Data: map[string]string{"smb.conf": r.renderSMBConfig(share, sharePath)}}
	_ = controllerutil.SetControllerReference(share, cm, r.Scheme)
	if err := upsertObject(ctx, r.Client, cm); err != nil {
		return err
	}

	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: share.Namespace, Labels: map[string]string{"app": cmName}},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "samba", Image: "ghcr.io/servercontainers/samba", Ports: []corev1.ContainerPort{{ContainerPort: 445}}, LivenessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt(445)}}, InitialDelaySeconds: 10}, ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt(445)}}, InitialDelaySeconds: 5}, VolumeMounts: []corev1.VolumeMount{{Name: "pool", MountPath: "/share"}, {Name: "config", MountPath: "/etc/samba/smb.conf", SubPath: "smb.conf"}}}}, Volumes: []corev1.Volume{{Name: "pool", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: sharePath}}}, {Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: cmName}}}}}}}
	_ = controllerutil.SetControllerReference(share, pod, r.Scheme)
	if err := upsertObject(ctx, r.Client, pod); err != nil {
		return err
	}

	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: cmName + "-svc", Namespace: share.Namespace}, Spec: corev1.ServiceSpec{Selector: map[string]string{"app": cmName}, Ports: []corev1.ServicePort{{Name: "smb", Port: 445, TargetPort: intstr.FromInt(445)}}}}
	_ = controllerutil.SetControllerReference(share, svc, r.Scheme)
	if err := upsertObject(ctx, r.Client, svc); err != nil {
		return err
	}
	share.Status.PodName = pod.Name
	share.Status.ServiceName = svc.Name
	return nil
}

func (r *ShareReconciler) reconcileNFSShare(ctx context.Context, log logr.Logger, share *storagev1alpha1.Share, pool *storagev1alpha1.Pool) error {
	sharePath := pool.Spec.MountPoint + share.Spec.Path
	cmName := fmt.Sprintf("kubenas-nfs-%s", share.Name)
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: share.Namespace}, Data: map[string]string{"exports": r.renderNFSExports(share, sharePath)}}
	_ = controllerutil.SetControllerReference(share, cm, r.Scheme)
	if err := upsertObject(ctx, r.Client, cm); err != nil {
		return err
	}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: share.Namespace, Labels: map[string]string{"app": cmName}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "nfs", Image: "ghcr.io/nfs-ganesha/nfs-ganesha", Ports: []corev1.ContainerPort{{ContainerPort: 2049}}, LivenessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt(2049)}}, InitialDelaySeconds: 10}, ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt(2049)}}, InitialDelaySeconds: 5}, VolumeMounts: []corev1.VolumeMount{{Name: "pool", MountPath: "/export"}}}}, Volumes: []corev1.Volume{{Name: "pool", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: sharePath}}}}}}
	_ = controllerutil.SetControllerReference(share, pod, r.Scheme)
	if err := upsertObject(ctx, r.Client, pod); err != nil {
		return err
	}
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: cmName + "-svc", Namespace: share.Namespace}, Spec: corev1.ServiceSpec{Selector: map[string]string{"app": cmName}, Ports: []corev1.ServicePort{{Name: "nfs", Port: 2049, TargetPort: intstr.FromInt(2049)}}}}
	_ = controllerutil.SetControllerReference(share, svc, r.Scheme)
	if err := upsertObject(ctx, r.Client, svc); err != nil {
		return err
	}
	share.Status.PodName = pod.Name
	share.Status.ServiceName = svc.Name
	return nil
}

func upsertObject(ctx context.Context, c client.Client, obj client.Object) error {
	key := types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}
	existing := obj.DeepCopyObject().(client.Object)
	if err := c.Get(ctx, key, existing); err != nil {
		if errors.IsNotFound(err) {
			return c.Create(ctx, obj)
		}
		return err
	}
	obj.SetResourceVersion(existing.GetResourceVersion())
	return c.Update(ctx, obj)
}

func (r *ShareReconciler) renderSMBConfig(share *storagev1alpha1.Share, sharePath string) string {
	return fmt.Sprintf("[global]\nworkgroup = WORKGROUP\n[%s]\npath = %s\n", share.Name, sharePath)
}
func (r *ShareReconciler) renderNFSExports(share *storagev1alpha1.Share, sharePath string) string {
	return fmt.Sprintf("%s *(rw,sync,no_subtree_check)\n", sharePath)
}
func (r *ShareReconciler) resolveEffectiveSubjects(ctx context.Context, share *storagev1alpha1.Share) ([]string, error) {
	return nil, nil
}

func (r *ShareReconciler) setSharePhase(ctx context.Context, share *storagev1alpha1.Share, phase, reason, msg string) (ctrl.Result, error) {
	share.Status.Phase = phase
	setCondition(&share.Status.Conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionTrue, Reason: reason, Message: msg, LastTransitionTime: metav1.Now()})
	if phase != "Ready" {
		share.Status.Conditions[len(share.Status.Conditions)-1].Status = metav1.ConditionFalse
		share.Status.Published = false
	}
	if err := r.Status().Update(ctx, share); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *ShareReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&storagev1alpha1.Share{}).Complete(r)
}
