package mergerfs

import "testing"

func TestBuildMountOptions(t *testing.T) {
	got := BuildMountOptions("mfs", "10G", "cache.files=off")
	want := "defaults,allow_other,category.create=mfs,minfreespace=10G,cache.files=off"
	if got != want {
		t.Fatalf("want %q got %q", want, got)
	}
}
