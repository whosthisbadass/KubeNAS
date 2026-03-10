package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1alpha1 "github.com/kubenas/kubenas/operator/api/v1alpha1"
)

type CachePoolReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *CachePoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cp := &storagev1alpha1.CachePool{}
	if err := r.Get(ctx, req.NamespacedName, cp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	cp.Status.Phase = "NotImplemented"
	if err := r.Status().Update(ctx, cp); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating cachepool status: %w", err)
	}
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *CachePoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&storagev1alpha1.CachePool{}).Complete(r)
}

func setSimpleCondition(conditions *[]metav1.Condition, t, reason, msg string, status metav1.ConditionStatus) {
	setCondition(conditions, metav1.Condition{Type: t, Status: status, Reason: reason, Message: msg, LastTransitionTime: metav1.Now()})
}
