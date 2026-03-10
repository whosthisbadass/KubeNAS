package controllers

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1alpha1 "github.com/kubenas/kubenas/operator/api/v1alpha1"
)

type PlacementPolicyReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *PlacementPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policy := &storagev1alpha1.PlacementPolicy{}
	if err := r.Get(ctx, req.NamespacedName, policy); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	b, _ := json.Marshal(policy.Spec)
	cm := &corev1.ConfigMap{}
	name := "kubenas-placement-policy"
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: req.Namespace}, cm)
	if err != nil {
		cm.Name = name
		cm.Namespace = req.Namespace
		cm.Data = map[string]string{"policy": string(b), "generation": "1"}
		return ctrl.Result{}, r.Create(ctx, cm)
	}
	cm.Data["policy"] = string(b)
	return ctrl.Result{}, r.Update(ctx, cm)
}

func (r *PlacementPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&storagev1alpha1.PlacementPolicy{}).Complete(r)
}
