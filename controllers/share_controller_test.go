package controllers

import (
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestIsImmutablePodUpdateError(t *testing.T) {
	err := apierrors.NewInvalid(
		schema.GroupKind{Group: "", Kind: "Pod"},
		"x",
		field.ErrorList{field.Forbidden(field.NewPath("spec"), "field is immutable")},
	)
	if !isImmutablePodUpdateError(err) {
		t.Fatal("expected immutable pod error to be detected")
	}
}
