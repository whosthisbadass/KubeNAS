package v1alpha1

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (d *Disk) ValidateCreate() error                   { return d.validate() }
func (d *Disk) ValidateUpdate(old runtime.Object) error { return d.validate() }
func (d *Disk) ValidateDelete() error                   { return nil }

func (d *Disk) validate() error {
	if d.Spec.DevicePath == "" || d.Spec.MountPoint == "" || d.Spec.NodeName == "" {
		return apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "Disk"}, d.Name, nil)
	}
	if d.Spec.DevicePath == d.Spec.MountPoint {
		return fmt.Errorf("devicePath and mountPoint must differ")
	}
	return nil
}

func (r *Disk) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(r).Complete()
}
