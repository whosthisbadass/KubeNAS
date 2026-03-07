package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	storagev1 "github.com/kubenas/kubenas/operator/api/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const reconcileAnnotation = "kubenas.io/last-reconciled"

type controllerBase struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (b *controllerBase) reconcileObject(ctx context.Context, obj client.Object, kind string) (ctrl.Result, error) {
	if obj.GetAnnotations() == nil {
		obj.SetAnnotations(map[string]string{})
	}
	ann := obj.GetAnnotations()
	now := time.Now().UTC().Format(time.RFC3339)
	if ann[reconcileAnnotation] == now {
		return ctrl.Result{}, nil
	}
	ann[reconcileAnnotation] = now
	obj.SetAnnotations(ann)
	if err := b.Update(ctx, obj); err != nil {
		return ctrl.Result{}, err
	}
	b.Log.Info("reconciled resource", "kind", kind, "name", obj.GetName())
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

type DiskController struct{ controllerBase }

type DiskClaimController struct{ controllerBase }

type UnassignedDiskController struct{ controllerBase }

type PoolController struct{ controllerBase }

type FilesystemController struct{ controllerBase }

type ShareControllerV1 struct{ controllerBase }

type VolumeController struct{ controllerBase }

type TierController struct{ controllerBase }

func NewDiskController(c client.Client, l logr.Logger, s *runtime.Scheme) *DiskController {
	return &DiskController{controllerBase: NewControllerBase(c, l, s)}
}
func NewDiskClaimController(c client.Client, l logr.Logger, s *runtime.Scheme) *DiskClaimController {
	return &DiskClaimController{controllerBase: NewControllerBase(c, l, s)}
}
func NewUnassignedDiskController(c client.Client, l logr.Logger, s *runtime.Scheme) *UnassignedDiskController {
	return &UnassignedDiskController{controllerBase: NewControllerBase(c, l, s)}
}
func NewPoolController(c client.Client, l logr.Logger, s *runtime.Scheme) *PoolController {
	return &PoolController{controllerBase: NewControllerBase(c, l, s)}
}
func NewFilesystemController(c client.Client, l logr.Logger, s *runtime.Scheme) *FilesystemController {
	return &FilesystemController{controllerBase: NewControllerBase(c, l, s)}
}
func NewShareControllerV1(c client.Client, l logr.Logger, s *runtime.Scheme) *ShareControllerV1 {
	return &ShareControllerV1{controllerBase: NewControllerBase(c, l, s)}
}
func NewVolumeController(c client.Client, l logr.Logger, s *runtime.Scheme) *VolumeController {
	return &VolumeController{controllerBase: NewControllerBase(c, l, s)}
}
func NewTierController(c client.Client, l logr.Logger, s *runtime.Scheme) *TierController {
	return &TierController{controllerBase: NewControllerBase(c, l, s)}
}

func (r *DiskController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj := &storagev1.Disk{}
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if obj.Status.State == "" {
		obj.Status.State = storagev1.DiskStateUnassigned
		obj.Status.Health = "Healthy"
		_ = r.Status().Update(ctx, obj)
	}
	return r.reconcileObject(ctx, obj, "Disk")
}

func (r *DiskClaimController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj := &storagev1.DiskClaim{}
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	return r.reconcileObject(ctx, obj, "DiskClaim")
}

func (r *UnassignedDiskController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj := &storagev1.UnassignedDisk{}
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	return r.reconcileObject(ctx, obj, "UnassignedDisk")
}

func (r *PoolController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj := &storagev1.StoragePool{}
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	return r.reconcileObject(ctx, obj, "StoragePool")
}

func (r *FilesystemController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj := &storagev1.Filesystem{}
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	return r.reconcileObject(ctx, obj, "Filesystem")
}

func (r *ShareControllerV1) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj := &storagev1.Share{}
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	return r.reconcileObject(ctx, obj, "Share")
}

func (r *VolumeController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj := &storagev1.Volume{}
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	return r.reconcileObject(ctx, obj, "Volume")
}

func (r *TierController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj := &storagev1.TierPolicy{}
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	return r.reconcileObject(ctx, obj, "TierPolicy")
}

func (r *DiskController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&storagev1.Disk{}).Named("disk-controller-v1").Complete(r)
}
func (r *DiskClaimController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&storagev1.DiskClaim{}).Named("diskclaim-controller-v1").Complete(r)
}
func (r *UnassignedDiskController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&storagev1.UnassignedDisk{}).Named("unassigneddisk-controller-v1").Complete(r)
}
func (r *PoolController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&storagev1.StoragePool{}).Named("storagepool-controller-v1").Complete(r)
}
func (r *FilesystemController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&storagev1.Filesystem{}).Named("filesystem-controller-v1").Complete(r)
}
func (r *ShareControllerV1) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&storagev1.Share{}).Named("share-controller-v1").Complete(r)
}
func (r *VolumeController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&storagev1.Volume{}).Named("volume-controller-v1").Complete(r)
}
func (r *TierController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&storagev1.TierPolicy{}).Named("tier-controller-v1").Complete(r)
}

func NewControllerBase(c client.Client, l logr.Logger, s *runtime.Scheme) controllerBase {
	return controllerBase{Client: c, Log: l, Scheme: s}
}

func RebuildDiskFlowMessage(oldState, newState storagev1.DiskLifecycleState) string {
	return fmt.Sprintf("disk transition: %s -> %s", oldState, newState)
}
