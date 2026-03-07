package mergerfs

import (
	"fmt"
	"os/exec"
)

// MountPool mounts all disk mountpoints into a single mergerfs pool path.
func MountPool(sourceGlob, pool string) error {
	cmd := exec.Command("mergerfs", sourceGlob, pool, "-o", "category.create=mfs,minfreespace=10G")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mergerfs mount failed: %w (%s)", err, string(out))
	}
	return nil
}
