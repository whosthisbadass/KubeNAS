package v1alpha1

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
)

func (a *Array) ValidateCreate() error                   { return a.validate() }
func (a *Array) ValidateUpdate(old runtime.Object) error { return a.validate() }
func (a *Array) ValidateDelete() error                   { return nil }

func (a *Array) validate() error {
	if len(a.Spec.ParityDisks) == 0 || len(a.Spec.DataDisks) == 0 {
		return fmt.Errorf("array must include data and parity disks")
	}
	return nil
}

func (r *Array) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(r).Complete()
}
