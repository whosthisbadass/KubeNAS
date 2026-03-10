package v1alpha1

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
)

func (s *Share) ValidateCreate() error                   { return s.validate() }
func (s *Share) ValidateUpdate(old runtime.Object) error { return s.validate() }
func (s *Share) ValidateDelete() error                   { return nil }

func (s *Share) validate() error {
	if !strings.HasPrefix(s.Spec.Path, "/") {
		return fmt.Errorf("share path must be absolute")
	}
	return nil
}

func (r *Share) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(r).Complete()
}
