package mount

import "github.com/kubenas/kubenas/node-agent/pkg/disk"

func Device(devicePath, mountPoint, fs string) error {
	return disk.MountDevice(devicePath, mountPoint, fs)
}

func Unmount(mountPoint string) error {
	return disk.UnmountDevice(mountPoint)
}

func MergerFS(branches []string, mountPoint, categoryCreate, minFreeSpace, extra string) error {
	return disk.MountMergerFS(branches, mountPoint, categoryCreate, minFreeSpace, extra)
}
