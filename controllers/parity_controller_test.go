package controllers

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	storagev1alpha1 "github.com/whosthisbadass/KubeNAS/api/v1alpha1"
)

func TestBuildCronJobUsesResolvedSnapraidConfigName(t *testing.T) {
	r := &ParityReconciler{}
	ps := &storagev1alpha1.ParitySchedule{ObjectMeta: metav1.ObjectMeta{Name: "ps", Namespace: "ns"}, Spec: storagev1alpha1.ParityScheduleSpec{ArrayRef: "array-a"}}

	cj := r.buildCronJob(ps, "job", "0 1 * * *", "snapraid sync", "kubenas-snapraid-node-a")
	got := cj.Spec.JobTemplate.Spec.Template.Spec.Volumes[1].VolumeSource.ConfigMap.LocalObjectReference.Name
	if got != "kubenas-snapraid-node-a" {
		t.Fatalf("expected snapraid config name to be passed through, got %q", got)
	}
}
